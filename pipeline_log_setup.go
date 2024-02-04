package webapi

import "github.com/cmstar/go-logx"

// LogSetup 定义一个过程，此过程用于向 [ApiState] 填充日志信息。
type LogSetup interface {
	// Setup 可将日志信息写入 [ApiState.LogLevel] 和 [ApiState.LogMessage] 。
	Setup(state *ApiState)
}

// LogSetupFunc 用于将函数 [LogSetup.Setup] 。
type LogSetupFunc func(state *ApiState)

// DecodeArg implements [ArgumentDecoder.DecodeArg].
func (f LogSetupFunc) Setup(state *ApiState) {
	f(state)
}

// LogSetupPipeline 是 [LogSetup] 组成的管道，实现 [ApiLogger] 。
//
// 在 [ApiLogger.Log] 时，依次执行每个 [LogSetup.Setup] ，并将得到的 [ApiState.LogLevel] 和 [ApiState.LogMessage] 输出到日志。
// 若 [LogLevel] 未被设置，默认使用 [logx.LevelInfo] 级别。
type LogSetupPipeline []LogSetup

var _ ApiLogger = (*LogSetupPipeline)(nil)

// NewLogSetupPipeline 返回一个 [LogSetupPipeline] 。
func NewLogSetupPipeline(s ...LogSetup) LogSetupPipeline {
	return LogSetupPipeline(s)
}

// Log implements [ApiLogger.Log].
func (p LogSetupPipeline) Log(state *ApiState) {
	logger := state.Logger
	if logger == nil || len(p) == 0 {
		return
	}

	for _, v := range p {
		v.Setup(state)
	}

	if state.LogLevel == 0 {
		state.LogLevel = logx.LevelInfo
	}

	logger.Log(state.LogLevel, "", state.LogMessage...)
}
