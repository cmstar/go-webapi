package slimauth

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/cmstar/go-logx"
	"github.com/cmstar/go-webapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	_requestTypeGet  = 0
	_requestTypeForm = 1
	_requestTypeJson = 2

	// 没有时间戳校验时，固定用此时间戳测试，以便获得稳定可断言的 hash 。
	_timestamp = 1661934251

	// 测试时表示合法的 access-key 。
	_key = "key"

	// 当 access-key 为 key 时，返回这个密钥。
	_secret = "secret"
)

// baseUrl 可留空。
func newRequest(baseUrl, pathAndQuery string, typ int, body string) *http.Request {
	if baseUrl == "" {
		baseUrl = "http://temp.org"
	}

	url, err := url.Parse(baseUrl + pathAndQuery)
	if err != nil {
		panic(err)
	}

	r := &http.Request{
		URL:    url,
		Header: make(http.Header),
	}

	if body != "" {
		r.Body = io.NopCloser(strings.NewReader(body))
	}

	switch typ {
	case _requestTypeGet:
		r.Method = "GET"

	case _requestTypeForm:
		r.Method = "POST"
		r.Header.Set(webapi.HttpHeaderContentType, webapi.ContentTypeForm)

	case _requestTypeJson:
		r.Method = "POST"
		r.Header.Set(webapi.HttpHeaderContentType, webapi.ContentTypeJson)
	}

	return r
}

type methodProvider struct{}

func (methodProvider) Plus(req struct{ X, Y int }) int {
	return req.X + req.Y
}

func (methodProvider) GetKey(auth *Authorization) string {
	return auth.Key
}

// op.SecretFinder 不需要给定，会自动赋值。
func newTestServer(op SlimAuthApiHandlerOption) *httptest.Server {
	op.SecretFinder = func(accessKey string) string {
		switch accessKey {
		case _key:
			return _secret

		default:
			return ""
		}
	}

	handler := NewSlimAuthApiHandler(op)
	handler.RegisterMethods(methodProvider{})

	logger := logx.NopLogger
	handlerFunc := webapi.CreateHandlerFunc(handler, logx.NewSingleLoggerLogFinder(logger))
	ts := httptest.NewServer(http.HandlerFunc(handlerFunc))
	return ts
}

func testRequest(t *testing.T, r *http.Request, want string) {
	client := new(http.Client)
	res, _ := client.Do(r)
	body, _ := io.ReadAll(res.Body)
	assert.Equal(t, want, string(body))
}

// 测试不包含时间戳校验的其他错误。
func TestSlimAuthApiHandler_errors(t *testing.T) {
	s := newTestServer(SlimAuthApiHandlerOption{
		TimeChecker: NoTimeChecker,
	})

	t.Run("NoMethod", func(t *testing.T) {
		r, _ := http.NewRequest("GET", s.URL, nil)
		testRequest(t, r, `{"Code":400,"Message":"invalid Authorization","Data":null}`)
	})

	t.Run("InvalidAuth", func(t *testing.T) {
		r, _ := http.NewRequest("GET", s.URL+"?Plus", nil)
		testRequest(t, r, `{"Code":400,"Message":"invalid Authorization","Data":null}`)
	})

	t.Run("InvalidAuthVersion", func(t *testing.T) {
		r, _ := http.NewRequest("GET", s.URL+"?Plus", nil)
		r.Header.Set(HttpHeaderAuthorization, "SLIM-AUTH Key=key, Sign=sign, Timestamp=1, Version=-100")

		testRequest(t, r, `{"Code":400,"Message":"unsupported signature version","Data":null}`)
	})

	t.Run("UnknownKey", func(t *testing.T) {
		r, _ := http.NewRequest("GET", s.URL+"?Plus", nil)
		r.Header.Set(HttpHeaderAuthorization, "SLIM-AUTH Key=unknown, Sign=sign, Timestamp=1")

		testRequest(t, r, `{"Code":400,"Message":"unknown key","Data":null}`)
	})

	t.Run("NoContentType", func(t *testing.T) {
		r, _ := http.NewRequest("POST", s.URL+"?Plus", nil)
		r.Header.Set(HttpHeaderAuthorization, "SLIM-AUTH Key=key, Sign=sign, Timestamp=1")

		testRequest(t, r, `{"Code":400,"Message":"missing Content-Type","Data":null}`)
	})

	t.Run("UnsupportedContentType", func(t *testing.T) {
		r, _ := http.NewRequest("POST", s.URL+"?Plus", nil)
		r.Header.Set(HttpHeaderAuthorization, "SLIM-AUTH Key=key, Sign=sign, Timestamp=1")
		r.Header.Set(webapi.HttpHeaderContentType, "Invalid-Content-Type")

		testRequest(t, r, `{"Code":400,"Message":"unsupported Content-Type","Data":null}`)
	})

	t.Run("InvalidForm", func(t *testing.T) {
		r, _ := http.NewRequest("POST", s.URL+"?Plus", strings.NewReader(";=;"))
		r.Header.Set(HttpHeaderAuthorization, "SLIM-AUTH Key=key, Sign=sign, Timestamp=1")
		r.Header.Set(webapi.HttpHeaderContentType, webapi.ContentTypeForm)

		testRequest(t, r, `{"Code":400,"Message":"invalid request body","Data":null}`)
	})

	t.Run("BadSign", func(t *testing.T) {
		auth := BuildAuthorizationHeader(Authorization{
			Key:       _key,
			Sign:      "bad",
			Timestamp: _timestamp,
		})

		r, _ := http.NewRequest("GET", s.URL+"?Plus&x=1", nil)
		r.Header.Set(HttpHeaderAuthorization, auth)

		testRequest(t, r, `{"Code":400,"Message":"signature error","Data":null}`)
	})
}

// 测试时间戳校验。
func TestSlimAuthApiHandler_timeChecker(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		s := newTestServer(SlimAuthApiHandlerOption{}) // 自动使用 DefaultTimeChecker 。

		r, _ := http.NewRequest("GET", s.URL+"?Plus&x=1", nil)
		signResult := AppendSign(r, _key, _secret, "", time.Now().Unix())
		require.Equal(t, SignResultType_OK, signResult.Type)

		testRequest(t, r, `{"Code":0,"Message":"","Data":1}`)
	})

	t.Run("TimestampError", func(t *testing.T) {
		s := newTestServer(SlimAuthApiHandlerOption{
			TimeChecker: DefaultTimeChecker,
		})

		timestamp := time.Now().Unix() + 1000

		r, _ := http.NewRequest("GET", s.URL+"?Plus&x=1", nil)
		signResult := AppendSign(r, _key, _secret, "", timestamp)
		require.Equal(t, SignResultType_OK, signResult.Type)

		testRequest(t, r, `{"Code":400,"Message":"timestamp error","Data":null}`)
	})
}

func TestSlimAuthApiHandler_ok(t *testing.T) {
	s := newTestServer(SlimAuthApiHandlerOption{
		TimeChecker: NoTimeChecker,
	})

	t.Run("PlusViaForm", func(t *testing.T) {
		/*
			data to sign:
				1661934251
				POST
				/
				aPlus1122
				bc
				END
		*/

		auth := BuildAuthorizationHeader(Authorization{
			Key:       _key,
			Sign:      "a9bb620ee2689035dbac5970deb6ba3789be2c0f61f0feb72b705410a9ac06f2",
			Timestamp: _timestamp,
		})

		r, _ := http.NewRequest("POST", s.URL+"?Plus&x=11&AA=a&y=22", strings.NewReader("c=c&b=b"))
		r.Header.Set(webapi.HttpHeaderContentType, webapi.ContentTypeForm)
		r.Header.Set(HttpHeaderAuthorization, auth)

		testRequest(t, r, `{"Code":0,"Message":"","Data":33}`)
	})

	t.Run("GetKey", func(t *testing.T) {
		/*
			data to sign:
				1661934251
				POST
				/
				GetKey
				{}
				END
		*/

		auth := BuildAuthorizationHeader(Authorization{
			Key:       _key,
			Sign:      "92f92b7840aefdc6bbb35f338ecdd2a80a2543c46f713d12c7b54825d7b69ab6",
			Timestamp: _timestamp,
		})

		r, _ := http.NewRequest("POST", s.URL+"?GetKey", strings.NewReader(`{}`))
		r.Header.Set(webapi.HttpHeaderContentType, webapi.ContentTypeJson)
		r.Header.Set(HttpHeaderAuthorization, auth)

		testRequest(t, r, `{"Code":0,"Message":"","Data":"key"}`)
	})
}

func TestSlimAuthApiHandler_customScheme(t *testing.T) {
	const scheme = "CUSTOM-SCHEME"

	s := newTestServer(SlimAuthApiHandlerOption{
		AuthScheme:  scheme,
		TimeChecker: NoTimeChecker,
	})

	// 复用测试用例，除 scheme 不一样外，其他都一样。
	t.Run("PlusViaForm", func(t *testing.T) {
		auth := BuildAuthorizationHeader(Authorization{
			AuthScheme: scheme,
			Key:        _key,
			Sign:       "a9bb620ee2689035dbac5970deb6ba3789be2c0f61f0feb72b705410a9ac06f2",
			Timestamp:  _timestamp,
		})

		r, _ := http.NewRequest("POST", s.URL+"?Plus&x=11&AA=a&y=22", strings.NewReader("c=c&b=b"))
		r.Header.Set(webapi.HttpHeaderContentType, webapi.ContentTypeForm)
		r.Header.Set(HttpHeaderAuthorization, auth)

		testRequest(t, r, `{"Code":0,"Message":"","Data":33}`)
	})
}
