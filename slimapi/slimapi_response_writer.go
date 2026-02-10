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
func (x *slimApiResponseWriter) WriteResponse(state *webapi.ApiState) {
	if state.ResponseBody != nil {
		return
	}

	// 分为流式和非流式两种情况。如果是流式，则强制使用其自带的 Content-Type 和格式。
	streamingResponse, ok := state.Data.(webapi.StreamingResponse)
	if ok {
		x.writeStreamingResponse(state, streamingResponse)
		return
	} else {
		x.writeGenericResponse(state)
	}
}

func (x *slimApiResponseWriter) writeGenericResponse(state *webapi.ApiState) {
	/*
	 * GO 的字符串都是 UTF-8 编码，和 SlimAPI 的要求一致，没有转码需要。
	 * ApiState.ResponseContentType 应在 slimApiNameResolver 中完成初始化，这里仅做最后的防御。
	 */
	if state.ResponseContentType == "" {
		state.ResponseContentType = "text/plain"
	}

	response := x.buildJsonResponse(state, state.Data, state.Error)
	if response == nil {
		return
	}

	buf := new(bytes.Buffer)

	// -> callback(
	callback := getCallback(state)
	if callback != "" {
		buf.WriteString(callback)
		buf.WriteByte('(')
	}

	// -> callback(body
	buf.Write(response)

	// -> callback(body)
	if callback != "" {
		buf.WriteByte(')')
	}

	state.ResponseBody = func(yield func([]byte) bool) {
		yield(buf.Bytes())
	}
}

func (x *slimApiResponseWriter) writeStreamingResponse(state *webapi.ApiState, streaming webapi.StreamingResponse) {
	state.ResponseContentType = streaming.ContentType()

	state.ResponseBody = func(yield func([]byte) bool) {
		// 因为 ResponseBody 的迭代是串行的，这里可以复用同一个 buf 以提高性能。
		buf := new(bytes.Buffer)

		for data, err := range streaming.Iter() {
			if err != nil {
				state.Error = err
				// 并没有严格要求 error 必须是 StreamingResponse 的最后一段。故此处不需要 break 。
			}

			response := x.buildJsonResponse(state, data, err)
			if response == nil {
				continue
			}

			streaming.WriteJsonBlock(buf, response)
			seg := buf.Bytes()
			if !yield(seg) {
				break
			}

			buf.Reset()
		}
	}
}

func (x *slimApiResponseWriter) buildJsonResponse(state *webapi.ApiState, callResult any, callError error) []byte {
	response := state.Handler.BuildResponse(state, callResult, callError)
	if response == nil {
		return nil
	}

	b, err := json.Marshal(response)
	if err != nil {
		webapi.PanicApiError(state, err, "json encoding error")
	}

	return b
}
