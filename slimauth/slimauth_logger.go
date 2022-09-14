package slimauth

import (
	"github.com/cmstar/go-webapi"
	"github.com/cmstar/go-webapi/slimapi"
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
	// 除了多记录一个 Authorization 头的内容，其他都是 SlimAPI 一样。
	pipe := slimapi.NewSlimApiLogger()

	// 按相关性， Authorization 头的内容适合放在 body 前面。
	// 尝试插到管道里 LogBody 这一截的前面。
	i := 0
	found := false
	for ; i < len(pipe); i++ {
		if pipe[i] == slimapi.LogBody {
			// 将元素依次往后挪一格，留一个格子用于插入。
			pipe = append(pipe[:i+1], pipe[i:]...)
			found = true
			break
		}
	}

	if found {
		pipe[i] = LogAuthorization
	} else {
		pipe = append(pipe, LogAuthorization)
	}

	return pipe
}
