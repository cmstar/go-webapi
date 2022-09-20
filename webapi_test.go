// webapi 包定义一组抽象过程与辅助类型，用于开发特定协议的 WebAPI 框架，如 SlimAPI 。

package webapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/cmstar/go-errx"
	"github.com/cmstar/go-logx"
	"github.com/stretchr/testify/require"
)

type emptyApiMethodRegister struct{}

func (emptyApiMethodRegister) RegisterMethod(m ApiMethod) {}

func (emptyApiMethodRegister) RegisterMethods(providerStruct any) {}

func (emptyApiMethodRegister) GetMethod(name string) (method ApiMethod, ok bool) {
	return ApiMethod{
		Name:  "name",
		Value: reflect.ValueOf(func() {}),
	}, true
}

func createHandlerFuncForTest(w *ApiHandlerWrapper) http.HandlerFunc {
	w.HttpMethods = []string{"GET"}
	w.ApiMethodRegister = emptyApiMethodRegister{}

	if w.ApiUserHostResolver == nil {
		w.ApiUserHostResolver = ApiUserHostResolverFunc(func(state *ApiState) {
			state.UserHost = "host"
		})
	}

	if w.ApiNameResolver == nil {
		w.ApiNameResolver = ApiNameResolverFunc(func(state *ApiState) {
			state.Name = "name"
		})
	}

	if w.ApiDecoder == nil {
		w.ApiDecoder = ApiDecoderFunc(func(state *ApiState) {
			state.Args = make([]reflect.Value, 0)
		})
	}

	if w.ApiMethodCaller == nil {
		w.ApiMethodCaller = ApiMethodCallerFunc(func(state *ApiState) {
			state.Data = "data"
		})
	}

	if w.ApiResponseBuilder == nil {
		w.ApiResponseBuilder = ApiResponseBuilderFunc(func(state *ApiState) {
			state.Response = &ApiResponse[any]{Data: state.Data}
		})
	}

	if w.ApiResponseWriter == nil {
		w.ApiResponseWriter = ApiResponseWriterFunc(func(state *ApiState) {
			state.ResponseContentType = "custom"
			state.ResponseBody = strings.NewReader("body")
		})
	}

	if w.ApiLogger == nil {
		w.ApiLogger = ApiLoggerFunc(func(state *ApiState) {
			if state.Logger != nil {
				state.Logger.Log(state.LogLevel, "msg")
			}
		})
	}

	handlerFunc := CreateHandlerFunc(w, logx.DefaultManager)
	return handlerFunc
}

func TestCreateHandlerFunc(t *testing.T) {
	uri, _ := url.Parse("http://temp.org")
	handlerFunc := createHandlerFuncForTest(&ApiHandlerWrapper{})

	recorder := httptest.NewRecorder()
	handlerFunc.ServeHTTP(recorder, &http.Request{
		URL: uri,
	})
}

func TestCreateHandlerFunc_panic(t *testing.T) {
	uri, _ := url.Parse("http://temp.org")

	t.Run("ApiUserHostResolver_StackfulError", func(t *testing.T) {
		var s *ApiState

		handlerFunc := createHandlerFuncForTest(&ApiHandlerWrapper{
			ApiUserHostResolver: ApiUserHostResolverFunc(func(state *ApiState) {
				s = state
				panic(errx.Wrap("stackful", nil))
			}),
		})
		recorder := httptest.NewRecorder()
		handlerFunc.ServeHTTP(recorder, &http.Request{URL: uri})

		require.Error(t, s.Error)
		require.Regexp(t, "stackful", s.Error.Error())
	})

	t.Run("ApiNameResolver_error", func(t *testing.T) {
		var s *ApiState

		handlerFunc := createHandlerFuncForTest(&ApiHandlerWrapper{
			ApiNameResolver: ApiNameResolverFunc(func(state *ApiState) {
				s = state
				panic(errors.New("msg"))
			}),
		})
		recorder := httptest.NewRecorder()
		handlerFunc.ServeHTTP(recorder, &http.Request{URL: uri})

		require.Error(t, s.Error)
		require.Regexp(t, "msg", s.Error.Error())
	})

	t.Run("ApiDecoder_string", func(t *testing.T) {
		var s *ApiState

		handlerFunc := createHandlerFuncForTest(&ApiHandlerWrapper{
			ApiDecoder: ApiDecoderFunc(func(state *ApiState) {
				s = state
				panic("string")
			}),
		})
		recorder := httptest.NewRecorder()
		handlerFunc.ServeHTTP(recorder, &http.Request{URL: uri})

		require.Error(t, s.Error)
		require.Regexp(t, "string", s.Error.Error())
	})

	t.Run("other", func(t *testing.T) {
		var s *ApiState
		p := false

		handlerFunc := createHandlerFuncForTest(&ApiHandlerWrapper{
			ApiResponseBuilder: ApiResponseBuilderFunc(func(state *ApiState) {
				s = state

				// ApiResponseBuilder 有两次调用，一次是正常流程，一次是 panic 后用于处理错误的。
				// 这里让正常流程 panic ，错误处理流程（第二次）则不会报错，否则整个请求就崩溃了。
				if p {
					return
				}

				p = true
				type msg string
				panic(msg("msg"))
			}),
		})
		recorder := httptest.NewRecorder()
		handlerFunc.ServeHTTP(recorder, &http.Request{URL: uri})

		require.Error(t, s.Error)
		require.Regexp(t, "msg", s.Error.Error())
	})
}
