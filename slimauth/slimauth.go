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

// SecretFinderFunc 用于获取绑定到指定 accessKey 的 secret 。
// 若给定的 accessKey 没有绑定，返回空字符串。
// 若获取过程出错，直接 panic ，其错误处理方式与普通的 API 方法一致。
type SecretFinderFunc func(accessKey string) string

// SlimAuthApiHandlerOption 用于初始化 SlimAuth 协议的 [webapi.ApiHandler] 。
type SlimAuthApiHandlerOption struct {
	// 名称。
	Name string

	// 用于查找签名所需的 secret 。必须提供。
	SecretFinder SecretFinderFunc

	// 用于校验签名信息中携带的时间戳的有效性。
	// 若为 nil ，将自动使用 [DefaultTimeChecker] ；若不需要校验，可给定 [NoTimeChecker] 。
	TimeChecker TimeCheckerFunc
}

// NewSlimAuthApiHandler 创建 SlimAuth 协议的 [webapi.ApiHandler] 。
func NewSlimAuthApiHandler(op SlimAuthApiHandlerOption) *webapi.ApiHandlerWrapper {
	timeChecker := op.TimeChecker
	if timeChecker == nil {
		timeChecker = DefaultTimeChecker
	}

	h := slimapi.NewSlimApiHandler(op.Name)
	h.ApiNameResolver = NewSlimAuthApiNameResolver(op.SecretFinder, timeChecker)
	h.ApiDecoder = NewSlimAuthApiDecoder()
	h.ApiLogger = NewSlimAuthApiLogger()
	return h
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
