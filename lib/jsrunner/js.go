//go:build js && wasm

package jsrunner

import (
	"context"
	"fmt"
	"sync"
	"syscall/js"
)

var (
	instance JSRunner
	once     sync.Once
)

type jsRunner struct {
	global js.Value
}

type jsValue struct {
	val js.Value
}

func NewJSRunner() JSRunner {
	once.Do(func() {
		instance = &jsRunner{global: js.Global()}
	})
	return instance
}

func (j *jsRunner) Engine() Engine {
	return Native
}

func (j *jsRunner) MustGet(key string) (JSValue, error) {
	result := j.global.Get(key)
	if result.IsUndefined() {
		return nil, fmt.Errorf("key %q not found in global scope", key)
	}
	defer j.global.Set(key, js.Undefined())
	return &jsValue{val: result}, nil
}

func (j *jsRunner) RunString(code string) (_ JSValue, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	result := j.global.Call("eval", code)
	return &jsValue{val: result}, nil
}

func (v *jsValue) String() string {
	switch v.val.Type() {
	case js.TypeString:
		return v.val.String()
	default:
		return v.val.Call("toString").String()
	}
}

func (v *jsValue) Export() interface{} {
	switch v.val.Type() {
	case js.TypeString:
		return v.val.String()
	case js.TypeNumber:
		return v.val.Float()
	case js.TypeBoolean:
		return v.val.Bool()
	case js.TypeObject:
		if v.val.InstanceOf(js.Global().Get("Array")) {
			length := v.val.Length()
			arr := make([]interface{}, length)
			for i := 0; i < length; i++ {
				arr[i] = (&jsValue{val: v.val.Index(i)}).Export()
			}
			return arr
		}
		obj := make(map[string]interface{})
		keys := js.Global().Get("Object").Call("keys", v.val)
		length := keys.Length()
		for i := 0; i < length; i++ {
			key := keys.Index(i).String()
			val := v.val.Get(key)
			obj[key] = (&jsValue{val: val}).Export()
		}
		return obj
	default:
		return nil
	}
}

// TODO probably don't need
func (j *jsRunner) NewObject() JSObject {
	return &jsValue{val: js.Global().Get("Object").New()}
}

func (j *jsRunner) Set(name string, value interface{}) error {
	if name == "console" {
		js.Global().Set("console", js.Global().Get("console"))
		return nil
	}
	if jsObj, ok := value.(JSObject); ok {
		jsVal := jsObj.(*jsValue)
		js.Global().Set(name, jsVal.val)
		return nil
	}

	js.Global().Set(name, js.ValueOf(value))
	return nil
}

func (j *jsRunner) WaitPromise(ctx context.Context, val JSValue) (interface{}, error) {
	// Check if the value is actually a promise
	jsVal := val.(*jsValue)

	// If it's not a promise, just export it
	if !jsVal.val.InstanceOf(js.Global().Get("Promise")) {
		return val.Export(), nil
	}

	// For WASM environment, we need to handle promises differently
	// We'll use a JavaScript approach to wait for the promise
	waitCode := `
		(function() {
			var promise = arguments[0];
			var result = null;
			var error = null;
			var resolved = false;
			
			promise.then(function(value) {
				result = value;
				resolved = true;
			}).catch(function(err) {
				error = err;
				resolved = true;
			});
			
			// Wait for the promise to resolve
			var startTime = Date.now();
			while (!resolved && (Date.now() - startTime) < 10000) {
				// Small delay to allow promise resolution
				var dummy = 0;
				for (var i = 0; i < 1000; i++) {
					dummy += i;
				}
			}
			
			if (!resolved) {
				throw new Error("Promise timeout");
			}
			
			if (error) {
				throw error;
			}
			
			return result;
		})
	`

	waitFunc := js.Global().Call("eval", waitCode)
	result := waitFunc.Invoke(jsVal.val)

	// Check if the result is an error
	if result.Type() == js.TypeObject && !result.Get("message").IsUndefined() {
		return nil, fmt.Errorf("JavaScript error: %s", result.Get("message").String())
	}

	return (&jsValue{val: result}).Export(), nil
}
