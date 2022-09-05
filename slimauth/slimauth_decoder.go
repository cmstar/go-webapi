package slimauth

import (
	"fmt"
	"reflect"

	"github.com/cmstar/go-webapi"
	"github.com/cmstar/go-webapi/slimapi"
)

type slimAuthApiDecoder struct {
	finder SecretFinder
	pipe   webapi.ArgumentDecoderPipeline
}

// NewSlimAuthApiDecoder 返回 SlimAuth 协议的 [webapi.ApiDecoder] 。它增加了签名校验，其余都和 SlimAPI 一样。
func NewSlimAuthApiDecoder(finder SecretFinder) webapi.ApiDecoder {
	if finder == nil {
		panic("finder must be provided")
	}

	decoder := &slimAuthApiDecoder{
		finder: finder,
		pipe:   make(webapi.ArgumentDecoderPipeline, 0, 2),
	}

	decoder.pipe = append(decoder.pipe, webapi.ToArgumentDecoder(decoder.decodeAuthorizationArg))
	decoder.pipe = append(decoder.pipe, slimapi.NewSlimApiDecoder()...)

	return decoder
}

// 解析方法上的 Authorization 参数，也可以是其指针。
func (x slimAuthApiDecoder) decodeAuthorizationArg(state *webapi.ApiState, index int, argType reflect.Type) (ok bool, v interface{}, err error) {
	isPtr := argType.Kind() == reflect.Ptr
	if isPtr {
		argType = argType.Elem()
	}

	if argType != reflect.TypeOf(Authorization{}) {
		return
	}

	v = GetBufferedAuthorization(state)
	ok = true

	if isPtr {
		t := v.(Authorization)
		v = &t
	}

	return
}

// Decode implements [webapi.ApiDecoder.Decode].
func (x slimAuthApiDecoder) Decode(state *webapi.ApiState) {
	x.verifySignature(state)
	x.pipe.Decode(state)
}

func (x slimAuthApiDecoder) verifySignature(state *webapi.ApiState) {
	r := state.RawRequest
	auth, err := ParseAuthorizationHeader(r)
	if err != nil {
		panic(webapi.CreateBadRequestError(state, err, "invalid Authorization"))
	}

	// 签名算吧目前就一个版本，不允许出现其他值。
	if auth.Version != DefaultSignVersion {
		panic(webapi.CreateBadRequestError(state, err, "unsupported signature version"))
	}

	secret := x.finder.GetSecret(auth.Key)
	if secret == "" {
		panic(webapi.CreateBadRequestError(state, nil, "unknown key"))
	}

	// 后续走 SlimAPI 的 decode 过程，需要重读 body 。
	signResult := Sign(r, true, secret, auth.Timestamp)

	switch signResult.Type {
	case SignResultType_MissingContentType:
		panic(webapi.CreateBadRequestError(state, signResult.Cause, "missing Content-Type"))

	case SignResultType_UnsupportedContentType:
		panic(webapi.CreateBadRequestError(state, signResult.Cause, "unsupported Content-Type"))

	case SignResultType_InvalidFormData:
		panic(webapi.CreateBadRequestError(state, signResult.Cause, "invalid form data"))
	}

	if signResult.Sign != auth.Sign {
		err = fmt.Errorf("signature mismatch, want %s, got %s", signResult.Sign, auth.Sign)
		panic(webapi.CreateBadRequestError(state, err, "signature error"))
	}

	// 存起来，用于后续方法参数的赋值。
	SetBufferedAuthorization(state, auth)
}
