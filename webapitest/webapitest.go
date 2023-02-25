// webapitest 包提供用于测试 webapi 包的辅助方法。
package webapitest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/cmstar/go-webapi"
)

// NoOpHandler 是一个空的 webapi.ApiHandler ，用于测试用例中不需要访问其方法只需要一个实例占位的场景。
var NoOpHandler webapi.ApiHandler = &webapi.ApiHandlerWrapper{}

// NewStateSetup 用于设置用于测试 HTTP 请求。
type NewStateSetup struct {
	HttpMethod  string            // HTTP 请求的方法， GET/POST/PUT/DELETE 。若未给定值，默认为 GET 。
	ContentType string            // 指定 HTTP Content-Type 头，若未给定值，则不会添加此字段。
	Headers     map[string]string // 其他 HTTP 头。
	BodyString  string            // 指定请求的 body ，优先级高于 BodyReader 。给定值时 BodyReader 被忽略。
	BodyReader  io.Reader         // 指定请求的 body ，仅在 BodyString 为空时生效。
	RouteParams map[string]string // 指定路由参数。若为 nil 或为空集则不会初始化路由参数。
}

// NewStateForTest 基于 httptest 包创建用于测试 HTTP 请求的相关实例。
func NewStateForTest(apiHandler webapi.ApiHandler, uri string, setup NewStateSetup) (*webapi.ApiState, *httptest.ResponseRecorder) {
	httpMethod := setup.HttpMethod
	if httpMethod == "" {
		httpMethod = http.MethodGet
	}

	req := httptest.NewRequest(httpMethod, uri, nil)

	if setup.ContentType != "" {
		req.Header.Add(webapi.HttpHeaderContentType, setup.ContentType)
	}

	if setup.Headers != nil {
		for k, v := range setup.Headers {
			req.Header.Add(k, v)
		}
	}

	if setup.BodyString != "" {
		req.Body = io.NopCloser(strings.NewReader(setup.BodyString))
	} else if setup.BodyReader != nil {
		readCloser, ok := setup.BodyReader.(io.ReadCloser)
		if ok {
			req.Body = readCloser
		} else {
			req.Body = io.NopCloser(setup.BodyReader)
		}
	}

	req = webapi.SetRouteParams(req, setup.RouteParams)
	rec := httptest.NewRecorder()
	state := webapi.NewState(rec, req, apiHandler)
	return state, rec
}
