package slimapi

import (
	"github.com/cmstar/go-webapi"
	"github.com/cmstar/go-webapi/logfunc"
)

// NewSlimApiApiLogger 返回用于 SlimAPI 协议的 [webapi.ApiLogger] 实现。
func NewSlimApiApiLogger() webapi.LogFuncPipeline {
	logBody := func(state *webapi.ApiState) {
		body := getBufferedBody(state)
		if len(body) > 0 {
			state.LogMessage = append(state.LogMessage,
				"Length", len(body),
				"Body", body,
			)
		}
	}

	return webapi.NewLogFuncPipeline(
		logfunc.IP,
		logfunc.URL,
		logBody,
		logfunc.Files,
		logfunc.Error,
	)
}
