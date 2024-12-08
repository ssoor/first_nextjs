package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/gin-gonic/gin"
)

const (
	APIFunctionPath = "/api"
	APIFunctionHost = "api.m.jd.com"
	APIEIDKey       = "3AB9D23F7A4B3C9B"
	APIEIDTokenKey  = "3AB9D23F7A4B3CSS"
	APIReferer      = "https://paipai.m.jd.com/"
)

type UnixTime int64

func (ut UnixTime) Time() time.Time {
	return time.Unix(int64(ut/1000), int64(ut%1000))
}
func (ut UnixTime) String() string {
	return ut.Time().Format("2006-01-02T15:04:05.999Z07:00")
}
func (ut UnixTime) MarshalText() (text []byte, err error) {
	return []byte(ut.Time().Format("2006-01-02T15:04:05.999Z07:00")), nil
}

type BidderStatus int

func (bs BidderStatus) String() string {
	switch bs {
	case 1:
		return "即将开始"
	case 2:
		return "正在进行"
	case 3:
		return "已经结束"
	default:
		return "未知状态"
	}
}
func (bs BidderStatus) MarshalText() (text []byte, err error) {
	return []byte(bs.String()), nil
}

type DBDProductInfo struct {
	Id                    int          `json:"id,omitempty"`             // ID
	Status                BidderStatus `json:"status,omitempty"`         // 状态 1 即将开始；2 正在进行
	Quality               string       `json:"quality,omitempty"`        // 质量
	PrimaryPic            string       `json:"primaryPic,omitempty"`     // 封面
	ProductName           string       `json:"productName,omitempty"`    // 产品名称
	StartTime             UnixTime     `json:"startTime,omitempty"`      // 开始时间
	EndTime               UnixTime     `json:"endTime,omitempty"`        // 结束时间
	StartPrice            float64      `json:"startPrice,omitempty"`     // 起步价
	CappedPrice           float64      `json:"cappedPrice,omitempty"`    // 封顶价
	CurrentPrice          float64      `json:"currentPrice,omitempty"`   // 当前价
	CurrentBidder         string       `json:"currentBidder,omitempty"`  // 当前竞拍打码账号
	CurrentBidderNickName string       `json:"bidderNickName,omitempty"` // 当前竞拍打码昵称
	SpectatorCount        int          `json:"spectatorCount,omitempty"` // 围观数
}

type DBDProductBidder struct {
	Id             int          `json:"auctionId,omitempty"`      // ID
	Status         BidderStatus `json:"status,omitempty"`         // 状态 3 结束
	CurrentPrice   float64      `json:"currentPrice,omitempty"`   // 当前价格
	CurrentBidder  string       `json:"currentBidder,omitempty"`  // 当前竞拍打码账号
	BidderNickName string       `json:"bidderNickName,omitempty"` // 当前竞拍打码昵称
	ActualEndTime  UnixTime     `json:"actualEndTime,omitempty"`  // 实际结束时间
	SpectatorCount int          `json:"spectatorCount,omitempty"` // 观众数
	// DelayCount        int64        `json:"delayCount,omitempty"`        // 不知道是啥
	// VirtualDelayCount UnixTime     `json:"virtualDelayCount,omitempty"` // 不知道是啥
}

type DBDAuctionDetail struct {
	AuctionInfo struct {
		Id             int      `json:"id,omitempty"`             // ID
		ProductName    string   `json:"productName,omitempty"`    // 产品名称
		CurrentPrice   float64  `json:"currentPrice,omitempty"`   // 当前价格
		CurrentBidder  string   `json:"bidder,omitempty"`         // 当前竞拍打码账号
		BidderNickName string   `json:"bidderNickName,omitempty"` // 当前竞拍打码昵称
		ActualEndTime  UnixTime `json:"actualEndTime,omitempty"`  // 实际结束时间
		Creater        string   `json:"creater,omitempty"`        // 竞拍创建人
		Created        UnixTime `json:"created,omitempty"`        // 竞拍创建时间
		Modified       UnixTime `json:"modified,omitempty"`       // 竞拍修改时间
	} `json:"auctionInfo,omitempty"` // 拍卖信息
	FreightArea     string `json:"freightArea,omitempty"`     // 收货地址内部编码
	FreightAreaText string `json:"freightAreaText,omitempty"` // 收货地址

	HistoryPriceAve float64 `json:"historyPriceAve,omitempty"` // 平均成交价格 - 原始返回不存在，需要自己算
	HistoryPriceMax float64 `json:"historyPriceMax,omitempty"` // 最高成交价格
	HistoryPriceMin float64 `json:"historyPriceMin,omitempty"` // 最低成交价格
	HistoryRecord   []struct {
		UserNickname string   `json:"userNickname,omitempty"` // 用户昵称
		UserImage    string   `json:"userImage,omitempty"`    // 用户头像
		EndTime      UnixTime `json:"endTime,omitempty"`      // 拍卖结束时间
		OfferPrice   float64  `json:"offerPrice,omitempty"`   // 最终拍卖价格
	} `json:"historyRecord,omitempty"` // 历史成交记录

	IsAreaLimit bool `json:"isAreaLimit,omitempty"` // 是否存在区域限制
	// AreaLimitDetail string `json:"areaLimitDetail,omitempty"` // 区域限制详情，不知道内容格式

	SpectatorCount int `json:"spectatorCount,omitempty"` // 围观数

	HasStock        bool   `json:"hasStock,omitempty"`        // 不知道
	HasAuctionStock bool   `json:"hasAuctionStock,omitempty"` // 不知道
	StockCheckArea  string `json:"stockCheckArea,omitempty"`  // 不知道干啥用的，值好像是这个 freightArea
}

type dbd struct {
	client  *http.Client
	cookies *cookiejar.Jar
}

func NewDBD() (*dbd, error) {
	ckj, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	c := &http.Client{
		Jar: ckj,
	}

	return &dbd{
		client:  c,
		cookies: ckj,
	}, nil
}

func (m dbd) request(method string, path string, params url.Values, headers http.Header, body io.Reader) (*http.Response, error) {
	u, err := url.Parse(fmt.Sprintf("https://%s", APIFunctionHost))
	u.Path = path
	u.RawQuery = params.Encode()

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header = headers

	return m.client.Do(req)
}

type JSToken struct {
	Eid   string      `json:"eid"`
	Token string      `json:"token"`
	Ds    int         `json:"ds"`
	GiaD  int         `json:"gia_d"`
	DeMap interface{} `json:"deMap"`
}

func (m dbd) getApiEIDToken() (*JSToken, error) {
	vm := goja.New()
	jsScript2, _ := os.ReadFile("./getJsToken.js")
	_, err := vm.RunString(string(jsScript2))
	if err != nil {
		fmt.Println("JS代码有问题！")
		return nil, err
	}

	var getJsToken func(userAgent, url, bizId, eid string) goja.Promise
	err = vm.ExportTo(vm.Get("getJsToken"), &getJsToken)
	if err != nil {
		fmt.Println("Js函数映射到 Go 函数失败！")
		return nil, err
	}

	eid := ""
	m.RangeCookie(func(name, value string) bool {
		if name == APIEIDKey {
			eid = value
			return false
		}

		return true
	})
	if eid == "" {
		return nil, fmt.Errorf("没有找到可用的 EID")
	}
	v := getJsToken(
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		"https://paipai.jd.com/auction-detail/393024207",
		"paipai_sale_pc",
		eid,
	)
	v.Result()
	v2 := v.Result().ToObject(vm)

	// fmt.Println(v2.Get("a").ToString())
	// fmt.Println(v2.Get("d").ToString())

	params := url.Values{
		"a": []string{v2.Get("a").ToString().String()},
		"d": []string{v2.Get("d").ToString().String()},
	}
	req, err := http.NewRequest("POST", "https://jra.jd.com/jsTk.do", strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}

	header := req.Header
	header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBin, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// fmt.Println((v2.Export()))
	// fmt.Println(string(bodyBin))

	respStruct := struct {
		Code    int     `json:"code"`
		Message string  `json:"msg"`
		Data    JSToken `json:"data"`
	}{}

	if err := json.Unmarshal(bodyBin, &respStruct); err != nil {
		return nil, err
	}

	if respStruct.Code != 0 {
		return nil, fmt.Errorf("resp: %v=%v", respStruct.Code, respStruct.Message)
	}

	return &respStruct.Data, nil
}

func (m dbd) callFunction(method string, functionId string, bodyParams map[string]interface{}) (reqsp *http.Response, err error) {
	t := strconv.FormatInt(time.Now().UnixMilli(), 10)

	params := url.Values{}
	params.Set("t", t)
	params.Set("appid", "paipai_h5")
	// params.Set("appid", "paipai_sale_pc")
	params.Set("functionId", functionId)

	// jsToken, err := m.getApiEIDToken()
	// if err != nil {
	// 	return nil, err
	// }
	// params.Set("x-api-eid-token", jsToken.Token)

	headers := http.Header{}
	headers.Set("Sec-Fetch-Mode", "cors")
	headers.Set("Content-Type", "application/x-www-form-urlencoded")
	headers.Set("User-Agent", "jdapp;android;12.0.2;;;M/5.0;appBuild/98787;ef/1;ep/%7B%22hdid%22%3A%22JM9F1ywUPwflvMIpYPok0tt5k9kW4ArJEU3lfLhxBqw%3D%22%2C%22ts%22%3A1685444654944%2C%22ridx%22%3A-1%2C%22cipher%22%3A%7B%22sv%22%3A%22CJC%3D%22%2C%22ad%22%3A%22CtG3YtCyDtc3EJCmC2OyYm%3D%3D%22%2C%22od%22%3A%22CzY5ZJU0CQU3C2OyEJvwYq%3D%3D%22%2C%22ov%22%3A%22CzC%3D%22%2C%22ud%22%3A%22CtG3YtCyDtc3EJCmC2OyYm%3D%3D%22%7D%2C%22ciphertype%22%3A5%2C%22version%22%3A%221.2.0%22%2C%22appname%22%3A%22com.jingdong.app.mall%22%7D;jdSupportDarkMode/0;Mozilla/5.0 (Linux; Android 13; MI 8 Build/TKQ1.220905.001; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/89.0.4389.72 MQQBrowser/6.2 TBS/046247 Mobile Safari/537.36")
	headers.Set("Referer", APIReferer)

	bodyJSON := []byte{}
	if bodyParams != nil {
		bodyJSON, err = json.Marshal(bodyParams)
		if err != nil {
			return nil, err
		}
	}

	switch method {
	case "GET":
		params.Set("body", string(bodyJSON))
	case "POST":
		bodyJSON = []byte(url.Values{"body": []string{string(bodyJSON)}}.Encode())
	}

	return m.request(method, APIFunctionPath, params, headers, bytes.NewBuffer(bodyJSON))
}
func (m dbd) callFunctionEx(method string, functionId string, bodyParams map[string]interface{}, outResp interface{}) (err error) {
	resp, err := m.callFunction(method, functionId, bodyParams)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code != 200, status = %v", resp.Status)
	}

	bodyBin, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	respStruct := struct {
		Code    interface{} `json:"code"` // 有两种返回 "1"+echo,0+message
		Echo    string      `json:"echo"`
		Message string      `json:"message"`
		Result  struct {
			Code    int         `json:"code"`
			Message string      `json:"message"`
			Data    interface{} `json:"data"`
		} `json:"result"`
	}{}
	respStruct.Result.Data = outResp

	if err := json.Unmarshal(bodyBin, &respStruct); err != nil {
		return err
	}

	if code, ok := respStruct.Code.(string); ok && code != "" {
		return fmt.Errorf("resp: %v=%v", respStruct.Code, respStruct.Echo)
	}

	if code, ok := respStruct.Code.(float64); ok && code != 0 {
		return fmt.Errorf("resp: %v=%v", respStruct.Code, respStruct.Message)
	}

	if respStruct.Result.Code != 200 {
		return fmt.Errorf("resp.result: %v=%v", respStruct.Result.Code, respStruct.Result.Message)
	}

	return nil
}

func (m dbd) RangeCookie(cb func(name, value string) bool) {
	targetURL, err := url.Parse("https://paipai.jd.com")
	if err != nil {
		panic(err)
	}

	cookies := m.cookies.Cookies(targetURL)
	for _, cookie := range cookies {
		if !cb(cookie.Name, cookie.Value) {
			break
		}
	}
}

func (m dbd) SetCookie(cookie string) {
	targetURL, err := url.Parse("https://paipai.jd.com")
	if err != nil {
		panic(err)
	}

	// shshshfpa=9349f29c-2864-d94d-fddd-397aab2be464-1703899777; shshshfpx=9349f29c-2864-d94d-fddd-397aab2be464-1703899777;

	cookies := []*http.Cookie{}
	for _, cookie := range strings.Split(cookie, ";") {
		cookie := strings.SplitN(strings.TrimSpace(cookie), "=", 2)

		cookies = append(cookies, &http.Cookie{
			Name:   cookie[0],
			Value:  cookie[1],
			Domain: ".jd.com",
			Path:   "/",
		})
	}

	m.cookies.SetCookies(targetURL, cookies)
}

func (m dbd) AuctionDetail(id int) (*DBDAuctionDetail, error) {
	body := map[string]interface{}{
		"t":             time.Now().UnixMilli(),
		"auctionId":     id,
		"dbdApiVersion": "20200623",
	}
	functionID := "paipai.auction.detail"

	data := DBDAuctionDetail{}
	if err := m.callFunctionEx("GET", functionID, body, &data); err != nil {
		return nil, err
	}

	if len(data.HistoryRecord) != 0 {
		total := float64(0.0)
		for _, v := range data.HistoryRecord {
			total += v.OfferPrice
		}

		data.HistoryPriceAve = total / float64(len(data.HistoryRecord))
	}

	return &data, nil
}

func (m dbd) ProductPrice(id int, address string, price int) (map[int]DBDProductBidder, error) {
	eid := ""
	m.RangeCookie(func(name, value string) bool {
		if name == APIEIDKey {
			eid = value
			return false
		}

		return true
	})
	if eid == "" {
		return nil, fmt.Errorf("没有找到可用的 EID")
	}

	body := map[string]interface{}{
		"price":            price,
		"auctionId":        id,
		"eid":              eid,
		"address":          address,
		"entryid":          "",
		"transformRequest": []interface{}{nil},
	}
	functionID := "paipai.auction.offerPrice"

	resp, err := m.callFunction("POST", functionID, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBin, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	respStruct := struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Result  struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"result"`
	}{}

	if err := json.Unmarshal(bodyBin, &respStruct); err != nil {
		return nil, err
	}

	if respStruct.Code != 0 {
		return nil, fmt.Errorf("resp: %v=%v", respStruct.Code, respStruct.Message)
	}
	if respStruct.Result.Code != 200 {
		return nil, fmt.Errorf("resp.result: %v=%v", respStruct.Result.Code, respStruct.Result.Message)
	}

	return nil, nil
}

func (m dbd) ProductBidder(ids ...int) (map[int]DBDProductBidder, error) {
	actionIDs := []string{}
	for _, v := range ids {
		actionIDs = append(actionIDs, strconv.Itoa(v))
	}

	body := map[string]interface{}{
		"auctionId": strings.Join(actionIDs, ","),
	}
	functionID := "paipai.auction.current_bid_info"

	data := map[string]DBDProductBidder{}
	if err := m.callFunctionEx("GET", functionID, body, &data); err != nil {
		return nil, err
	}

	bidders := map[int]DBDProductBidder{}
	for idStr, bidder := range data {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return nil, err
		}
		if bidder.BidderNickName == "***" {
			bidder.BidderNickName = ""
		}
		bidders[id] = bidder
	}

	return bidders, nil
}

func (m dbd) ProductSearch(query string, status string, page int) ([]DBDProductInfo, error) {
	body := map[string]interface{}{
		"pageNo":    page,
		"pageSize":  20,
		"key":       query,
		"status":    status,
		"mpSource":  1,
		"sourceTag": 1,
	}

	functionID := ""
	if query == "" {
		body["p"] = 2
		body["skuGroup"] = 1
		body["sort"] = "endTime_asc"
		body["isPersonalRecommend"] = 0
		body["auctionFilterTime"] = 30

		functionID = "dbd.auction.list.v2"
	} else {
		body["sort"] = "endTime_asc"
		body["specialType"] = 1

		functionID = "pp.dbd.biz.search.query"
	}

	data := struct {
		ItemList     []DBDProductInfo `json:"itemList"`
		AuctionInfos []DBDProductInfo `json:"auctionInfos"`
	}{}
	if err := m.callFunctionEx("GET", functionID, body, &data); err != nil {
		return nil, err
	}

	products := []DBDProductInfo{}
	products = append(products, data.ItemList...)
	products = append(products, data.AuctionInfos...)

	sort.Slice(products, func(i, j int) bool {
		return products[i].EndTime.Time().Before(products[j].EndTime.Time())
	})

	return products, nil
}

func main() {
	dbd, err := NewDBD()
	if err != nil {
		panic(err)
	}
	cookie, err := os.ReadFile("./cookie.txt")
	if err != nil {
		panic(err)
	}
	dbd.SetCookie(string(cookie))

	r := gin.Default()
	r.GET("/serach/*query", func(c *gin.Context) {
		query := strings.TrimSuffix(c.Params.ByName("query"), "/")
		status := c.DefaultQuery("status", "")
		detail := c.DefaultQuery("detail", "false")
		page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}

		products, err := dbd.ProductSearch(query, status, page)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}

		// if query != "" {
		// 	ids := []int{}
		// 	for _, product := range products {
		// 		ids = append(ids, product.Id)
		// 	}

		// 	bidders, err := dbd.ProductBidder(ids...)
		// 	if err != nil {
		// 		c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
		// 		return
		// 	}

		// 	for i := range products {
		// 		id := products[i].Id
		// 		bidder, exist := bidders[id]
		// 		if !exist {
		// 			continue
		// 		}

		// 		products[i].Status = bidder.Status
		// 		products[i].SpectatorCount = bidder.SpectatorCount

		// 		products[i].CurrentPrice = bidder.CurrentPrice
		// 		products[i].CurrentBidder = bidder.CurrentBidder
		// 		products[i].CurrentBidderNickName = bidder.BidderNickName
		// 	}
		// }

		infos := []map[string]interface{}{}
		mores := []map[string]interface{}{}
		for _, v := range products {

			price := map[string]interface{}{
				"curr":   v.CurrentPrice,
				"capped": v.CappedPrice,
			}
			if detail == "true" {
				time.Sleep(time.Second)

				detail, err := dbd.AuctionDetail(v.Id)
				if err != nil {
					c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
					return
				}

				price = map[string]interface{}{
					"ave":    detail.HistoryPriceAve,
					"max":    detail.HistoryPriceMax,
					"min":    detail.HistoryPriceMin,
					"curr":   detail.AuctionInfo.CurrentPrice,
					"capped": v.CappedPrice,
				}
			}

			/*
				export type Auction = {
				  id: number;
				  name: string;
				  quality: string;
				  primary_pic: string;
				  status: string;
				  capped_price: number;
				  current_price: number;
				  current_bidder: string;
				  current_bidder_nickname: string;
				  start_timestamp: number;
				  end_timestamp: number;
				};
			*/

			// imageUrl: computed(
			// 	() => (primaryPic, size) => API.image_url + size + "/" + (primaryPic.startsWith("jfs") ? primaryPic : "jfs/" + primaryPic)
			//   ),

			imageUrl := fmt.Sprintf("https://img10.360buyimg.com/%v/jfs/%v", "n4", strings.TrimPrefix(v.PrimaryPic, "jfs/"))
			infos = append(infos, map[string]interface{}{
				"id":                      v.Id,
				"name":                    v.ProductName,
				"quality":                 v.Quality,
				"primary_pic":             imageUrl,
				"status":                  v.Status,
				"capped_price":            v.CappedPrice,
				"current_price":           v.CurrentPrice,
				"current_bidder":          v.CurrentBidder,
				"current_bidder_nickname": v.CurrentBidderNickName,
				"end_timestamp":           int64(v.EndTime),
				"start_timestamp":         int64(v.EndTime),
			})

			mores = append(mores, map[string]interface{}{
				"products": v,
				"detail":   detail,
				"info": map[string]interface{}{
					"price":   price,
					"id":      v.Id,
					"name":    v.ProductName,
					"status":  v.Status,
					"quality": v.Quality,
					"time": map[string]interface{}{
						"end":    v.EndTime,
						"start":  v.StartTime,
						"remain": v.EndTime.Time().Sub(time.Now()),
					},
				},
			})
		}

		out := map[string]interface{}{
			"detail": infos,
			"mores":  mores,
		}

		outJson, _ := json.MarshalIndent(out, "", "  ")

		os.WriteFile("./output.json", outJson, 0644)

		c.JSON(http.StatusOK, out["detail"])
	})

	r.GET("/product/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Params.ByName("id"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}

		detail, err := dbd.AuctionDetail(id)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}

		c.JSON(http.StatusOK, detail)
	})

	r.GET("/didder/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Params.ByName("id"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}
		count, err := strconv.Atoi(c.DefaultQuery("count", "1"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}

		var prevBidder DBDProductBidder
		for i := 0; i < count; i++ {
			bidder, err := dbd.ProductBidder(id)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
				return
			}
			time.Sleep(100 * time.Millisecond)

			if reflect.DeepEqual(bidder[id], prevBidder) {
				fmt.Printf("%v,", i)
				continue
			}

			fmt.Println(time.Now())
			prevBidder = bidder[id]
			fmt.Printf("%v: %+v=%v\n", i, bidder[id], err)
		}

		c.JSON(http.StatusOK, prevBidder)
	})

	r.GET("/auction/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Params.ByName("id"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}
		address := c.DefaultQuery("address", "")
		price, err := strconv.Atoi(c.DefaultQuery("price", "0"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}

		if price == 0 {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(fmt.Errorf("土豪，您的出价不合法呦！")))
			return
		}

		if address == "" {
			detail, err := dbd.AuctionDetail(id)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
				return
			}
			address = detail.FreightArea
		}

		productPrice, err := dbd.ProductPrice(id, address, price)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}

		c.JSON(http.StatusOK, productPrice)
	})

	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")

	return
}
