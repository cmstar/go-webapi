// webapi 包定义一组抽象过程与辅助类型，用于开发特定协议的 WebAPI 框架，如 SlimAPI 。
package webapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/cmstar/go-errx"
	"github.com/cmstar/go-logx"
)

/*
当前文件包含框架内的基础接口定义和执行流程。
*/

// ApiHandler 定义了 WebAPI 处理过程中的抽象环节。
// CreateHandlerFunc() 返回一个函数，基于 ApiHandler 实现完整的处理过程。
//
// 其中 ApiNameResolver 、 ApiUserHostResolver 、 ApiDecoder 、 ApiMethodCaller
type ApiHandler interface {
	ApiMethodRegister
	ApiNameResolver
	ApiUserHostResolver
	ApiDecoder
	ApiMethodCaller
	ApiResponseBuilder
	ApiResponseWriter
	ApiLogger

	// Name 获取当前 ApiHandler 的标识符。每个 ApiHandler 应具有唯一的名称。
	// 名称可以是任意值，包括空字符串。但应尽量给定容易识别的名称。
	Name() string

	// SupportedHttpMethods 返回当前 ApiHandler 支持的 HTTP 方法。如 GET 、 POST 、 PUT 、 DELETE 等。
	SupportedHttpMethods() []string
}

// ApiHandlerWrapper 用于组装各个接口，以实现 ApiHandler 。
// 各种 ApiHandler 的实现中，可使用此类型作为脚手架，组装各个内嵌接口。
type ApiHandlerWrapper struct {
	ApiMethodRegister
	ApiNameResolver
	ApiUserHostResolver
	ApiDecoder
	ApiMethodCaller
	ApiResponseBuilder
	ApiResponseWriter
	ApiLogger

	// HandlerName 是 ApiHandler.Name() 的返回值。
	HandlerName string

	// HttpMethods 是 ApiHandler.SupportedHttpMethods() 的返回值。
	HttpMethods []string
}

var _ ApiHandler = (*ApiHandlerWrapper)(nil)

// SupportedHttpMethods 实现 ApiHandler.SupportedHttpMethods() 。
func (w *ApiHandlerWrapper) SupportedHttpMethods() []string {
	return w.HttpMethods
}

// Name 实现 ApiHandler.Name() 。
func (w *ApiHandlerWrapper) Name() string {
	return w.HandlerName
}

// ApiMethod 表示一个通过 ApiMethodRegister 注册的方法。
type ApiMethod struct {
	// Name 是注册的 WebAPI 方法的名称。
	// 虽然在检索时使用大小写不敏感的方式，但这里通常记录注册时所使用的可区分大小写的名称。
	Name string

	// Value 记录目标方法。
	Value reflect.Value

	// Provider 指定方法提供者的名称，用于对方法加以分类，可为空。
	Provider string
}

// ApiMethodRegister 用于向 ApiHandler 中注册 WebAPI 方法。
// 此过程用于初始化 ApiHandler ，初始化过程应在接收第一个请求前完成，并以单线程方式进行。
// 注册方法时，应对方法的输入输出类型做合法性校验。
type ApiMethodRegister interface {
	// RegisterMethod 注册一个方法。
	// 注册时，对于方法名称应采用大小写不敏感的方式处理。若多次注册同一个名称，最后注册的将之前的覆盖。
	//
	// 允许方法具有0-2个输出参数。
	//   - 1个参数时，参数可以是任意 struct/map[string]*/基础类型 或者此三类作为元素的 slice ，也可以是 error 。
	//   - 2个参数时，第一个参数可以是  struct/map[string]*/基础类型 或者此三类作为元素的 slice ，第二个参数必须是 error 。
	//
	RegisterMethod(m ApiMethod)

	// RegisterMethods 将给定的 struct 上的所有公开方法注册为 WebAPI 。若给定的不是 struct ，则 panic 。
	// 通过此方法注册后，通过 GetMethod() 获取的 ApiMethod.Provider 为给定的 struct 的名称，对应 reflect.Type.Name() 的值。
	//
	// 对方法名称使用一组约定（下划线使用名称中的第一个下划线）：
	//   - 若方法名称格式为 Method__Name （使用两个下划线分割），则 Name 被注册为 WebAPI 名称；
	//   - 若方法名称格式为 Method__ （使用两个下划线结尾）或 Method____ （两个下划线之后也只有下划线），则此方法不会被注册为 WebAPI ；
	//   - 其余情况，均使用方法的原始名称作为 WebAPI 名称。
	// 这里 Method 和 Name 均为可变量， Method 用于指代代码内有意义的方法名称， Name 指代 WebAPI 名称。例如 GetName__13 注册一个名称为
	// “13”的 API 方法，其方法业务意义为 GetName 。
	//
	// 每个方法的注册逻辑与 RegisterMethod 一致。
	// 特别的，如果格式为 Method____abc ，两个下划线之后存在有效名称，则 WebAPI 名称为 __abc ，从两个下划线后的下一个字符（还是下划线）开始取。
	//
	RegisterMethods(providerStruct interface{})

	// GetMethod 返回具有指定名称的方法。若方法存在，返回 ApiMethod 和 true ；若未被注册，返回零值和 false 。
	// 对于方法名称应采用大小写不敏感的方式处理。
	GetMethod(name string) (method ApiMethod, ok bool)
}

// ApiNameResolver 用于从当前 HTTP 请求中，解析得到目标 API 方法的名称。
type ApiNameResolver interface {
	// FillMethod 从当前 HTTP 请求里获取 API 方法的名称，并填入 ApiState.Name ；如果未能解析到名称，则不需要填写。
	// 若请求非法，可填写 ApiState.Error ，跳过 ApiDecoder 和 ApiMethodCaller 的执行。
	FillMethod(state *ApiState)
}

// ApiUserHostResolver 用于获取发起 HTTP 请求的客户端 IP 地址。
// 一个请求可能经过多次代理转发，原始地址通常需要从特定 HTTP 头获取，比如 X-Forwarded-For 。
type ApiUserHostResolver interface {
	// FillUserHost 获取发起 HTTP 请求的客户端 IP 地址，并填入 ApiState.UserHost 。
	//
	// HTTP 服务经常通过反向代理访问，可能转好几层，需要通过如 X-Forwarded-For 头获取客户端 IP 。
	//
	FillUserHost(state *ApiState)
}

// ApiDecoder 用于构建调用方法的参数表。
type ApiDecoder interface {
	// Decode 从 HTTP 请求中，构建用于调用 ApiState.Method 的参数，并填入 ApiState.Args 。
	// 若参数转换失败，填写 ApiState.Error ，将跳过 ApiMethodCaller 的执行。
	Decode(state *ApiState)
}

// ApiMethodCaller 用于调用特定的方法。
type ApiMethodCaller interface {
	// 使用参数 ApiState.Args 调用 ApiState.Method 所对应的方法，将调用结果填入 ApiState.Data 和 ApiState.Error 。
	// 应仅在 ApiState.Error 为 nil 时调用此方法。方法通过 ApiMethodRegister 注册时已完成类型校验。
	Call(state *ApiState)
}

// ApiResponseBuilder 处理 ApiDecoder 和 ApiMethodCaller 执行过程中产生的错误。
type ApiResponseBuilder interface {
	// BuildResponse 根据 ApiState.Data 和 ApiState.Error ，填写 ApiState.Response 。
	BuildResponse(state *ApiState)
}

// ApiResponseWriter 处理 ApiMethodCaller 的处理结果，获得实际需要返回的数据，填入 Response* （以 Response 开头）字段。
type ApiResponseWriter interface {
	// 处理 ApiState.Response ，获得实际需要返回的数据，填入 ApiState.Response* （以 Response 开头）字段。
	// 此方法执行之后， ApiState 中以 Response 开头字段，如 ResponseBody 、 ResponseContentType ，
	// 均需要完成赋值。
	WriteResponse(state *ApiState)
}

// ApiLogger 在 ApiResponseWriter.WriteResponse 被调用后，生成日志。
type ApiLogger interface {
	// Log 根据 ApiState 的内容生成日志，日志由 ApiState.Logger 接收。
	// 若 ApiState.Logger 为 nil ，则不生成日志。
	Log(state *ApiState)
}

// CreateHandlerFunc 返回一个封装了给定的 ApiHandler 的 http.HandlerFunc 。
//
// logFinder 用于获取 Logger ，该 Logger 会赋值给 ApiState.Logger 。可为 nil 表示不记录日志。
// 对于每个请求，其日志名称基于响应该请求的方法，由两部分构成，格式为“{ApiHandler.Name()}.{ApiMethod.Provider}.{ApiMethod.Name}”。
// 如果未能检索到对应的方法，则日志名称为 ApiHandler.Name() 。
func CreateHandlerFunc(handler ApiHandler, logFinder logx.LogFinder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := NewState(w, r, handler)

		handler.FillUserHost(state)

		// 把比较可能 panic 的步骤抽出来，添加一个 defer 捕获错误并填到 state.Error 是上，使 panic 后仍
		// 可以预定义的报文返回结果。
		handleRequest(state, handler, logFinder)

		if !handleResponse(state, handler, logFinder) {
			// handleResponse 没成功，最大可能是方法返回值是不能序列化的。
			// 尝试清空返回值，再输出一次。 state.Error 则被保留下来，能够体现哪里出错。
			// 如果再 panic 就拯救不了了，交给外层框架处理。
			state.Data = nil
			handler.BuildResponse(state)
			handler.WriteResponse(state)
		}

		w.Header().Set(string(HttpHeaderContentType), string(state.ResponseContentType))
		_, err := io.Copy(w, state.ResponseBody)
		if err != nil {
			PanicApiError(state, err, "write response body")
		}

		handler.Log(state)
	}
}

func handleRequest(state *ApiState, handler ApiHandler, logFinder logx.LogFinder) {
	defer handlerPanic(state, handler, logFinder)

	handler.FillMethod(state)

	method, ok := handler.GetMethod(state.Name)
	if !ok {
		state.Error = CreateBadRequestError(state, errors.New("method not found"), "bad request")
		if logFinder != nil {
			state.Logger = logFinder.Find(handler.Name())
		}
		return
	}

	state.Method = method
	loggerName := handler.Name() + "." + method.Provider + "." + method.Name
	if logFinder != nil {
		state.Logger = logFinder.Find(loggerName)
	}

	handler.Decode(state)
	if state.Error == nil {
		handler.Call(state)
	}
}

func handleResponse(state *ApiState, handler ApiHandler, logFinder logx.LogFinder) bool {
	defer handlerPanic(state, handler, logFinder)
	handler.BuildResponse(state)
	handler.WriteResponse(state)
	return true
}

func handlerPanic(state *ApiState, handler ApiHandler, logFinder logx.LogFinder) {
	r := recover()
	if r == nil {
		return
	}

	// 尽量保留方法调用栈信息，如果没有，就放一个上去。
	switch v := r.(type) {
	case errx.StackfulError: // 含 BizError 。
		state.Error = v
	case error:
		state.Error = errx.Wrap(state.Name, v)
	case string:
		state.Error = errx.Wrap(state.Name, errors.New(v))
	default:
		// panic 的不是 error 和字符串也应该是个能转成字符串的东西。
		e := fmt.Errorf("%v", v)
		state.Error = errx.Wrap(state.Name, e)
	}

	if logFinder != nil {
		state.Logger = logFinder.Find(handler.Name())
	}
}
