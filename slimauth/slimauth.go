// slimauth 是带有签名校验逻辑的 SlimAPI 协议的扩展。
package slimauth

import (
	"github.com/cmstar/go-webapi"
	"github.com/cmstar/go-webapi/slimapi"
)

const (
	// 默认的签名算法版本，当 Authorization 头没有写 Version 字段时，默认为此版本。
	DefaultSignVersion = 1

	// SlimAuth 协议在 HTTP Authorization 头的 <scheme> 部分，固定值。
	AuthorizationScheme = "SLIM-AUTH"

	// HTTP 协议的 Authorization 头。
	HttpHeaderAuthorization = "Authorization"

	// 用于缓存 Authorization 参数的 key 。
	_authorizationArgumentKey = "slimauthAuthorizationData"
)

// SecretFinder 用于获取绑定到指定 accessKey 的 secret 。
type SecretFinder interface {
	// GetSecret 获取绑定到指定 accessKey 的 secret 。
	// 若给定的 accessKey 没有绑定，返回空字符串。
	// 若获取过程出错，直接 panic 。
	GetSecret(accessKey string) string
}

type secretFinderWrapper struct {
	f func(accessKey string) string
}

func (x secretFinderWrapper) GetSecret(accessKey string) string {
	return x.f(accessKey)
}

// SecretFinderFunc 将给定的函数包装为 [SecretFinder] 。
func SecretFinderFunc(f func(accessKey string) string) SecretFinder {
	return secretFinderWrapper{f}
}

// NewSlimAuthApiHandler 创建 SlimAuth 协议的 [webapi.ApiHandler] 。
func NewSlimAuthApiHandler(name string, finder SecretFinder) *webapi.ApiHandlerWrapper {
	h := slimapi.NewSlimApiHandler(name)
	h.ApiNameResolver = NewSlimAuthApiNameResolver(finder)
	h.ApiDecoder = NewSlimAuthApiDecoder()
	h.ApiResponseWriter = &slimAuthApiResponseWriter{h.ApiResponseWriter}
	h.ApiLogger = NewSlimAuthApiLogger()
	return h
}

// SlimAuth 协议的 ApiResponseWriter 。除了将 Code 介于1-999时将其作为 HTTP 状态码返回，其余都和 SlimAPI 一样。
type slimAuthApiResponseWriter struct {
	raw webapi.ApiResponseWriter
}

func (x slimAuthApiResponseWriter) WriteResponse(state *webapi.ApiState) {
	x.raw.WriteResponse(state)

	if state.Response.Code > 0 && state.Response.Code < 1000 {
		state.RawResponse.WriteHeader(state.Response.Code)
	}
}

// 获取当前请求中缓存的 [Authorization] 。若值不存在， panic 。
// 在 [webapi.ApiNameResolver.FillMethod] 发生后，被解析到的 [Authorization] 将被缓存。
func MustGetBufferedAuthorization(state *webapi.ApiState) Authorization {
	v, ok := GetBufferedAuthorization(state)
	if !ok {
		webapi.PanicApiError(state, nil, "Authorization not set, there may be a bug.")
	}
	return v
}

// 获取当前请求中缓存的 [Authorization] 。若值不存在，返回默认值及 ok=false 。
// 在 [webapi.ApiNameResolver.FillMethod] 发生后，被解析到的 [Authorization] 将被缓存。
func GetBufferedAuthorization(state *webapi.ApiState) (auth Authorization, ok bool) {
	v, ok := state.GetCustomData(_authorizationArgumentKey)
	if !ok {
		return Authorization{}, false
	}
	return v.(Authorization), true
}

// 缓存解析到的 [Authorization] 。
func SetBufferedAuthorization(state *webapi.ApiState, auth Authorization) {
	state.SetCustomData(_authorizationArgumentKey, auth)
}
