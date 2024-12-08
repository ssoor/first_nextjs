package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

func Test_T2(t *testing.T) {
	vm := goja.New()
	jsScript2, _ := os.ReadFile("./getJsToken.js")
	_, err := vm.RunString(string(jsScript2))
	if err != nil {
		fmt.Println("JS代码有问题！")
		panic(err)
	}

	var getJsToken func(userAgent, url, bizId, eid string) goja.Promise
	err = vm.ExportTo(vm.Get("getJsToken"), &getJsToken)
	if err != nil {
		fmt.Println("Js函数映射到 Go 函数失败！")
		panic(err)
	}
	v := getJsToken(
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		"https://paipai.jd.com/auction-detail/393024207",
		"paipai_sale_pc",
		"F3E5M3YU3O67BDGCSBX55H6C7DIED5DTJADUYLQETEGSKDI3KJZB6EZFJKCCIB2DWFUUU5UYJLGB4RHX6FHD2L2JW4",
	)
	v.Result()
	v2 := v.Result().ToObject(vm)

	// fmt.Println(v2.Get("a").ToString())
	// fmt.Println(v2.Get("d").ToString())

	params := url.Values{
		"a": []string{v2.Get("a").ToString().String()},
		"d": []string{v2.Get("d").ToString().String()},
	}
	// fmt.Println(params.Encode())
	req, err := http.NewRequest("POST", "https://jra.jd.com/jsTk.do", strings.NewReader(params.Encode()))
	if err != nil {
		panic(err)
	}

	header := req.Header
	header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println((v2.Export()))
	fmt.Println(string(respBody))

	t.Fatal("成功")
}

func evalJS(vm *goja.Runtime, regex string, body []byte) []byte {
	callRegex, err := regexp.Compile(regex)
	if err != nil {
		panic(err)
	}

	for {
		matchIndex := callRegex.FindIndex(body)
		if matchIndex == nil {
			break
		}

		v := string(body[matchIndex[0]:matchIndex[1]])
		val, err := vm.RunString(v)
		if err != nil {
			panic(err)
		}

		valJSON, err := json.Marshal(val.String())
		if err != nil {
			panic(err)
		}
		newBody := append([]byte{}, body[:matchIndex[0]]...)
		newBody = append(newBody, valJSON...)
		newBody = append(newBody, body[matchIndex[1]:]...)

		body = newBody
		fmt.Printf("matchStrs = %v = %v\n", v, val.String())
	}

	return body
}

func Test_dbd_ProductSearch(t *testing.T) {
	body, err := os.ReadFile("./pc-tk.js2")
	if err != nil {
		panic(err)
	}

	vm := goja.New()
	_, err = vm.RunString(jsScript)
	if err != nil {
		fmt.Println("JS代码有问题！")
		panic(err)
	}

	body = evalJS(vm, `_0x2e0a6d\(\d+\)`, body)
	body = evalJS(vm, `_0x201a\(\d+\)`, body)
	body = evalJS(vm, `[_coidtraesn]\(\d+\)`, body)

	bodyStr := string(body)
	bodyStr = strings.ReplaceAll(bodyStr, "t.call(r, e[n], n, e) === {})", "false)")
	bodyStr = strings.ReplaceAll(bodyStr, `t["call"](r, e[i], i, e) === {})`, "false)")
	bodyStr = strings.ReplaceAll(bodyStr, "_0x5221fe", "__log_report")
	bodyStr = strings.ReplaceAll(bodyStr, "_0x201a", "__tostr_e")
	bodyStr = strings.ReplaceAll(bodyStr, "_0x2e0a6d", "__tostr_s")
	bodyStr = strings.ReplaceAll(bodyStr, "_0x5f424c", "__get_jstoken")
	bodyStr = strings.ReplaceAll(bodyStr, "_0x370d2c", "__tokenStruct")
	bodyStr = strings.ReplaceAll(bodyStr, "_0x29c61f", "__execCallback")
	bodyStr = strings.ReplaceAll(bodyStr, "_0x13997a", "__tkKeys")
	bodyStr = strings.ReplaceAll(bodyStr, "_0x1eda92", "__tkKeyMode")
	bodyStr = strings.ReplaceAll(bodyStr, "_0x292020", "__localStorage")
	os.WriteFile("./pc-tk-des.js2", []byte(bodyStr), 0644)

	var fn func(int32) string
	err = vm.ExportTo(vm.Get("__tostr_e"), &fn)
	if err != nil {
		fmt.Println("Js函数映射到 Go 函数失败！")
		panic(err)
	}
	fmt.Println("斐波那契数列第30项的值为：", fn(297))
	//
	//
	t.Fatal("完成")
}

const jsScript = `
e = __tostr_e;
o = __tostr_e;
i = __tostr_e;
d = __tostr_e;
c = __tostr_e;
_ = __tostr_e;
n = __tostr_e;
s = __tostr_e;
t = __tostr_e;
r = __tostr_e;
a = __tostr_e;
_0x201a = __tostr_e;
_0x2e0a6d = __tostr_e;

_0x1eda92 = (() => {
        for (var e = __tostr_e, t = __tostr_gettokens(); ;)
            try {
                if (140987 == +parseInt(e(588)) + -parseInt(e(570)) / 2 + parseInt(e(369)) / 3 + parseInt(e(361)) / 4 * (parseInt(e(288)) / 5) + -parseInt(e(325)) / 6 + -parseInt(e(242)) / 7 * (-parseInt(e(590)) / 8) + parseInt(e(619)) / 9 * (-parseInt(e(568)) / 10))
                    break;
                t.push(t.shift())
            } catch (e) {
                t.push(t.shift())
            }
    }
    )()

function __tostr_e(e, t) {
    var r = __tostr_gettokens();
    return (__tostr_e = function (e, t) {
        return r[e -= 224]
    }
    )(e, t)
}
function __tostr_gettokens() {
    var e = ["windows", "?????", "MAX_TEXTURE_SIZE", "getSupportFonts", "removeChild", "getBrowserMode", "w60", "withCredentials", "w52", "addBehavior", "ceil", "substring", "offsetWidth", "webgl", "globalStorage", "Msxml2.XMLHTTP", "clearColor", "attribute vec2 attrVertex;varying vec2 varyinTexCoordinate;uniform vec2 uniformOffset;void main(){varyinTexCoordinate=attrVertex+uniformOffset;gl_Position=vec4(attrVertex,0,1);}", "ARRAY_BUFFER", "$version", "VENDOR", "Shell.UIHelper", "3AB9D23F7A4B3C9B", "MAX_VERTEX_ATTRIBS", "cfp:", "isJsTokenFinished", "sun", "openDatabase", "/jsTk.do", "hidden", "w25", "MAX_TEXTURE_MAX_ANISOTROPY_EXT", "init func error :", "w12", "font", "xhr", "string", "env", "rangeMax", "getCanvasInfo", "canvas", "ShockwaveFlash.ShockwaveFlash", "w34", "w22", "strict", "rgba(102, 204, 0, 0.2)", "3AB9D23F7A4B3CFF", "createShader", "getCanvasFp", "plugins", "getIndexedDBSupport", "w58", "win", "sdkToken", "chrome/", "push", "append", "extensions", "BlobBuilder", "getCurrentPageProtocol", "ldeKey", "w17", "call", "wuv:", "getScreenInfo", "OSF1", "vertexAttribPointer", "compareVersion", "23IL<N01c7KvwZO56RSTAfghiFyzWJqVabGH4PQdopUrsCuX*xeBjkltDEmn89.-", "test", "toLowerCase", "onreadystatechange", "getFpServerDomain", "HIGH_FLOAT", "getNavigatorCpuClass", "innerHTML", "attachShader", "shadingLV", "sendRequest error : ", "colorDepth", "MAX_TEXTURE_IMAGE_UNITS", "w13", "fp keys:", "w42", "getUrlQueryStr", "opera", "closePath", "taintEnabled", "FRAGMENT_SHADER", "_gia_d", "lil", "javaEnabled", "w23", "getUserAgent", "availHeight,availWidth,colorDepth,bufferDepth,deviceXDPI,deviceYDPI,height,width,logicalXDPI,logicalYDPI,pixelDepth,updateInterval", "getEnvExcludeOptions", "SHADING_LANGUAGE_VERSION", "w57", "PCA9D23F7A4B3CSS", "getFp", "getBrowserInfo", "join", "itemSize", "evenodd", "color", "Microsoft Internet Explorer", "w38", "getLanguage", "undefined", "get", "getStoreCheck", "360???", "w24", "rmocx.RealPlayer G2 Control", ".localdomain", "getWebglFp", "4game;AdblockPlugin;AdobeExManCCDetect;AdobeExManDetect;Alawar NPAPI utils;Aliedit Plug-In;Alipay Security Control 3;AliSSOLogin plugin;AmazonMP3DownloaderPlugin;AOL Media Playback Plugin;AppUp;ArchiCAD;AVG SiteSafety plugin;Babylon ToolBar;Battlelog Game Launcher;BitCometAgent;Bitdefender QuickScan;BlueStacks Install Detector;CatalinaGroup Update;Citrix ICA Client;Citrix online plug-in;Citrix Receiver Plug-in;Coowon Update;DealPlyLive Update;Default Browser Helper;DivX Browser Plug-In;DivX Plus Web Player;DivX VOD Helper Plug-in;doubleTwist Web Plugin;Downloaders plugin;downloadUpdater;eMusicPlugin DLM6;ESN Launch Mozilla Plugin;ESN Sonar API;Exif Everywhere;Facebook Plugin;File Downloader Plug-in;FileLab plugin;FlyOrDie Games Plugin;Folx 3 Browser Plugin;FUZEShare;GDL Object Web Plug-in 16.00;GFACE Plugin;Ginger;Gnome Shell Integration;Google Earth Plugin;Google Earth Plug-in;Google Gears 0.5.33.0;Google Talk Effects Plugin;Google Update;Harmony Firefox Plugin;Harmony Plug-In;Heroes & Generals live;HPDetect;Html5 location provider;IE Tab plugin;iGetterScriptablePlugin;iMesh plugin;Kaspersky Password Manager;LastPass;LogMeIn Plugin 1.0.0.935;LogMeIn Plugin 1.0.0.961;Ma-Config.com plugin;Microsoft Office 2013;MinibarPlugin;Native Client;Nitro PDF Plug-In;Nokia Suite Enabler Plugin;Norton Identity Safe;npAPI Plugin;NPLastPass;NPPlayerShell;npTongbuAddin;NyxLauncher;Octoshape Streaming Services;Online Storage plug-in;Orbit Downloader;Pando Web Plugin;Parom.TV player plugin;PDF integrado do WebKit;PDF-XChange Viewer;PhotoCenterPlugin1.1.2.2;Picasa;PlayOn Plug-in;QQ2013 Firefox Plugin;QQDownload Plugin;QQMiniDL Plugin;QQMusic;RealDownloader Plugin;Roblox Launcher Plugin;RockMelt Update;Safer Update;SafeSearch;Scripting.Dictionary;SefClient Plugin;Shell.UIHelper;Silverlight Plug-In;Simple Pass;Skype Web Plugin;SumatraPDF Browser Plugin;Symantec PKI Client;Tencent FTN plug-in;Thunder DapCtrl NPAPI Plugin;TorchHelper;Unity Player;Uplay PC;VDownloader;Veetle TV Core;VLC Multimedia Plugin;Web Components;WebKit-integrierte PDF;WEBZEN Browser Extension;Wolfram Mathematica;WordCaptureX;WPI Detector 1.4;Yandex Media Plugin;Yandex PDF Viewer;YouTube Plug-in;zako", "webgl,experimental-webgl,moz-webgl,webkit-3d", "numItems", "deviceTime", "w54", "4.90", "isDegrade", "toUTCString", "w26", "yes", "w53", "protocol", "#f60", "bizTimeout", "AgControl.AgControl", "object", "applewebkit/", "UNMASKED_VENDOR_WEBGL", "display", "DevalVRXCtrl.DevalVRXCtrl.1", "errTrace", "jdd03", "w32", "10350fnlnoj", "CA1AN5BV0CA8DS2EPC", "93182cCAEsp", "monospace", "random", "w21", "fill", "setTime", "rangeMin", "FP function : [", "style", "function", "deMap", "worker", "wuv", "WebGLRenderingContext", "startTime", "w10", "drawArrays", "COLOR_BUFFER_BIT", "30166QilrIT", "fillText", "1879096lFsCCB", "trim", "createElement", "enable", "stack", "textBaseline", "match", "FLOAT", "language", "powerPC", "resp parse error", "getBlob", "metasr", "getLocalStorageSupport", "w41", "ALIASED_POINT_SIZE_RANGE", "PDF.PdfCtrl", "indexedDb", "msDoNotTrack", "SWCtl.SWCtl", "getItem", "inline", "indexedDB", "0123456789abcdef", "open", "getJdEid", "ENV Collector function : [", "uniform2f", "getJsToken", "261TIWKHM", "fpKey", "Skype.Detection", "STATIC_DRAW", "fillStyle", "Cwwm aa fjorddbank glbyphs veext qtuiz, ?", "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefghijklmnopqrstuvwxyz", "code", "createWorker", "env collect error", "application/javascript", "report log error :", "length", "useProgram", "getTimezoneOffset", "getIEPluginsString", "span", "serif", "navigator/", "RENDERER", "MEDIUM_FLOAT", "gia_d", "enumerationOrder", "sans-serif", "getContext", "w49", "each", "navigator", "firefox", "ActiveXObject", "QuickTimeCheckObject.QuickTimeCheck.1", "getParameter", "Microsoft.XMLHTTP", "w40", "sogoumobilebrowser", "w29", "onabort", "MAX_VERTEX_UNIFORM_VECTORS", "experimental-webgl", "timezoneOffset", "~~~", "w33", "w48", "w61", "toDataURL", "getColorDepth", "WEBGL_debug_renderer_info", "browser", "getExtension", "networkError_", "PCA9D23F7A4B3CFF", "randomStr", "http://", "360se", "https://mllog.jd.com/mlog/unite.do", "canvas_fp_md5", "cw:", "hasOwnProperty", "nt 6.1", "isReady", "LEQUAL", "fonts", "uuid", "HASH", "jsTokenKey", "map", "getAddBehaviorSupport", "safari/", "w18", "ibm", "visibility", "platform", "statusText", "getSessionStorageSupport", "stringify", "ALPHA_BITS", "opera/", "floor", "cookieEnabled", "__fp_domain", "setRequestHeader", "getDoNotTrack", "getPlugins", "w11", "screenResolution", "fillRect", "alphabetic", "NetBSD", "https://", "devicePixelRatio,screenTop,screenLeft", "ucbrowser", "hash128", "] Cost time :", "BSD", "isValidJsToken", "close", "createBuffer", "osVersion", "store", "w16", "hardwareConcurrency", "giaDKey", "localStorage", "cpuClass", "FreeBSD", "appendChild", "nativeForEach", "applewebkit_chrome", "02_pt_XY8T_53299815385", "; path=/; domain=", "callTime", "body", "the world", "HIGH_INT", "getFeature", "MAX_COMBINED_TEXTURE_IMAGE_UNITS", "domain", "feature", "storeCheck", "cookie", "slice", "multiply", "VERTEX_SHADER", "LOW_FLOAT", "???????", "w59", "bufferData", "msie ", "screen", "readyState", "RealPlayer.RealPlayer(tm) ActiveX Control (32-bit)", "XMLHttpRequest", "w31", "fpTsKey", "7vucKee", "responseTime", "Adodb.Stream", "safari", "oscpu", "position", "w45", "w30", "abort_", "&d=", "MozBlobBuilder", "JS_DEVICE_EMPTY", "rect", "getHardwareConcurrency", "isPointInPath", "exec", "rmocx.RealPlayer G2 Control.1", "w56", "application/x-www-form-urlencoded;charset=UTF-8", "aix", "reportWorker", "qd_uid", "nativeMap", "SymbianOS/", "sampleRate", "getOpenDatabaseSupport", "appName", "LOW_INT", "Worker", "beginPath", "MOZ_EXT_texture_filter_anisotropic", "wil", "w14", "browserMode", "chrome", "extensions:", "https:", "getContextAttributes", "callMethod", "shaderSource", "eid", "QQ???", "MEDIUM_INT", "wur:", "compatMode", "version", "45pNPWOe", "onmessage", "nt 6.0", "PCTSD23F7A4B3CFF", "freebsd", "getNavigatorPlatform", "antialias", "getOsInfo", "mac", "fast", "GIA_LDE_MAP_KEY", "onmessage = function (event) {\n    var data = JSON.parse(event.data);\n    try {\n        var httpRequest;\n        try {\n            httpRequest = new XMLHttpRequest();\n        } catch (h) {}\n        if (!httpRequest)\n            try {\n                httpRequest = new (window['ActiveXObject'])('Microsoft.XMLHTTP')\n            } catch (l) {}\n        if (!httpRequest)\n            try {\n                httpRequest = new (window['ActiveXObject'])('Msxml2.XMLHTTP')\n            } catch (r) {}\n        if (!httpRequest)\n            try {\n                httpRequest = new (window['ActiveXObject'])('Msxml3.XMLHTTP')\n            } catch (n) {}\n\n        if(data){\n            httpRequest['open']('POST', data.url, data.async);\n            httpRequest['withCredentials'] = true;\n            httpRequest['setRequestHeader']('Content-Type', data.isJson ? 'application/json;charset=UTF-8' : 'application/x-www-form-urlencoded;charset=UTF-8');\n            httpRequest['onreadystatechange'] = function () {\n                if (4 === httpRequest['readyState'] && 200 === httpRequest['status']) {\n                    postMessage(httpRequest.responseText);\n                }\n            };\n            httpRequest['send'](data.data);\n        }\n\n    }catch (e){console.error(e);}\n};", "00000000", "w47", "audioKey", "getShaderPrecisionFormat", "webglversion", "PCTSD23F7A4B3CSS", "AcroPDF.PDF", "error", "setCookie", "prototype", "obtainPin", "split", "getCookie", "Firefox", "getRegularPluginsString", "startsWith", "MD5", "w15", "MAX_CUBE_MAP_TEXTURE_SIZE", "attrVertex", "deviceEndTime", "eidKey", "browserVersion", "width", "hex_md5", "1428096hiVuNL", "data", "symbianos", "getTime", "bindBuffer", "Msxml2.DOMDocument", "EXT_texture_filter_anisotropic", "arc", "bsd", "TRIANGLE_STRIP", "Netscape", "charAt", "responseText", "timeout_", "getAttribLocation", "hpux", "forEach", "indexOf", "qqbrowser", "precision", "deviceInfo", "tencenttraveler", "osf1", "enableVertexAttribArray", "description", "fontFamily", "div", "w43", "18pt Arial", "_*_UL05XPWG8HAE4UG7", "DEPTH_BUFFER_BIT", "#069", "timeout", "offsetHeight", "load", "reportTime", "46740VSddXQ", "webkitAudioContext", "execute", "height", "w46", "jra.jd.com", "expires=", "fontSize", "256173yWGDcE", "location", "getPropertyValue", "Msxml3.XMLHTTP", "win 9x", "vendor", "getBizId", "Scripting.Dictionary", "log", "contextName", "getWebglCanvas", "cleanAndPushDeS", "name", "MacromediaFlashPaper.MacromediaFlashPaper", "offsetUniform", "getAudioKey", "status", "href", "jsToken", "getSupportedExtensions", "doNotTrack", "getFpExcludeOptions", "nt 5.1", "POST", "charCodeAt", "suffixes", "vertexPosAttrib", "toString", "firefox/", "w51", "getColorRgb", "TDCCtl.TDCCtl", "sessionStorageKey", "VERSION", "init", "reportCnt", "TDEncrypt", "Content-Type", "getPositionInfo", "parse", "WMPlayer.OCX", "canvasFpKey", "token", "depthFunc", "w28", "unix", "userAgent", "sendRequest", "ontimeout", "endsWith", "11pt no-real-font-123", "sessionStorage", "2000", "set", "maxthon", "getScreenResolution", "Abadi MT Condensed Light;Adobe Fangsong Std;Adobe Hebrew;Adobe Ming Std;Agency FB;Arab;Arabic Typesetting;Arial Black;Batang;Bauhaus 93;Bell MT;Bitstream Vera Serif;Bodoni MT;Bookman Old Style;Braggadocio;Broadway;Calibri;Californian FB;Castellar;Casual;Centaur;Century Gothic;Chalkduster;Colonna MT;Copperplate Gothic Light;DejaVu LGC Sans Mono;Desdemona;DFKai-SB;Dotum;Engravers MT;Eras Bold ITC;Eurostile;FangSong;Forte;Franklin Gothic Heavy;French Script MT;Gabriola;Gigi;Gisha;Goudy Old Style;Gulim;GungSeo;Haettenschweiler;Harrington;Hiragino Sans GB;Impact;Informal Roman;KacstOne;Kino MT;Kozuka Gothic Pr6N;Lohit Gujarati;Loma;Lucida Bright;Lucida Fax;Magneto;Malgun Gothic;Matura MT Script Capitals;Menlo;MingLiU-ExtB;MoolBoran;MS PMincho;MS Reference Sans Serif;News Gothic MT;Niagara Solid;Nyala;Palace Script MT;Papyrus;Perpetua;Playbill;PMingLiU;Rachana;Rockwell;Sawasdee;Script MT Bold;Segoe Print;Showcard Gothic;SimHei;Snap ITC;TlwgMono;Tw Cen MT Condensed Extra Bold;Ubuntu;Umpush;Univers;Utopia;Vladimir Script;Wide Latin", "globalCompositeOperation", "applewebkit", "UC???"];
    return (__tostr_gettokens = function () {
        return e
    }
    )()
}


`
