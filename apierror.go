package webapi

import (
	"fmt"
	"reflect"

	"github.com/cmstar/go-errx"
	"github.com/cmstar/go-logx"
)

/*
当前文件提供 webapi 的相关错误类型及处理错误的方法。
*/

// withinStateError 描述一个请求的处理过程中的错误，用作其他错误的内嵌结构。
type withinStateError struct {
	errx.ErrorCause

	State   *ApiState // State 记录当前的请求状态。
	Message string    // Message 记录错误的描述信息。
}

var _ error = (*withinStateError)(nil)

// Error 实现 error 接口。
func (e withinStateError) Error() string {
	return e.Message
}

// ApiError 用于表示 ApiHandler 处理过程中的内部错误，这些错误通常表示代码存在问题（如编码 bug）。
// 这些问题不能在程序生命周期中自动解决，通常使用 panic 中断程序。
type ApiError struct {
	withinStateError
}

// CreateApiError 创建一个 ApiError 。
// message 和 args 指定描述信息，使用 fmt.Sprintf() 格式化。 cause 是引起此错误的错误，可以为 nil 。
// message 会体现在  ApiError.Error() ，格式为：
//
//	message:: cause.Error()
func CreateApiError(state *ApiState, cause error, message string, args ...any) ApiError {
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}

	if cause != nil {
		if message != "" {
			message += ":: "
		}
		message += cause.Error()
	}

	e := ApiError{
		withinStateError{
			ErrorCause: errx.ErrorCause{Err: cause},
			State:      state,
			Message:    message,
		},
	}
	return e
}

// PanicApiError 使用 CreateApiError 创建 ApiError ，并直接直接 panic 。
// 当 ApiHandler 遇见不应该发生（如编码 bug）的异常情况时，可使用此方法中断处理过程。
func PanicApiError(state *ApiState, cause error, message string, args ...any) {
	e := CreateApiError(state, cause, message, args...)
	panic(e)
}

// BadRequestError 记录一个不正确的请求，例如请求的参数不符合要求，请求的 API 方法不存在等。
// 这些错误是外部请求而不是编码导致（假设没 bug ）的， WebAPI 流程应能够正确处理这些错误并返回对应结果。
// 可以为 BadRequestError 指定一个描述信息，此信息可能作为 WebAPI 的返回值，被请求者看到。
type BadRequestError struct {
	withinStateError
}

// CreateBadRequestError 创建一个 BadRequestError 。
// message 和 args 指定其消息，使用 fmt.Sprintf() 格式化。
// 描述信息可能作为 WebAPI 的返回值，被请求者看到，故可能不应当过多暴露程序细节。更具体的错误可以放在 cause 上。
func CreateBadRequestError(state *ApiState, cause error, message string, args ...any) BadRequestError {
	message = fmt.Sprintf(message, args...)
	e := BadRequestError{
		withinStateError{
			ErrorCause: errx.ErrorCause{Err: cause},
			State:      state,
			Message:    message,
		},
	}
	return e
}

// DescribeError 根据给定的错误，返回错误的日志级别、名称和错误描述。 如果 err 为 nil ，返回 logx.LevelInfo 和空字符串。
// 此方法可用于搭配 ApiLogger.Log() 输出带有错误描述的日志。
//
// 描述信息使用 common.Errors.Describe() 获取。
func DescribeError(err error) (logLevel logx.Level, errTypeName, errDescription string) {
	if err == nil {
		return logx.LevelInfo, "", ""
	}

	errTypeName = getErrTypeName(err)
	errDescription = errx.Describe(err)

	logLevel = logx.LevelError
	switch err.(type) {
	case errx.BizError:
		logLevel = logx.LevelWarn
	case BadRequestError:
		logLevel = logx.LevelError
	case ApiError:
		// 属于代码不能正常执行的严重问题。
		logLevel = logx.LevelFatal
	}

	return
}

func getErrTypeName(err error) string {
	// 取 error 内在的实际类型的名称。
	typ := reflect.TypeOf(err)
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	name := typ.Name()

	// 如果是个公开类型（首字母大写），直接用其名称。
	if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
		return name
	}

	// 非公开的错误，如果是几个预定义且常见的，返回其接口名称。
	if _, ok := err.(errx.BizError); ok {
		return "BizError"
	}
	if _, ok := err.(errx.StackfulError); ok {
		return "StackfulError"
	}
	return name
}
