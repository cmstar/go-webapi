package webapi

import (
	"reflect"
)

// DecodeFunc 定义一个过程，此过程用于从 [ApiState] 中解析得到 API 方法的特定参数的值。
// 一组 [DecodeFunc] 形成一个解析 API 方法中每个参数的管道：
// 若给定的 API 参数（通过 index 和 argType 识别）可被当前函数解析，则返回 ok=true 及解析结果 v ，或者返回 ok=false 及解析错误；
// 若当前函数不支持给定参数的解析，则返回无错误的 ok=false 和 v=nil 。
type DecodeFunc func(state *ApiState, index int, argType reflect.Type) (ok bool, v interface{}, err error)

// ApiStateDecodeFunc 是一个 [DecodeFunc] ，它用于解析并赋值 [*ApiState] 。
func ApiStateDecodeFunc(state *ApiState, index int, argType reflect.Type) (ok bool, v interface{}, err error) {
	// ApiState 必须用指针获取，不允许用值类型。
	if argType == reflect.TypeOf(state).Elem() {
		PanicApiError(state, nil, "method '%s' arg%d %v: must be a pointer", state.Name, index, argType)
	}

	if argType != reflect.TypeOf(state) {
		return false, nil, nil
	}
	return true, state, nil
}

// DecodeFuncPipeline 是 [DecodeFunc] 组成的管道。
// 实现 [ApiDecoder] ，此实现要求被调用的每个方法，其参数表中的参数类型是不重复的。
//
// 在 [ApiDecoder.Decode] 时，将依次执行管道内的每个 [DecodeFunc] 。
// 可以通过增减和调整元素的顺序定制执行的过程。
type DecodeFuncPipeline []DecodeFunc

var _ ApiDecoder = (*DecodeFuncPipeline)(nil)

// NewDecodeFuncPipeline 返回一个 [DecodeFuncPipeline] 。
// 其第一个元素是预定义的 [ApiStateDecodeFunc] ，用于赋值 [*ApiState] ； decodeFuncs 会追加在后面。
func NewDecodeFuncPipeline(decodeFuncs ...DecodeFunc) DecodeFuncPipeline {
	p := make([]DecodeFunc, 0, len(decodeFuncs)+1)
	p = append(p, ApiStateDecodeFunc)
	p = append(p, decodeFuncs...)
	return p
}

// Decode implements ApiDecoder.Decode.
func (d DecodeFuncPipeline) Decode(state *ApiState) {
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
		for j := 0; j < len(d); j++ {
			argFunc := d[j]
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
