package gojs

import (
	"fmt"
	"sync"
	"time"

	"github.com/dop251/goja"
)

func injectFunctions(runtime *Runtime) error {
	manager := NewTimeoutManager(runtime.Runtime)

	// 注册 setTimeout 和 clearTimeout 到全局对象
	runtime.Set("setTimeout", manager.SetTimeout)
	runtime.Set("clearTimeout", manager.ClearTimeout)

	// 定义 console 对象
	console := map[string]interface{}{
		"log": func(call goja.FunctionCall) goja.Value {
			output := ""
			// 将输出写入到缓冲区
			for _, arg := range call.Arguments {
				output += arg.String() + " "
			}
			output += "\n"

			fmt.Print(output)
			runtime.consoleLog.WriteString(output)
			return goja.Undefined()
		},
	}

	runtime.Set("console", console)

	return nil
}

// TimeoutManager 结构体管理定时器
type TimeoutManager struct {
	timers  map[int]*time.Timer
	mutex   sync.Mutex
	nextID  int
	runtime *goja.Runtime
}

// NewTimeoutManager 创建一个新的 TimeoutManager
func NewTimeoutManager(runtime *goja.Runtime) *TimeoutManager {
	return &TimeoutManager{
		timers:  make(map[int]*time.Timer),
		nextID:  1,
		runtime: runtime,
	}
}

// SetTimeout 实现 setTimeout
func (tm *TimeoutManager) SetTimeout(call goja.FunctionCall) goja.Value {
	callback := call.Argument(0) // 第一个参数是回调函数
	_, ok := goja.AssertFunction(callback)
	if !ok {
		panic(tm.runtime.NewTypeError("First argument must be a function"))
	}

	delay := call.Argument(1).ToInteger() // 第二个参数是延迟时间（毫秒）
	if delay < 0 {
		delay = 0
	}

	id := tm.nextID
	tm.nextID++

	// callable, ok := goja.AssertFunction(callback)
	// if !ok {
	// 	fmt.Printf("Error in setTimeout callback: %v\n", "not function")
	// }

	// // 执行回调函数
	// _, err := callable(goja.Undefined())
	// if err != nil {
	// 	fmt.Printf("Error in setTimeout callback: %v\n", err)
	// }

	timer := time.AfterFunc(time.Duration(delay)*time.Millisecond, func() {
		tm.mutex.Lock()
		defer tm.mutex.Unlock()

		callback := call.Argument(0) // 第一个参数是回调函数
		callable, ok := goja.AssertFunction(callback)
		if !ok {
			return
		}

		// 执行回调函数
		_, err := callable(goja.Undefined())
		if err != nil {
			fmt.Printf("Error in setTimeout callback: %v\n", err)
		}

		// 移除已完成的定时器
		delete(tm.timers, id)
	})

	tm.mutex.Lock()
	tm.timers[id] = timer
	tm.mutex.Unlock()

	// 返回定时器的 ID
	return tm.runtime.ToValue(id)
}

// ClearTimeout 实现 clearTimeout
func (tm *TimeoutManager) ClearTimeout(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) == 0 {
		return goja.Undefined()
	}
	id := int(call.Argument(0).ToInteger())

	tm.mutex.Lock()
	timer, exists := tm.timers[id]
	if exists {
		timer.Stop()
		delete(tm.timers, id)
	}
	tm.mutex.Unlock()

	return goja.Undefined()
}
