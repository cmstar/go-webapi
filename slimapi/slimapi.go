package slimapi

import (
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

	// 自定义字段。记录当前请求使用的格式（对应 meta_RequestFormat_* 常量）。格式优先从 URL 上解析，其次是 Content-Type 头。
	customData_RequestFormat = "RequestFormat"

	// 对于 JSONP 请求，记录回调方法的名称。
	customData_ResponseCallback = "ResponseCallback"

	// 自定义字段。记录当前请求 body 部分， ApiDecoder.Decode() 在执行后，将读取到的 body 存储在此字段上。
	customData_BufferedBody = "RequestBody"
)

// slimApiConv 用于 SlimAPI 中的 map 到 struct 的映射。
var slimApiConv = conv.Conv{
	Conf: conv.Config{
		// 使用大小写不敏感（case-insensitive）的方式处理字段。
		FieldMatcherCreator: &conv.SimpleMatcherCreator{
			Conf: conv.SimpleMatcherConfig{
				CaseInsensitive: true,
			},
		},

		// 支持 SlimAPI 规定的时间格式 yyyyMMdd HH:mm:ss 。
		StringToTime: parseTime,

		// 支持字符串到数组的转换，使用 ~ 分割，如将 "1~2~3" 转为 [1, 2, 3] 。
		StringSplitter: func(v string) []string { return strings.Split(v, "~") },
	},
}

// NewSlimApiHandler 创建一个实现 SlimAPI 协议的 webapi.ApiHandlerWrapper 。
// 可通过替换其成员实现接口的定制。
func NewSlimApiHandler(name string) *webapi.ApiHandlerWrapper {
	return &webapi.ApiHandlerWrapper{
		HandlerName:         name,
		HttpMethods:         []string{"GET", "POST"},
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

func setRequestFormat(state *webapi.ApiState, v string) {
	state.SetCustomData(customData_RequestFormat, v)
}

func getRequestFormat(state *webapi.ApiState) string {
	return getCustomString(state, customData_RequestFormat)
}

func setCallback(state *webapi.ApiState, callback string) {
	state.SetCustomData(customData_ResponseCallback, callback)
}

func getCallback(state *webapi.ApiState) string {
	return getCustomString(state, customData_ResponseCallback)
}

func setBufferedBody(state *webapi.ApiState, body string) {
	state.SetCustomData(customData_BufferedBody, body)
}

func getBufferedBody(state *webapi.ApiState) string {
	return getCustomString(state, customData_BufferedBody)
}

func getCustomString(state *webapi.ApiState, key string) string {
	v, ok := state.GetCustomData(key)
	if ok {
		return v.(string)
	}
	return ""
}
