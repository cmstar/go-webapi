package webapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

/*
目前 mux 部分直接基于 chi 库的，直接调用 chi 的方法即可。
*/

// GetRouteParam 从给定的请求中获取指定名称的路由参数。参数不存在时，返回空字符串。
func GetRouteParam(r *http.Request, name string) string {
	return chi.URLParam(r, name)
}

// SetRouteParams 向当前请求中添加一组路由参数，返回追加参数后的请求。
// 若给定参数表为 nil 或不包含元素，则返回原始请求。
func SetRouteParams(r *http.Request, params map[string]string) *http.Request {
	routeParamLen := len(params)
	if routeParamLen == 0 {
		return r
	}

	chiCtx := chi.RouteContext(r.Context())
	if chiCtx == nil {
		paramNames := make([]string, 0, routeParamLen)
		paramValues := make([]string, 0, routeParamLen)
		for k, v := range params {
			paramNames = append(paramNames, k)
			paramValues = append(paramValues, v)
		}

		chiCtx = chi.NewRouteContext()
		chiCtx.URLParams = chi.RouteParams{
			Keys:   paramNames,
			Values: paramValues,
		}
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))
	} else {
		paramNames := chiCtx.URLParams.Keys
		paramValues := chiCtx.URLParams.Values
		for k, v := range params {
			paramNames = append(paramNames, k)
			paramValues = append(paramValues, v)
		}

		chiCtx.URLParams = chi.RouteParams{
			Keys:   paramNames,
			Values: paramValues,
		}
	}
	return r
}
