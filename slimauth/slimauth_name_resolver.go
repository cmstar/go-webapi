package slimauth

import (
	"fmt"

	"github.com/cmstar/go-webapi"
	"github.com/cmstar/go-webapi/slimapi"
)

type slimAuthApiNameResolver struct {
	secretFinder SecretFinderFunc
	timeChecker  TimeCheckerFunc
	raw          webapi.ApiNameResolver
}

// NewSlimAuthApiNameResolver 返回 SlimAuth 协议的 [webapi.ApiNameResolver] 。
// 它增加了签名校验，其余都和 SlimAPI 一样。
func NewSlimAuthApiNameResolver(secretFinder SecretFinderFunc, timeChecker TimeCheckerFunc) webapi.ApiNameResolver {
	if secretFinder == nil {
		panic("secretFinder must be provided")
	}

	if timeChecker == nil {
		panic("timeChecker must be provided")
	}

	return &slimAuthApiNameResolver{
		secretFinder: secretFinder,
		timeChecker:  timeChecker,
		raw:          slimapi.NewSlimApiNameResolver(),
	}
}

// FillMethod implements [webapi.ApiNameResolver.FillMethod].
func (x slimAuthApiNameResolver) FillMethod(state *webapi.ApiState) {
	x.verifySignature(state)
	x.raw.FillMethod(state)
}

func (x slimAuthApiNameResolver) verifySignature(state *webapi.ApiState) {
	r := state.RawRequest
	auth, err := ParseAuthorizationHeader(r)
	if err != nil {
		panic(webapi.CreateBadRequestError(state, err, "invalid Authorization"))
	}

	// 存起来，用于后续方法参数的赋值和日志输出。
	SetBufferedAuthorization(state, auth)

	// 签名算法目前就一个版本，不允许出现其他值。
	if auth.Version != DefaultSignVersion {
		panic(webapi.CreateBadRequestError(state, err, "unsupported signature version"))
	}

	secret := x.secretFinder(auth.Key)
	if secret == "" {
		panic(webapi.CreateBadRequestError(state, nil, "unknown key"))
	}

	// 后续走 SlimAPI 的 decode 过程，需要重读 body 。
	signResult := Sign(r, true, secret, auth.Timestamp)

	// 时间戳校验。
	timeCheckErr := x.timeChecker(auth.Timestamp)
	if timeCheckErr != nil {
		panic(webapi.CreateBadRequestError(state, timeCheckErr, "timestamp error"))
	}

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
}
