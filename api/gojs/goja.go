package gojs

import (
	"bytes"
	"fmt"
	"io"

	"github.com/dop251/goja"
)

type Runtime struct {
	*goja.Runtime
	Document
	consoleLog bytes.Buffer
}

func (r *Runtime) ConsoleLog() io.Reader {
	return &r.consoleLog
}

func New() (*Runtime, error) {
	runtime := &Runtime{
		Runtime:    goja.New(),
		consoleLog: bytes.Buffer{},
	}

	if err := injectWindow(runtime, nil); err != nil {
		return nil, err
	}

	if err := injectFunctions(runtime); err != nil {
		return nil, err
	}

	i := 0
	randoms := []float64{
		0.4344425854257551,
		0.30677431741595645,

		0.6969584016327766,
		0.2998505516555874,
		0.3643531370591748,
		0.9201465817378611,
		0.5870992096697567,
		0.5135277503653166,
		0.9051229891129458,
		0.453226454916563,
		0.5522444392248091,
		0.6802094390059936,
		0.1332380778151574,
		0.008326238996670732,
	}

	// 创建一个对象用于保存 RegExp.$1-$9
	mathJS := runtime.Get("Math").(*goja.Object)
	// mathPrototype := mathJS.Get("prototype").(*goja.Object)
	originalRandom := mathJS.Get("random")
	mathJS.Set("random", func(call goja.FunctionCall) goja.Value {

		callable, ok := goja.AssertFunction(originalRandom)
		if !ok {
			return goja.Undefined()
		}
		result, err := callable(call.This, call.Arguments...)
		if err != nil {
			return goja.Undefined()
		}

		if i < len(randoms) {
			result = runtime.ToValue(randoms[i])
			i++
		}

		fmt.Println("Math.random", result.Export())

		return result
	})

	// RegExp.$1 写法兼容
	// 定义一个全局变量用于存储最近的正则匹配结果
	var lastMatch []string

	// 创建一个对象用于保存 RegExp.$1-$9
	regexpJS := runtime.Get("RegExp").(*goja.Object)
	for i := 1; i <= 9; i++ {
		prop := fmt.Sprintf("$%d", i)
		regexpJS.Set(prop, "")
	}

	// 包装 exec 方法来捕获正则匹配结果
	regexpPrototype := regexpJS.Get("prototype").(*goja.Object)
	originalExec := regexpPrototype.Get("exec")

	newExec := func(call goja.FunctionCall) goja.Value {
		// 调用原始 exec 方法
		callable, ok := goja.AssertFunction(originalExec)
		if !ok {
			return goja.Undefined()
		}
		result, err := callable(call.This, call.Arguments...)
		if err != nil {
			return goja.Undefined()
		}
		if result == goja.Null() || result == nil {
			// 如果没有匹配结果，清空 lastMatch
			lastMatch = nil
			for i := 1; i <= 9; i++ {
				prop := fmt.Sprintf("$%d", i)
				regexpJS.Set(prop, "")
			}
			return result
		}

		// 如果有匹配结果，保存到 lastMatch，并更新静态属性
		matchArray := result.(*goja.Object)
		lastMatch = make([]string, 10)
		for i := 1; i <= 9; i++ {
			prop := fmt.Sprintf("$%d", i)
			value := matchArray.Get(fmt.Sprintf("%d", i))
			if value != nil && value != goja.Undefined() {
				lastMatch[i] = value.String()
				regexpJS.Set(prop, value.String())
			} else {
				lastMatch[i] = ""
				regexpJS.Set(prop, "")
			}
		}

		return result
	}
	regexpPrototype.Set("test", newExec)
	regexpPrototype.Set("exec", newExec)

	_, err := runtime.RunString(string(`
		window = {...window};
	`))
	if err != nil {
		return nil, err
	}

	return runtime, nil
}

// WaitForPromise 等待 JS Promise 完成并返回其结果
func WaitForPromise(vm *goja.Runtime, promise goja.Value) (goja.Value, error) {
	// 创建一个 Go 通道，用于等待 Promise 的结果
	resultChan := make(chan goja.Value)
	errorChan := make(chan error)

	// 定义 then 回调函数
	thenFunc := func(call goja.FunctionCall) goja.Value {
		// 获取 Promise 的解析值（resolved value）
		result := call.Argument(0)
		// 发送结果到通道
		go func() {
			resultChan <- result
		}()
		return nil
	}

	// 定义 catch 回调函数
	catchFunc := func(call goja.FunctionCall) goja.Value {
		// 获取 Promise 的拒绝值（rejected reason）
		err := call.Argument(0)
		// 发送错误到通道
		go func() {
			errorChan <- fmt.Errorf("%v", err)
		}()
		return nil
	}

	// 获取 Promise 的 then 和 catch 方法
	then, ok := goja.AssertFunction(promise.ToObject(vm).Get("then"))
	if !ok {
		return nil, fmt.Errorf("value is not a Promise")
	}
	catch, ok := goja.AssertFunction(promise.ToObject(vm).Get("catch"))
	if !ok {
		return nil, fmt.Errorf("value is not a Promise")
	}

	// 调用 then 和 catch 方法
	_, err := then(promise, vm.ToValue(thenFunc))
	if err != nil {
		return nil, err
	}
	_, err = catch(promise, vm.ToValue(catchFunc))
	if err != nil {
		return nil, err
	}

	// 等待结果或错误
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return nil, err
	}
}
