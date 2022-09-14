package webapi

import (
	"reflect"
)

// ArgumentDecoder 定义一个过程，此过程用于从 [ApiState] 中解析得到 API 方法的特定参数的值。
// 一组实例形成一个解析 API 方法中每个参数的管道 [ArgumentDecoderPipeline] 。
type ArgumentDecoder interface {
	// DecodeArg 尝试机械 API 方法中的一个参数。
	//
	// 若给定的 API 参数（通过 index 和 argType 识别）可被当前函数解析，则返回 ok=true 及解析结果 v ，或者返回 ok=false 及解析错误；
	// 若当前函数不支持给定参数的解析，则返回无错误的 ok=false 和 v=nil 。
	DecodeArg(state *ApiState, index int, argType reflect.Type) (ok bool, v any, err error)
}

// ArgumentDecodeFunc 是 [ArgumentDecoder.DecodeArg] 的函数签名。
type ArgumentDecodeFunc func(state *ApiState, index int, argType reflect.Type) (ok bool, v any, err error)

type argumentDecoderWrap struct {
	f func(state *ApiState, index int, argType reflect.Type) (ok bool, v any, err error)
}

// ToArgumentDecoder 将 [ArgumentDecodeFunc] 包装成 [ArgumentDecoder] 。
func ToArgumentDecoder(f ArgumentDecodeFunc) ArgumentDecoder {
	return argumentDecoderWrap{f}
}

// DecodeArg implements [ArgumentDecoder.DecodeArg].
func (x argumentDecoderWrap) DecodeArg(state *ApiState, index int, argType reflect.Type) (ok bool, v any, err error) {
	return x.f(state, index, argType)
}

// ApiStateDecodeFunc 是一个 [ArgumentDecoder] ，它用于解析并赋值 [*ApiState] 。
//
// 这是一个单例。
var ApiStateArgumentDecoder = apiStateArgumentDecoder{}

type apiStateArgumentDecoder struct{}

var _ ArgumentDecoder = (*apiStateArgumentDecoder)(nil)

func (apiStateArgumentDecoder) DecodeArg(state *ApiState, index int, argType reflect.Type) (ok bool, v any, err error) {
	// ApiState 必须用指针获取，不允许用值类型。
	if argType == reflect.TypeOf(state).Elem() {
		PanicApiError(state, nil, "method '%s' arg%d %v: must be a pointer", state.Name, index, argType)
	}

	if argType != reflect.TypeOf(state) {
		return false, nil, nil
	}
	return true, state, nil
}

// ArgumentDecoderPipeline 是 [ArgumentDecoder] 组成的管道。
// 实现 [ApiDecoder] ，此实现要求被调用的每个方法，其参数表中的参数类型是不重复的。
//
// 在 [ApiDecoder.Decode] 时，将依次执行管道内的每个 [ArgumentDecoder.DecodeArg] 。
// 可以通过增减和调整元素的顺序定制执行的过程。
type ArgumentDecoderPipeline []ArgumentDecoder

var _ ApiDecoder = (*ArgumentDecoderPipeline)(nil)

// NewArgumentDecoderPipeline 返回一个 [ArgumentDecoderPipeline] 。
// 其第一个元素是预定义的 [ApiStateDecodeFunc] ，用于赋值 [*ApiState] ； decodeFuncs 会追加在后面。
func NewArgumentDecoderPipeline(d ...ArgumentDecoder) ArgumentDecoderPipeline {
	p := make([]ArgumentDecoder, 0, len(d)+1)
	p = append(p, ApiStateArgumentDecoder)
	p = append(p, d...)
	return p
}

// Decode implements ApiDecoder.Decode.
func (d ArgumentDecoderPipeline) Decode(state *ApiState) {
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
			d := d[j]
			ok, v, err := d.DecodeArg(state, i, argType)
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
