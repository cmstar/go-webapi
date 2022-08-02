package webapi

const (
	// ContentTypeNone 未指定类型。
	ContentTypeNone = ""

	// ContentTypeJson 对应 Content-Type: application/json 的值。
	ContentTypeJson = "application/json"

	// ContentTypeBinary 对应 Content-Type: application/octet-stream 的值。
	ContentTypeBinary = "application/octet-stream"

	// ContentTypeJavascript 对应 Content-Type: text/javascript 的值。
	ContentTypeJavascript = "text/javascript"

	// ContentTypePlainText 对应 Content-Type: text/javascript 的值。
	ContentTypePlainText = "text/plain"

	// ContentTypeForm 对应 Content-Type: application/x-www-form-urlencoded 的值。
	ContentTypeForm = "application/x-www-form-urlencoded"

	// ContentTypeMultipartForm 对应 Content-Type: multipart/form-data 的值。
	ContentTypeMultipartForm = "multipart/form-data"
)

const (
	// HttpHeaderContentType 对应 HTTP 头中的 Content-Type 字段。
	HttpHeaderContentType = "Content-Type"
)

// 用于 WebAPI 预定义的状态码。1000以下基本抄 HTTP 状态码。
const (
	// 错误码。表示不合规的请求数据。
	ErrorCodeBadRequest = 400

	// 错误码。表示发生内部错误。
	ErrorCodeInternalError = 500
)

// ApiResponse 用于表示返回的数据。
type ApiResponse[T any] struct {
	// 状态码， 0 表示一个成功的请求，其他值表示有错误。
	Code int

	// Message 在 Code 不为 0 时，记录用于描述错误的消息。
	Message string

	// Data 记录返回的数据本体。
	Data T
}

// SuccessResponse 返回一个表示成功的 ApiResponse 。
func SuccessResponse[T any](data T) *ApiResponse[T] {
	return &ApiResponse[T]{Data: data}
}

// BadRequestResponse 返回一个表示不合规的请求的 ApiResponse 。
func BadRequestResponse() *ApiResponse[any] {
	return &ApiResponse[any]{
		Code:    ErrorCodeBadRequest,
		Message: "bad request",
	}
}

// InternalErrorResponse 返回一个表示不合规的请求的 ApiResponse 。
func InternalErrorResponse() *ApiResponse[any] {
	return &ApiResponse[any]{
		Code:    ErrorCodeInternalError,
		Message: "internal error",
	}
}
