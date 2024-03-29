package slimapi

import (
	"github.com/cmstar/go-webapi"
	"github.com/cmstar/go-webapi/logsetup"
)

// LogBody 实现 [webapi.LogSetup] ，用于记录请求的 body 的相关信息。
//
// 这是一个单例。
var LogBody = logBody{}

type logBody struct{}

var _ webapi.LogSetup = (*logBody)(nil)

func (logBody) Setup(state *webapi.ApiState) {
	body := getRequestBodyDescription(state)
	if len(body) > 0 {
		state.LogMessage = append(state.LogMessage,
			"Length", len(body),
			"Body", body,
		)
	}
}

// NewSlimApiLogger 返回用于 SlimAPI 协议的 [webapi.ApiLogger] 实现。
func NewSlimApiLogger() webapi.LogSetupPipeline {
	return webapi.NewLogSetupPipeline(
		logsetup.IP,
		logsetup.URL,
		logsetup.ContentType,
		LogBody,
		logsetup.Error,
	)
}
