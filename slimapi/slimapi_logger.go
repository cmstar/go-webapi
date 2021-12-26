package slimapi

import (
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
			for _, fs := range req.MultipartForm.File {
				for _, f := range fs {
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
		}

		if state.Error != nil {
			keyValues = append(keyValues, "ErrorType", errTypeName)
			keyValues = append(keyValues, "Error", errDescription)
		}

		// Leave the output parameter @message an empty string.
		return
	})
}
