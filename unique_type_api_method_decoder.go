package webapi

import (
	"reflect"
)

// ApiMethodArgDecodeFunc 定义一个过程，此过程用于从 ApiState 中解析得到 API 方法的特定参数的值。
// 一组 ApiMethodArgDecodeFunc 形成一个解析 API 方法中每个参数的管道：
// 若给定的 API 参数（通过 index 和 argType 识别）可被当前函数解析，则返回 ok=true 及解析结果 v ，或者返回 ok=false 及解析错误；
// 若当前函数不支持给定参数的解析，则返回无错误的 ok=false 和 v=nil 。
type ApiMethodArgDecodeFunc func(state *ApiState, index int, argType reflect.Type) (ok bool, v interface{}, err error)

type uniqueApiMethodDecoder struct {
	fs []ApiMethodArgDecodeFunc
}

// NewUniqueTypeApiMethodDecoder 返回一个 ApiDecoder 的实现。此实现要求被调用的每个方法，其参数表中的参数类型是不重复的。
// 可通过此方法组装解析过程，进而实现一个 ApiDecoder 。
//
// decodeFuncs 给定各种类型参数的解析过程。它形成一个管道，当一个 API 方法被调用时，每个参数依次通过此管道中的每个函数，
// 当首次遇到返回 ok=true 时，参数取用该过程的值。
//
// 管道中的第一个元素是预定义的，其用于解析 *webapi.ApiState 并赋值。 decodeFuncs 会追加此过程后面。
//
func NewUniqueTypeApiMethodDecoder(decodeFuncs ...ApiMethodArgDecodeFunc) ApiDecoder {
	apiStateDecodeFunc := func(state *ApiState, index int, argType reflect.Type) (bool, interface{}, error) {
		// ApiState 必须用指针获取，不允许用值类型。
		if argType == reflect.TypeOf(state).Elem() {
			PanicApiError(state, nil, "method '%s' arg%d %v: must be a pointer", state.Name, index, argType)
		}

		if argType != reflect.TypeOf(state) {
			return false, nil, nil
		}
		return true, state, nil
	}

	fs := []ApiMethodArgDecodeFunc{apiStateDecodeFunc}
	fs = append(fs, decodeFuncs...)

	return &uniqueApiMethodDecoder{
		fs: fs,
	}
}

// Decode implements ApiDecoder.Decode.
func (d *uniqueApiMethodDecoder) Decode(state *ApiState) {
	methodType := state.Method.Value.Type()
	numIn := methodType.NumIn()
	args := make([]reflect.Value, 0, numIn)

	for i := 0; i < numIn; i++ {
		argType := methodType.In(i)

		// 参数表里一种类型只能出现一次。
		for j := 0; j < len(args); j++ {
			if args[j].Type() == argType {
				PanicApiError(state, nil, "method '%s' arg%d %v: argument type cannot be duplicated", state.Name, i, argType)
			}
		}

		hit := false
		for j := 0; j < len(d.fs); j++ {
			argFunc := d.fs[j]
			ok, v, err := argFunc(state, i, argType)
			if err != nil {
				state.Error = err
				return
			}

			if !ok {
				continue
			}

			if v == nil {
				PanicApiError(state, nil, "method '%s' arg%d %v: value is nil", state.Name, i, argType)
			}

			args = append(args, reflect.ValueOf(v))
			hit = true
			break
		}

		if !hit {
			PanicApiError(state, nil, "method '%s' arg%d %v: not supported", state.Name, i, argType)
		}
	}

	state.Args = args
}
