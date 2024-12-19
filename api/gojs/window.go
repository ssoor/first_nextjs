package gojs

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/dop251/goja"
)

func injectWindow(runtime *Runtime, html Element) error {
	if html == nil {
		// 初始化简单的 DOM 树
		html = createElement(runtime, "", "html", "", "")
		head := createElement(runtime, "", "head", "", "")
		body := createElement(runtime, "", "body", "", "")

		div := createElement(runtime, "myDiv", "body", "container", "Hello, World!")
		div.SetAttribute("class", "container")

		body.AppendChild(runtime.ToValue(map[string]interface{}(div)))
		html.AppendChild(runtime.ToValue(map[string]interface{}(head)))
		html.AppendChild(runtime.ToValue(map[string]interface{}(body)))
	}

	// 创建模拟的 document
	runtime.Document.Root = html

	// 创建 localStorage 模拟对象
	ls := NewLocalStorage()
	localStorage := map[string]interface{}{
		"setItem": func(key, value string) {
			ls.SetItem(key, value)
		},
		"getItem": func(key string) string {
			return ls.GetItem(key)
		},
		"removeItem": func(key string) {
			ls.RemoveItem(key)
		},
		"clear": func() {
			ls.Clear()
		},
		"key": func(index int) *string {
			return ls.Key(index)
		},
		"length": func() int {
			return ls.Length()
		},
	}
	navigator := map[string]interface{}{
		"userAgent":           "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		"language":            "zh-CN",
		"platform":            "MacIntel",
		"hardwareConcurrency": 8,
		"plugins": map[string]interface{}{
			"length": 5,
		},
		"languages": []string{"zh-CN", "zh", "cn"},
	}

	vm := runtime.Runtime
	// 创建 HTMLAllCollection.prototype 对象
	htmlAllCollectionProto := vm.NewObject()

	// 创建 HTMLAllCollection 构造函数
	htmlAllCollection := vm.NewObject()
	htmlAllCollection.Set("prototype", htmlAllCollectionProto)
	vm.Set("HTMLAllCollection", htmlAllCollection)

	document := map[string]interface{}{
		"all": map[string]interface{}{
			"__proto__": htmlAllCollectionProto,
		},
		"querySelector": func(selector string) map[string]interface{} {
			// fmt.Println("document.querySelector2", selector)
			return runtime.Document.QuerySelector(selector)
		},
		"head": map[string]interface{}{
			"childElementCount": 4,
		},
		"body": map[string]interface{}{
			"childElementCount": 4,
		},
		"cookie": "",
		"getElementsByTagName": func(name string) []map[string]interface{} {
			// fmt.Println("document.getElementsByTagName", name)
			eles := findElementByTag(runtime.Document.Root, name)

			outs := []map[string]interface{}{}
			for _, v := range eles {
				outs = append(outs, v)
			}
			return outs
		},
		"createElement": func(name string) map[string]interface{} {
			// fmt.Println("document.createElement", name)
			return createElement(runtime, "", name, "", "")
		},
	}

	window := map[string]interface{}{
		"navigator":    navigator,
		"localStorage": localStorage,
		"document":     document,
	}
	runtime.Set("window", window)
	runtime.Set("document", document)
	runtime.Set("navigator", navigator)
	runtime.Set("localStorage", localStorage)

	return nil
}

// Document 模拟 DOM 树的根
type Document struct {
	Root Element
}

// querySelector 实现简单的选择器逻辑
func (doc *Document) QuerySelector(selector string) Element {
	if selector == "" {
		return nil
	}
	// 支持简单的 ID 和 TagName 选择器
	if strings.HasPrefix(selector, "#") {
		id := selector[1:]
		return findElementByID(doc.Root, id)
	}
	eles := findElementByTag(doc.Root, selector)
	if len(eles) == 0 {
		return nil
	}
	return eles[0]
}

// 递归查找 ID 的节点
func findElementByID(e Element, id string) Element {
	if e.Get("ID") == id {
		return e
	}
	for _, child := range e.Children() {
		if result := findElementByID(child, id); result != nil {
			return result
		}
	}
	return nil
}

// 递归查找 TagName 的节点
func findElementByTag(e Element, tag string) (eles []Element) {
	if e.Get("TagName") == tag {
		eles = append(eles, e)
	}
	for _, child := range e.Children() {
		if result := findElementByTag(child, tag); result != nil {
			eles = append(eles, result...)
		}
	}
	return eles
}

// Element 模拟 DOM 节点
type Element map[string]interface{}

func createElement(r *Runtime, id, name, class, text string) Element {
	ele := Element{
		"_runtime":    r,
		"ID":          id,
		"TagName":     name,
		"ClassName":   class,
		"Children":    []Element{},
		"Attributes":  make(map[string]string),
		"TextContent": text,
		"parentNode":  nil,
	}

	ele["appendChild"] = ele.AppendChild
	ele["removeChild"] = ele.RemoveChild
	ele["SetAttribute"] = ele.SetAttribute
	ele["SetAttribute"] = ele.SetAttribute

	return ele
}

// 添加获取属性的方法
func (e Element) Get(name string) string {
	return e[name].(string)
}

// 添加获取属性的方法
func (e Element) FullPath() string {
	name := e.Get("TagName")

	if parent := e["parentNode"]; parent != nil {
		return parent.(Element).FullPath() + "/" + name
	}

	return name
}

func (e Element) AppendChild(c goja.Value) {
	runtime := e["_runtime"].(*Runtime)
	children := Element(c.ToObject(runtime.Runtime).Export().(map[string]interface{}))

	// fmt.Println("element.appendChild", e.FullPath(), children.Get("TagName"))
	c.Export().(map[string]interface{})["parentNode"] = e
	e["Children"] = append(e.Children(), children)

	switch children.Get("TagName") {
	case "script":
		src := children.Get("src")
		onload, _ := children["onload"].(func(goja.FunctionCall) goja.Value)
		onerror, _ := children["onerror"].(func(goja.FunctionCall) goja.Value)
		resp, err := http.Get(src)
		if err != nil {
			event := map[string]interface{}{
				"type":   "load",
				"target": "script",
			}
			onerror(goja.FunctionCall{Arguments: []goja.Value{runtime.ToValue(event)}})
		} else {
			defer resp.Body.Close()
			defer func() {
				err := recover()
				fmt.Println(err)
			}()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				event := map[string]interface{}{
					"type":   "load",
					"target": "script",
				}
				onerror(goja.FunctionCall{Arguments: []goja.Value{runtime.ToValue(event)}})
			} else {
				event := map[string]interface{}{
					"type":   "load",
					"target": "script",
				}
				runtime.RunScript(src, string(body))
				// fmt.Println("load.script", string(body))
				onload(goja.FunctionCall{Arguments: []goja.Value{runtime.ToValue(event)}})
			}
		}
	}
}

func (e *Element) RemoveChild(children Element) {
	fmt.Println("element.RemoveChild", e.FullPath(), children["TagName"])
}

// 添加获取属性的方法
func (e Element) GetAttribute(attr string) string {
	return e["Attributes"].(map[string]string)[attr]
}

// 添加设置属性的方法
func (e Element) SetAttribute(attr, value string) {
	e["Attributes"].(map[string]string)[attr] = value
}
func (e Element) Children() []Element {
	return e["Children"].([]Element)
}

// LocalStorage 模拟浏览器的 localStorage
type LocalStorage struct {
	data  map[string]string
	mutex sync.Mutex
}

// NewLocalStorage 创建一个新的 LocalStorage 实例
func NewLocalStorage() *LocalStorage {
	return &LocalStorage{
		data: make(map[string]string),
	}
}

// SetItem 设置键值对
func (ls *LocalStorage) SetItem(key, value string) {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
	ls.data[key] = value
	// fmt.Println("LocalStorage.Set", key, value)
}

// GetItem 获取指定键的值
func (ls *LocalStorage) GetItem(key string) string {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
	// fmt.Println("LocalStorage.Get", key, ls.data[key])
	return ls.data[key]
}

// RemoveItem 删除指定键的值
func (ls *LocalStorage) RemoveItem(key string) {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
	// fmt.Println("LocalStorage.RemoveItem", key, ls.data[key])
	delete(ls.data, key)
}

// Clear 清空所有数据
func (ls *LocalStorage) Clear() {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
	ls.data = make(map[string]string)
}

// Key 根据索引获取键名
func (ls *LocalStorage) Key(index int) *string {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
	i := 0
	for key := range ls.data {
		if i == index {
			return &key
		}
		i++
	}
	return nil
}

// Length 返回存储的数据数量
func (ls *LocalStorage) Length() int {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
	return len(ls.data)
}
