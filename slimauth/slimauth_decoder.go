package slimauth

import (
	"reflect"

	"github.com/cmstar/go-webapi"
	"github.com/cmstar/go-webapi/slimapi"
)

// NewSlimAuthApiDecoder 返回 SlimAuth 协议的 [webapi.ApiDecoder] 。
func NewSlimAuthApiDecoder() webapi.ApiDecoder {
	return webapi.NewArgumentDecoderPipeline(
		authorizationArgumentDecoder{},
		slimapi.StructArgumentDecoder,
	)
}

// 用于解析方法上的 Authorization 参数，也可以是其指针。
type authorizationArgumentDecoder struct{}

func (x authorizationArgumentDecoder) DecodeArg(state *webapi.ApiState, index int, argType reflect.Type) (ok bool, v interface{}, err error) {
	isPtr := argType.Kind() == reflect.Ptr
	if isPtr {
		argType = argType.Elem()
	}

	if argType != reflect.TypeOf(Authorization{}) {
		return
	}

	v = MustGetBufferedAuthorization(state)
	ok = true

	if isPtr {
		t := v.(Authorization)
		v = &t
	}

	return
}
