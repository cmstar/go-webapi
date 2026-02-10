package webapi

import (
	"iter"
	"net/http"
	"reflect"

	"github.com/cmstar/go-logx"
)

// ApiState 用于记录一个请求的处理流程中的数据。每个请求使用一个新的 ApiState 。
// 处理过程采用管道模式，每个步骤从 ApiState 获取所需数据，并将处理结果写回 ApiState 。
// 当处理过程结束后，以 Response 开头的字段应被填充。
type ApiState struct {
	// RawRequest 是原始的 HTTP 请求。对应 http.Handler 的参数。
	RawRequest *http.Request

	// RawResponse 用于写入 HTTP 回执。对应 http.Handler 的参数。
	RawResponse http.ResponseWriter

	// Query 是按 ASP.net 的方式解析 URL 上的参数。
	// 由于通信协议是按 .net 版的方式制定的，获取 query-string 应通过此字段进行。
	Query QueryString

	// Handler 当前的 ApiHandler 。
	Handler ApiHandler

	// Logger 用于接收当前请求的处理流程中需记录的日志。可以为 nil ，表示不记录日志。
	// 应在 Method 被调用前，即 ApiMethodCaller.Call() 之前初始化。
	Logger logx.Logger

	// Name 记录调用 WebAPI 给定的方法名称，它应被唯一的映射到一个 Method 。
	// ApiNameResolver 接口定义了初始化此字段的方法。
	Name string

	// Method 记录要调用的方法，和 Name 一一映射，可从通过 ApiMethodRegister.GetMethod(ApiState.Name) 得到。
	// 方法由 ApiMethodCaller 调用，参数从 Args 获取。
	Method ApiMethod

	// MethodArgs 存放用于调用 Method 的参数。
	// ApiDecoder 接口定义了初始化此字段的方法。
	Args []reflect.Value

	// UserHost 记录发起 HTTP 请求的客户端 IP 地址。
	// ApiUserHostResolver 接口定义了初始化此字段的方法。
	UserHost string

	// Data 记录 ApiMethodCaller.Call() 方法所调用的具体 WebAPI 方法返回的非 error 值。
	// 若方法没有返回值，此字段为 nil 。
	Data any

	// 用于定制输出日志时的日志级别，用于 [ApiLogger] 的日志输出。
	// 若 [ApiLogger.Log] 被调用前没有被赋值（为 0），则 Log 方法被调用后，应将其最终决定使用的级别回填到此字段。
	LogLevel logx.Level

	// LogMessage 用于记录各个处理流程中的日志信息，用于在 [ApiLogger] 中的输出。
	// 最终日志的输出由 [ApiLogger] 决定，这只是一个缓冲（ buffer ）。
	// key-value 对，与 [logx.Logger.Log] 的 keyValues 参数定义一致。
	LogMessage []any

	// Error [ApiHandler] 中各管道发生从错误。包括 panic 后被捕获的错误。没有错误时为 nil 。
	Error error

	// ResponseBody 提供实际返回的 HTTP body 的数据。若为 nil ，则 HTTP body 为空。
	//
	// 这是一个迭代器，若 [http.ResponseWriter] 实现了 [http.Flusher] ，则每轮迭代返回的结果均会立刻触发 flush 。
	// flush 完成后，才继续下一轮迭代。
	//
	// 使用此机制，可实现 HTTP response 的流式输出。
	//
	// 若某轮迭代返回的 slice 长度为0，则该结果不会触发输出，迭代应继续执行。若所有轮次均返回空 slice ，则 HTTP body 为空。
	ResponseBody iter.Seq[[]byte]

	// ResponseContentType 对应为返回的 HTTP 的 Content-Type 头的值。
	ResponseContentType string

	// customData 用于记录没有预定义的数据，即不在其他字段中体现的数据，由各处理过程自行决定。
	customData []struct{ k, v any }
}

// NewState 创建一个新的 ApiState ，每个请求应使用一个新的 ApiState 。
func NewState(r http.ResponseWriter, w *http.Request, handler ApiHandler) *ApiState {
	s := &ApiState{
		Handler:     handler,
		RawRequest:  w,
		RawResponse: r,
	}
	s.Query = ParseQueryString(w.URL.RawQuery)
	return s
}

// MustHaveName checks the Name field, panics if the field is not initialized.
func (s *ApiState) MustHaveName() {
	if s.Name == "" {
		PanicApiError(s, nil, "ApiState.Name not resoled")
	}
}

// MustHaveMethod checks the Method field, panics if the field is not initialized or is not a Func.
func (s *ApiState) MustHaveMethod() {
	if !s.Method.Value.IsValid() {
		PanicApiError(s, nil, "ApiState.Method not initialized")
	}

	if s.Method.Value.Type().Kind() != reflect.Func {
		PanicApiError(s, nil, "the value of ApiState.Method must be Func")
	}
}

// SetCustomData 在当前 [*ApiState] 中存储一个自定义的值。
// 原理和 [context.WithValue] 类似， key 必须是可比较的。
func (s *ApiState) SetCustomData(key, value any) {
	s.customData = append(s.customData, struct{ k, v any }{key, value})
}

// GetCustomData 读取 [SetCustomData] 方法存放的值。返回一个 bool 值表示 key 是否存在。
func (s *ApiState) GetCustomData(key any) (any, bool) {
	data := s.customData
	for i := 0; i < len(data); i++ {
		if data[i].k == key {
			return data[i].v, true
		}
	}
	return nil, false
}
