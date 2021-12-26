package slimapi

import (
	"mime/multipart"
	"sort"
	"strconv"

	"github.com/cmstar/go-webapi"
)

// slimApiLogger 实现 SlimAPI 的 webapi.ApiLogger 。
type slimApiLogger struct {
}

// NewSlimApiApiResponseWriter 返回用于 SlimAPI 协议的 webapi.ApiLogger 实现。
// 该实现是无状态且线程安全的。
func NewSlimApiApiLogger() webapi.ApiLogger {
	return &slimApiLogger{}
}

// Log implements ApiLogger.Log .
func (sl *slimApiLogger) Log(state *webapi.ApiState) {
	l := state.Logger
	if l == nil {
		return
	}

	logLevel, errTypeName, errDescription := webapi.DescribeError(state.Error)

	l.LogFn(logLevel, func() (message string, keyValues []interface{}) {
		keyValues = append(keyValues,
			"Ip", state.UserHost,
			"Url", state.Ctx.Request().RequestURI,
		)

		// getBufferedBody 可以获取到所有类型的文本参数，不含文件上传。
		body := getBufferedBody(state)
		if len(body) > 0 {
			keyValues = append(keyValues,
				"Length", len(body),
				"Body", body,
			)
		}

		// 单独处理上传的文件，只输出文件名和体积。
		// 文件可能有同名的， echo 框架把同名的并在一个 map 的 value 上了。
		fileNum := 0
		req := state.Ctx.Request()
		if req.MultipartForm != nil {
			for _, f := range sl.sortedFileHeaders(req.MultipartForm) {
				fileNumStr := strconv.Itoa(fileNum)
				keyValues = append(keyValues,
					"File"+fileNumStr, f.Filename,
					"Length"+fileNumStr, f.Size,
				)

				if f.Header != nil {
					if header, ok := f.Header[webapi.HttpHeaderContentType]; ok {
						keyValues = append(keyValues, "ContentType"+fileNumStr, header[0])
					}
				}

				fileNum++
			}
		}

		if state.Error != nil {
			keyValues = append(keyValues, "ErrorType", errTypeName)
			keyValues = append(keyValues, "Error", errDescription)
		}

		// Leave the output parameter @message an empty string.
		return
	})
}

// sortedFileHeaders 排序 MultipartForm ，解决 map 输出顺序不稳定的问题，以便获得稳定的日志。
func (*slimApiLogger) sortedFileHeaders(form *multipart.Form) []*multipart.FileHeader {
	keys := make([]string, 0, len(form.File))
	for k := range form.File {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var mergedHeaders []*multipart.FileHeader
	for i := 0; i < len(keys); i++ {
		headers := form.File[keys[i]]
		mergedHeaders = append(mergedHeaders, headers...)
	}
	return mergedHeaders
}
