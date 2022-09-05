package slimapi

import (
	"bytes"
	"encoding/json"

	"github.com/cmstar/go-webapi"
)

// slimApiResponseWriter 实现 SlimAPI 的 webapi.ApiResponseWriter 。
type slimApiResponseWriter struct {
}

// NewSlimApiResponseWriter 返回用于 SlimAPI 协议的 webapi.ApiResponseWriter 实现。
// 该实现是无状态且线程安全的。
func NewSlimApiResponseWriter() webapi.ApiResponseWriter {
	return &slimApiResponseWriter{}
}

// WriteResponse 实现 webapi.ApiResponseWriter.WriteResponse 。
func (*slimApiResponseWriter) WriteResponse(state *webapi.ApiState) {
	/*
	 * GO 的字符串都是 UTF-8 编码，和 SlimAPI 的要求一致，没有转码需要。
	 * ApiState.ResponseContentType 应在 slimApiNameResolver 中完成初始化，这里不用再处理。
	 */

	state.MustHaveResponse()

	// 序列化可能报错，放在前面先处理。
	jsonBody, err := json.Marshal(&state.Response)
	if err != nil {
		webapi.PanicApiError(state, err, "json encoding error")
	}

	buf := new(bytes.Buffer)

	// -> callback(
	callback := getCallback(state)
	if callback != "" {
		buf.WriteString(callback)
		buf.WriteByte('(')
	}

	// -> callback(body
	buf.Write(jsonBody)

	// -> callback(body)
	if callback != "" {
		buf.WriteByte(')')
	}

	state.ResponseBody = buf
}
