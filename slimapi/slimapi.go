package slimapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/cmstar/go-conv"
	"github.com/cmstar/go-webapi"
)

const (
	// URL 上的元参数名称。
	meta_Param_Method   = "~method"
	meta_Param_Format   = "~format"
	meta_Param_Callback = "~callback"

	// URL 上表示请求格式的串。用于兼容不方便指定 Content-Type 的情况。
	meta_RequestFormat_Json   = "json"
	meta_RequestFormat_Post   = "post"
	meta_RequestFormat_Get    = "get"
	meta_ResponseFormat_Plain = "plain"

	// 解析请求的 body 部分时最大可用的内存，读取 multipart-form-data 型数据时，超过此字节数将使用临时文件存储。
	// 另外，也是单独使用代码读取 body 时，允许的最大的字节数。
	maxMemorySizeParseRequestBody = 10 * 1024 * 1024
)

// 用作在 ApiState 上存储自定义数据的 key 。
type customDataKey int

const (
	// 自定义字段。记录当前请求使用的格式（对应 meta_RequestFormat_* 常量）。格式优先从 URL 上解析，其次是 Content-Type 头。
	customData_RequestFormat customDataKey = iota

	// 对于 JSONP 请求，记录回调方法的名称。
	customData_ResponseCallback

	// 自定义字段。记录当前请求 body 部分， ApiDecoder.Decode() 在执行后，将读取到的 body 存储在此字段上。
	customData_BufferedBody
)

// Conv 是用于 SlimAPI 的 [conv.Conv] 实例，它支持：
//   - 使用大小写不敏感（case-insensitive）的方式处理字段。
//   - 支持 SlimAPI 规定的时间格式 yyyyMMdd HH:mm:ss 。
//   - 支持字符串到数组的转换，使用 ~ 分割，如将 "1~2~3" 转为 [1, 2, 3] 。
var Conv = conv.Conv{
	Conf: conv.Config{
		FieldMatcherCreator: &conv.SimpleMatcherCreator{
			Conf: conv.SimpleMatcherConfig{
				CaseInsensitive: true,
			},
		},
		StringToTime:   ParseTime,
		StringSplitter: func(v string) []string { return strings.Split(v, "~") },
	},
}

// NewSlimApiHandler 创建一个实现 SlimAPI 协议的 webapi.ApiHandlerWrapper 。
// 可通过替换其成员实现接口的定制。
func NewSlimApiHandler(name string) *webapi.ApiHandlerWrapper {
	return &webapi.ApiHandlerWrapper{
		HandlerName:         name,
		HttpMethods:         SupportedHttpMethods(),
		ApiNameResolver:     NewSlimApiNameResolver(),
		ApiDecoder:          NewSlimApiDecoder(),
		ApiMethodCaller:     webapi.NewBasicApiMethodCaller(),
		ApiResponseBuilder:  webapi.NewBasicApiResponseBuilder(),
		ApiMethodRegister:   webapi.NewBasicApiMethodRegister(),
		ApiUserHostResolver: webapi.NewBasicApiUserHostResolver(),
		ApiResponseWriter:   NewSlimApiResponseWriter(),
		ApiLogger:           NewSlimApiLogger(),
	}
}

// SupportedHttpMethods 返回 SlimAPI 支持的 HTTP 请求方法。
// 当前支持 GET 和 POST 。
func SupportedHttpMethods() []string {
	return []string{http.MethodGet, http.MethodPost}
}

// 将解析到的 请求格式 存储到 ApiState 中。
func setRequestFormat(state *webapi.ApiState, v string) {
	state.SetCustomData(customData_RequestFormat, v)
}

// 读取 setRequestFormat 设置的值。
func getRequestFormat(state *webapi.ApiState) string {
	return getCustomString(state, customData_RequestFormat)
}

// 将解析到的 回调名称 存储到 ApiState 中。
func setCallback(state *webapi.ApiState, callback string) {
	state.SetCustomData(customData_ResponseCallback, callback)
}

// 读取 setCallback 设置的值。
func getCallback(state *webapi.ApiState) string {
	return getCustomString(state, customData_ResponseCallback)
}

// 将请求的 body 部分相关的信息存储到 ApiState 中。目前这部分描述用在日志上部分。
// body 可以是：
//   - string 直接输出在日志，无需转换。
//   - fmt.Stringer 通过 String() 方法被转换为 string 。
//   - 其他类型，通过json.Marshal() 序列化为字符串。
func setRequestBodyDescription(state *webapi.ApiState, body any) {
	state.SetCustomData(customData_BufferedBody, body)
}

// 读取 setRequestBodyDescription 设置的值。
func getRequestBodyDescription(state *webapi.ApiState) string {
	v, ok := state.GetCustomData(customData_BufferedBody)
	if !ok {
		return ""
	}

	if s, ok := v.(string); ok {
		return s
	}

	if s, ok := v.(fmt.Stringer); ok {
		return s.String()
	}

	s, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	return string(s)
}

func getCustomString(state *webapi.ApiState, key any) string {
	v, ok := state.GetCustomData(key)
	if ok {
		return v.(string)
	}
	return ""
}
