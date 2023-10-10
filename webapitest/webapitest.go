// webapitest 包提供用于测试 webapi 包的辅助方法。
package webapitest

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
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

// CreateMultipartFileHeader 根据给定的内容创建一个 multipart.FileHeader 实例。
func CreateMultipartFileHeader(fieldName, fileName string, body []byte, header map[string]string) *multipart.FileHeader {
	// 没有提供直接创建 FileHeader 的方法。这里通过 Writer 写出一份数据，再用 Reader 读出来。
	buf := new(bytes.Buffer)
	w := multipart.NewWriter(buf)

	mimeHeader := make(textproto.MIMEHeader)
	mimeHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fileName))
	for k, v := range header {
		mimeHeader[k] = []string{v}
	}
	file, err := w.CreatePart(mimeHeader)
	if err != nil {
		panic(err)
	}
	file.Write(body)

	w.Close()

	reader := multipart.NewReader(buf, w.Boundary())
	form, err := reader.ReadForm(10 << 20)
	if err != nil {
		panic(err)
	}

	for _, v := range form.File {
		return v[0]
	}

	panic("something wrong")
}
