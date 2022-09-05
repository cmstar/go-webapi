package slimauth

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/cmstar/go-logx"
	"github.com/cmstar/go-webapi"
	"github.com/stretchr/testify/assert"
)

const (
	_requestTypeGet  = 0
	_requestTypeForm = 1
	_requestTypeJson = 2

	// 固定用此时间戳测试，一遍获得稳定可断言的 hash 。
	_timestamp = 1661934251

	// 测试统一用这个密钥。
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

func newFinder() SecretFinder {
	return SecretFinderFunc(func(accessKey string) string {
		switch accessKey {
		case "key":
			return _secret

		default:
			return ""
		}
	})
}

type methodProvider struct{}

func (methodProvider) Plus(req struct{ X, Y int }) int {
	return req.X + req.Y
}

func (methodProvider) GetKey(auth *Authorization) string {
	return auth.Key
}

func newTestServer(secretFinder SecretFinder) *httptest.Server {
	handler := NewSlimAuthApiHandler("", secretFinder)
	handler.RegisterMethods(methodProvider{})

	logger := logx.NewStdLogger(nil)
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

func TestSlimAuthApiHandler_errors(t *testing.T) {
	s := newTestServer(newFinder())

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

		testRequest(t, r, `{"Code":400,"Message":"invalid form data","Data":null}`)
	})

	t.Run("BadSign", func(t *testing.T) {
		auth := BuildAuthorizationHeader(Authorization{
			Key:       "key",
			Sign:      "bad",
			Timestamp: _timestamp,
		})

		r, _ := http.NewRequest("GET", s.URL+"?Plus&x=1", nil)
		r.Header.Set(HttpHeaderAuthorization, auth)

		testRequest(t, r, `{"Code":400,"Message":"signature error","Data":null}`)
	})
}
func TestSlimAuthApiHandler_ok(t *testing.T) {
	s := newTestServer(newFinder())

	t.Run("PlusViaForm", func(t *testing.T) {
		auth := BuildAuthorizationHeader(Authorization{
			Key:       "key",
			Sign:      "66d4960c8b453050db7477c5c81afc366a95a98bcbffaad8d8732aacc812ed2b",
			Timestamp: _timestamp,
		})

		r, _ := http.NewRequest("POST", s.URL+"?Plus&x=11&aa=a&y=22", strings.NewReader("c=c&b=b"))
		r.Header.Set(webapi.HttpHeaderContentType, webapi.ContentTypeForm)
		r.Header.Set(HttpHeaderAuthorization, auth)

		testRequest(t, r, `{"Code":0,"Message":"","Data":33}`)
	})

	t.Run("GetKey", func(t *testing.T) {
		auth := BuildAuthorizationHeader(Authorization{
			Key:       "key",
			Sign:      "4137ecfe066394f7c46e171a0def0b831d9d27971ff1e15825e2294624f44b37",
			Timestamp: _timestamp,
		})

		r, _ := http.NewRequest("POST", s.URL+"?GetKey", strings.NewReader(`{}`))
		r.Header.Set(webapi.HttpHeaderContentType, webapi.ContentTypeJson)
		r.Header.Set(HttpHeaderAuthorization, auth)

		testRequest(t, r, `{"Code":0,"Message":"","Data":"key"}`)
	})
}
