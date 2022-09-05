package slimauth

import (
	"github.com/cmstar/go-webapi"
	"github.com/cmstar/go-webapi/logsetup"
)

// LogAuthorization 实现 [webapi.LogSetup] ，用于记录请求的 Authorization 头的相关信息。
//
// 这是一个单例。
var LogAuthorization = logAuthorization{}

type logAuthorization struct{}

var _ webapi.LogSetup = (*logAuthorization)(nil)

func (logAuthorization) Setup(state *webapi.ApiState) {
	auth, ok := GetBufferedAuthorization(state)
	if ok {
		state.LogMessage = append(state.LogMessage,
			"AccessKey", auth.Key,
			"Timestamp", auth.Timestamp,
		)
	}
}

// NewSlimAuthApiLogger 返回用于 SlimAuth 协议的 [webapi.ApiLogger] 实现。
func NewSlimAuthApiLogger() webapi.LogSetupPipeline {
	return webapi.NewLogSetupPipeline(
		logsetup.IP,
		logsetup.URL,
		LogAuthorization,
		logsetup.Error,
	)
}
