package main

// import (
// 	"bytes"
// 	"crypto/hmac"
// 	m "crypto/md5"
// 	"crypto/rand"
// 	"crypto/sha256"
// 	"crypto/sha512"
// 	"fmt"
// 	"log"
// 	"math/big"
// 	random "math/rand"
// 	"net/http"
// 	"net/url"
// 	"reflect"
// 	"regexp"
// 	"strconv"
// 	"strings"
// 	"sync"
// 	"time"
// )

// var (
// 	randSeek           = int64(1)
// 	l                  sync.Mutex
// 	token              string
// 	rd                 []string
// 	f                  string
// 	algo               string
// 	coins              int    // 金币数量
// 	foodNum            = 0    // 白菜数量
// 	activeId           = ""   // 活动ID
// 	share_code         string // 助力码
// 	phone_id           string // 设备ID
// 	egg_num            = 0    // 金蛋数量
// 	newcomer_task_step = [...]string{"A-1", "A-2", "A-3", "A-4", "A-5", "A-6", "A-7", "A-8", "A-9",
// 		"A-10", "A-11", "A-12", "B-1", "C-1", "D-1", "E-1", "E-2", "E-3", "E-4", "E-5",
// 		"F-1", "F-2", "G-1", "G-2", "G-3", "G-4", "G-5", "G-6", "G-7", "G-8", "G-9"}
// 	curTaskStep string
// )

// func Jxrequest(c *http.Request, path, body, user string) string {
// 	phoneId := GetRandomString(16)
// 	timestamp := time.Now().Unix() * 1000
// 	jsToken := tom5(fmt.Sprintf("%s%s%stPOamqCuk9NLgVPAljUyIHcPRmKlVxDy", user, strconv.FormatInt(timestamp, 10), phoneId))
// 	params := url.Values{}
// 	params.Add("channel", "7")
// 	params.Add("sceneid", "1001")
// 	params.Add("activeid", activeId)
// 	params.Add("activekey", "null")
// 	params.Add("_ste", "1")
// 	params.Add("_", strconv.FormatInt(time.Now().Unix()*1000+2, 10))
// 	params.Add("sceneval", "2")
// 	params.Add("g_login_type", "1")
// 	params.Add("callback", "")
// 	params.Add("g_ty", "ls")
// 	params.Add("jxmc_jstoken", jsToken)
// 	var mapBody map[string]string
// 	_ = j.Unmarshal([]byte(body), &mapBody)
// 	for s, i := range mapBody {
// 		if params.Has(s) {
// 			params.Add(s, i)
// 		} else {
// 			params.Set(s, i)
// 		}
// 	}
// 	purl := fmt.Sprintf("https://m.jingxi.com/%s?%s", path, params.Encode())
// 	h5st := encrypt(purl, "")
// 	purl = fmt.Sprintf("%s&h5st=%s", purl, h5st)
// 	timer := time.NewTimer(1 * time.Second)
// 	select {
// 	case <-timer.C:
// 		resp, err := c.SetHeaders(map[string]string{"referer": "https://st.jingxi.com/"}).Post(purl)
// 		if err != nil {
// 			log.Println(err)
// 		}
// 		timer.Stop()
// 		return string(resp.Body())
// 	}
// }

// func tom5(str string) string {
// 	data := []byte(str)
// 	has := m.Sum(data)
// 	return fmt.Sprintf("%x", has)
// }

// func encrypt(u, stk string) string {
// 	timestamp := time.Now().Format("20060102150405")
// 	timestamp = fmt.Sprintf("%s%s", timestamp, strconv.FormatInt(time.Now().UnixNano(), 10)[:3])
// 	r, _ := url.Parse(u)
// 	if stk == "" {
// 		stk = r.Query().Get("_stk")
// 	}
// 	s := fmt.Sprintf("%s%s%s%s%s", token, f, timestamp, "10001", rd[1])
// 	jxx := new(jx)
// 	method := reflect.ValueOf(jxx).MethodByName(fmt.Sprintf("Call%s", algo))
// 	var val []reflect.Value
// 	if strings.Contains(fmt.Sprintf("Call%s", algo), "Hmac") {
// 		val = method.Call([]reflect.Value{reflect.ValueOf(s), reflect.ValueOf(token)})
// 	} else {
// 		val = method.Call([]reflect.Value{reflect.ValueOf(s)})
// 	}
// 	var tp []string
// 	for _, s2 := range strings.Split(stk, ",") {
// 		tp = append(tp, fmt.Sprintf("%s:%s", s2, r.Query().Get(s2)))
// 	}
// 	hash := jxx.CallHmacSHA256(strings.Join(tp, "&"), val[0].String())
// 	return strings.Join([]string{timestamp, f, "10001", token, hash}, ";")
// }

// func fp() string {
// 	e := "0123456789"
// 	a := 13
// 	i := ""
// 	for a > 0 {
// 		result, _ := rand.Int(rand.Reader, big.NewInt(int64(len(e))))
// 		i += fmt.Sprintf("%s", result)
// 		a -= 1
// 	}
// 	i += fmt.Sprintf("%s", strconv.FormatInt(time.Now().Unix()*100, 10))
// 	return i[0:16]
// }

// func GetRandomString(num int, str ...string) string {
// 	s := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
// 	if len(str) > 0 {
// 		s = str[0]
// 	}
// 	l := len(s)
// 	r := random.New(random.NewSource(getRandSeek()))
// 	var buf bytes.Buffer
// 	for i := 0; i < num; i++ {
// 		x := r.Intn(l)
// 		buf.WriteString(s[x : x+1])
// 	}
// 	return buf.String()
// }

// func getRandSeek() int64 {
// 	l.Lock()
// 	if randSeek >= 100000000 {
// 		randSeek = 1
// 	}
// 	randSeek++
// 	l.Unlock()
// 	return time.Now().UnixNano() + randSeek

// }

// type jx struct {
// 	c string
// }

// func (t *jx) CallMD5(val string) string {
// 	data := []byte(val)
// 	has := m.Sum(data)
// 	return fmt.Sprintf("%x", has)
// }
// func (t *jx) CallHmacMD5(key, val string) string {
// 	h := hmac.New(m.New, []byte(key))
// 	h.Write([]byte(val))
// 	return fmt.Sprintf("%x", h.Sum(nil))
// }
// func (t *jx) CallSHA256(val string) string {
// 	return fmt.Sprintf("%x", sha256.Sum256([]byte(val)))
// }
// func (t *jx) CallHmacSHA256(key, val string) string {
// 	h := hmac.New(sha256.New, []byte(key))
// 	h.Write([]byte(val))
// 	return fmt.Sprintf("%x", h.Sum(nil))
// }
// func (t *jx) CallSHA512(val string) string {
// 	return fmt.Sprintf("%x", sha512.Sum512([]byte(val)))
// }
// func (t *jx) CallHmacSHA512(key, val string) string {
// 	h := hmac.New(sha512.New, []byte(key))
// 	h.Write([]byte(val))
// 	return fmt.Sprintf("%x", m.Sum(nil))
// }

// func getEncrypt() {
// 	f = fp()
// 	response, err := HttpClient.SetDebug(false).
// 		R().
// 		SetHeaders(map[string]string{
// 			"Authority":       "cactus.jd.com",
// 			"Pragma":          "no-cache",
// 			"Cache-Control":   "no-cache",
// 			"Accept":          "application/json",
// 			"Content-Type":    "application/json",
// 			"Origin":          "https://st.jingxi.com",
// 			"Sec-Fetch-Site":  "cross-site",
// 			"Sec-Fetch-Mode":  "cors",
// 			"Sec-Fetch-Dest":  "empty",
// 			"Referer":         "https://st.jingxi.com/",
// 			"Accept-Language": "zh-CN,zh;q=0.9,zh-TW;q=0.8,en;q=0.7",
// 		}).
// 		SetBody(map[string]string{
// 			"version":      "1.0",
// 			"fp":           f,
// 			"appId":        "10001",
// 			"timestamp":    strconv.FormatInt(time.Now().Unix()*1000, 10),
// 			"platform":     "web",
// 			"expandParams": "",
// 		}).
// 		Post("https://cactus.jd.com/request_algo?g_ty=ajax")
// 	body := string(response.Body())
// 	if err != nil {
// 		log.Println("签名算法获取失败:", err)
// 	}
// 	if json.Get(body, "status").Int() == 200 {
// 		token = json.Get(body, "data.result.tk").String()
// 		rd = regexp.MustCompile("rd='(.*)';").FindStringSubmatch(json.Get(body, "data.result.algo").String())
// 		algo = regexp.MustCompile(`algo\.(.*)\(`).FindStringSubmatch(json.Get(body, "data.result.algo").String())[1]
// 		log.Printf("获取到签名算法为: %s tk为: %s", algo, token)
// 	}

// }
