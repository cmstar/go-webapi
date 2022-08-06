package webapi

import "github.com/cmstar/go-logx"

// LogFunc 定义一个过程，此过程用于从 [ApiState.LogMessage] 填充信息。
type LogFunc func(state *ApiState)

// LogFuncPipeline 是 [LogFunc] 组成的管道，实现 [ApiLogger] 。
//
// 在 [ApiLogger.Log] 时，依次执行每个 [LogFunc] ，并将得到的 [ApiState.LogMessage] 输出到日志。
// 若 [LogLevel] 未被设置，默认使用 [logx.LevelInfo] 级别。
type LogFuncPipeline []LogFunc

var _ ApiLogger = (*LogFuncPipeline)(nil)

// NewLogFuncPipeline 返回一个 [LogFuncPipeline] 。
func NewLogFuncPipeline(fs ...LogFunc) LogFuncPipeline {
	return LogFuncPipeline(fs)
}

// Log implements [ApiLogger.Log].
func (p LogFuncPipeline) Log(state *ApiState) {
	logger := state.Logger
	if logger == nil || len(p) == 0 {
		return
	}

	for _, f := range p {
		f(state)
	}

	lv := state.LogLevel
	if state.LogLevel == 0 {
		lv = logx.LevelInfo
	}

	logger.Log(lv, "", state.LogMessage...)
}
