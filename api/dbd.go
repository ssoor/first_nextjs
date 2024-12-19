package main

import (
	"bytes"
	"crypto/sha256"
	"dbd/gojs"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
)

const (
	APIFunctionPath = "/api"
	APIFunctionHost = "api.m.jd.com"
	APIEIDKey       = "3AB9D23F7A4B3C9B"
	APIEIDTokenKey  = "3AB9D23F7A4B3CSS"
	APIReferer      = "https://paipai.m.jd.com/"
)

type dbdClient struct {
	client  *http.Client
	cookies *cookiejar.Jar
}

func NewDBD() (*dbdClient, error) {
	ckj, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	c := &http.Client{
		Jar: ckj,
	}

	return &dbdClient{
		client:  c,
		cookies: ckj,
	}, nil
}

func (m dbdClient) request(method string, path string, params url.Values, headers http.Header, body io.Reader) (*http.Response, error) {
	u, err := url.Parse(fmt.Sprintf("https://%s", APIFunctionHost))
	if err != nil {
		return nil, err
	}

	u.Path = path
	u.RawQuery = params.Encode()

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header = headers

	fmt.Println(u.String())

	return m.client.Do(req)
}

type JSToken struct {
	Eid   string      `json:"eid"`
	Token string      `json:"token"`
	Ds    int         `json:"ds"`
	GiaD  int         `json:"gia_d"`
	DeMap interface{} `json:"deMap"`
}

func (m dbdClient) getApiEIDToken() (*JSToken, error) {
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

type h5st struct {
	T          string `json:"t"`
	AppId      string `json:"appid"`
	FunctionId string `json:"functionId"`
	BodyHash   string `json:"body"`
	Sign       string `json:"h5st"`
	STE        int64  `json:"_ste"`
	STK        string `json:"_stk"`
}

func (m dbdClient) getH5st(params url.Values, body []byte) (*h5st, error) {
	jsScript2, err := os.ReadFile("./js_security_v3_0.1.5.js")
	if err != nil {
		return nil, err
	}

	runtime, err := gojs.New()
	if err != nil {
		return nil, err
	}
	defer func() {
		// fmt.Println(runtime.ConsoleLog())
	}()
	vm := runtime.Runtime

	_, err = vm.RunString(string(jsScript2))
	if err != nil {
		return nil, err
	}

	sha256Bin := sha256.Sum256(body)
	h5stNew := map[string]interface{}{
		"appId":      "86b9f",
		"debug":      true,
		"preRequest": false,
		// "onSign": func(args goja.FunctionCall) goja.Value {
		// 	fmt.Println("onSign", args.Arguments[0].ToObject(vm).Export())
		// 	return goja.Undefined()
		// },
		// "onRequestTokenRemotely": func(args goja.FunctionCall) goja.Value {
		// 	fmt.Println("onRequestTokenRemotely", args.Arguments[0].ToObject(vm).Export())
		// 	return goja.Undefined()
		// },
		// "onRequestToken": func(args goja.FunctionCall) goja.Value {
		// 	fmt.Println("onRequestToken", args.Arguments[0].ToObject(vm).Export())
		// 	return goja.Undefined()
		// },
	}

	h5stSign := map[string]interface{}{
		"t":          params.Get("t"),
		"appid":      params.Get("appid"),
		"functionId": params.Get("functionId"),
		"body":       hex.EncodeToString(sha256Bin[:]),
	}

	_, err = vm.RunString(string(`
		function getH5st(init, sign) {
			console.log("getH5st",JSON.stringify(init),JSON.stringify(sign));
			var t = new Date().getTime()-60000;
			var O = new window.ParamsSign(init);
			
			var cv = {"v":"fbea2b972ec4f9fdc58985bf6ca175b5","t":t,"e":31536000}
			var wgl = {"v":"3972f1759e9c6333090f380aa016c810","t":t,"e":31536000}
			localStorage.setItem("WQ_gather_cv1", JSON.stringify(cv))
			localStorage.setItem("WQ_gather_wgl1", JSON.stringify(wgl))

			localStorage.setItem("WQ_vk1_86b9f_4.9", JSON.stringify({"v":"vlhq85ffl1zzk143","t":1733997973765,"e":31536000}))

			var tk = {"v":"dGswM3djZDQ0MWNlNTE4bklVZ2VYYlgwRXVhUE1VaDhNdmkzUzFmeVJzaUdSX3l6VEJ4dExpZ2h1aDJVRVNiOEFXeEh2c0RPdG5WRnJfTkpDTUZKZmdrZ0NORjM=","t":t,"e":86400}
			var algo = {"v":"ZnVuY3Rpb24gdGVzdCh0ayxmcCx0cyxhaSxhbGdvKXt2YXIgcmQ9J2F5WFcwUFMyOTFiMSc7dmFyIHN0cj0iIi5jb25jYXQodGspLmNvbmNhdChmcCkuY29uY2F0KHRzKS5jb25jYXQoYWkpLmNvbmNhdChyZCk7cmV0dXJuIGFsZ28uSG1hY01ENShzdHIsdGspO30=","t":t,"e":86400}
			localStorage.setItem("WQ_dy_tk_s_vlhq85ffl1zzk143", JSON.stringify(tk))
			localStorage.setItem("WQ_dy_algo_s_vlhq85ffl1zzk143", JSON.stringify(algo))

			var out = O.sign(sign);
			return out;
		}
	`))
	if err != nil {
		return nil, err
	}
	var getH5st func(init, sign map[string]interface{}) goja.Value
	err = vm.ExportTo(vm.Get("getH5st"), &getH5st)
	if err != nil {
		return nil, err
	}

	v := getH5st(h5stNew, h5stSign)
	v2, err := gojs.WaitForPromise(vm, v)
	if err != nil {
		return nil, err
	}

	h5stJson, err := v2.ToObject(vm).MarshalJSON()
	if err != nil {
		return nil, err
	}

	resp := &h5st{}
	json.Unmarshal(h5stJson, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (m dbdClient) callFunction(method string, functionId string, bodyParams map[string]interface{}) (reqsp *http.Response, err error) {
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
	headers.Set("sec-fetch-mode", "cors")
	headers.Set("sec-ch-ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
	headers.Set("content-type", "application/x-www-form-urlencoded")
	headers.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	headers.Set("referer", APIReferer)

	bodyJSON := []byte{}
	if bodyParams != nil {
		bodyJSON, err = json.Marshal(bodyParams)
		if err != nil {
			return nil, err
		}
	}

	h5st, err := m.getH5st(params, bodyJSON)
	if err != nil {
		return nil, err
	}

	params2 := url.Values{}
	params2.Set("t", t)
	params2.Set("appid", "paipai_h5")
	// params.Set("appid", "paipai_sale_pc")
	params2.Set("functionId", "dbd.auction.detail.v2")

	// getH5st({
	// 	"debug": true,
	// 	"preRequest": false,
	// 	"appId": "86b9f"},{
	// 	"functionId": "dbd.auction.detail.v2",
	// 	"t": "1734597645338",
	// 	"appid": "paipai_h5",
	// 	"body": "2fa0fd021188a41acc6a9a732cb663c5c4d6201ac173ca9b1438bc79a96844f0"
	// })
	fmt.Println(dbd.getH5st(params2, bodyJSON))

	switch method {
	case "GET":
		params.Set("h5st", h5st.Sign)
		params.Set("body", string(bodyJSON))
	case "POST":
		body := url.Values{
			"h5st": []string{h5st.Sign},
			"body": []string{string(bodyJSON)},
		}
		fmt.Println(body)
		bodyJSON = []byte(body.Encode())
	}
	bodyJSON = []byte(strings.ReplaceAll(string(bodyJSON), "%3A", ":"))
	bodyJSON = []byte(strings.ReplaceAll(string(bodyJSON), "%2C", ","))

	fmt.Println(string(bodyJSON))
	return m.request(method, APIFunctionPath, params, headers, bytes.NewBuffer(bodyJSON))
}
func (m dbdClient) callFunctionEx(method string, functionId string, bodyParams map[string]interface{}, outResp interface{}) (err error) {
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
	fmt.Println(string(bodyBin))

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

func (m dbdClient) RangeCookie(cb func(name, value string) bool) {
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

func (m dbdClient) SetCookie(cookie string) {
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

func (m dbdClient) AuctionDetail(id int) (*DBDAuctionDetail, error) {
	body := map[string]interface{}{
		"ts":                 time.Now().UnixMilli(),
		"auctionId":          strconv.Itoa(id),
		"dbdApiVersion":      "20200623",
		"p":                  2,
		"sourceTag":          2,
		"mpSource":           1,
		"auctionProductType": 1,
	}
	functionID := "dbd.auction.detail.v2"

	data := DBDAuctionDetail{}
	if err := m.callFunctionEx("POST", functionID, body, &data); err != nil {
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

func (m dbdClient) AuctionPriceInfo(id int) (*DBDAuctionAddress, error) {
	body := map[string]interface{}{
		"auctionId": id,
		"mpSource":  1,
		"sourceTag": 2,
	}
	functionID := "dbd.auction.detail.saleInfo"

	data := DBDAuctionAddress{}
	if err := m.callFunctionEx("GET", functionID, body, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func (m dbdClient) AuctionPrice(id int, address string, price float64) error {
	eid := ""
	m.RangeCookie(func(name, value string) bool {
		if name == APIEIDKey {
			eid = value
			return false
		}

		return true
	})
	if eid == "" {
		return fmt.Errorf("没有找到可用的 EID")
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
		return err
	}
	defer resp.Body.Close()

	bodyBin, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
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
		return err
	}

	if respStruct.Code != 0 {
		return fmt.Errorf("resp: %v=%v", respStruct.Code, respStruct.Message)
	}
	if respStruct.Result.Code != 200 {
		return fmt.Errorf("resp.result: %v=%v", respStruct.Result.Code, respStruct.Result.Message)
	}

	return nil
}

func (m dbdClient) AuctionCurrentPrice(id int) (*DBDAuctionCurrentPrice, error) {
	body := map[string]interface{}{
		"auctionId": id,
		"mpSource":  1,
		"sourceTag": 2,
	}
	functionID := "paipai.auction.get_current_and_offerNum"

	resp, err := m.callFunction("POST", functionID, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBin, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Println(string(bodyBin))

	respStruct := struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Result  struct {
			Code    int                    `json:"code"`
			Message string                 `json:"message"`
			Data    DBDAuctionCurrentPrice `json:"data"`
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

	return &respStruct.Result.Data, nil
}

func (m dbdClient) AuctionPriceRecords(id int) ([]DBDAuctionPriceRecord, error) {
	body := map[string]interface{}{
		"auctionId": id,
		"mpSource":  1,
		"sourceTag": 2,
	}
	functionID := "paipai.auction.bidrecords"

	resp, err := m.callFunction("POST", functionID, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBin, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Println(string(bodyBin))

	respStruct := struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Result  struct {
			Code    int                     `json:"code"`
			Message string                  `json:"message"`
			Data    []DBDAuctionPriceRecord `json:"data"`
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

	return respStruct.Result.Data, nil
}

func (m dbdClient) ProductBidder(ids ...int) (map[int]DBDProductBidder, error) {
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

func (m dbdClient) ProductSearch(query string, status string, page int) ([]DBDProductInfo, error) {
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
