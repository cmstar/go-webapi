package webapi

import (
	"net/http"
	"strings"

	"github.com/cmstar/go-logx"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// ApiEngine 是一个 http.Handler 。表示一个抽象的 HTTP 服务器，基于 ApiHandler 注册和管理 WebAPI 。
type ApiEngine struct {
	router chi.Router
}

var _ http.Handler = (*ApiEngine)(nil)

// NewEngine 创建一个 ApiEngine 实例，并完成初始化设置。
func NewEngine() *ApiEngine {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	return &ApiEngine{
		router: r,
	}
}

// ServeHTTP implements http.Handler.ServeHTTP().
func (engine *ApiEngine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	engine.router.ServeHTTP(w, r)
}

// Handle 指定一个 ApiHandler ，响应对应 URL 路径下的请求。
// 通过 CreateHandlerFunc(handler, logFinder) 方法创建用于响应请求的过程。
// 返回 ApiSetup ，用于向 ApiHandler 注册 API 方法。
//
// path 为相对路径，以 / 开头。参考 https://github.com/go-chi/chi
//
func (engine *ApiEngine) Handle(path string, handler ApiHandler, logFinder logx.LogFinder) ApiSetup {
	handlerFunc := CreateHandlerFunc(handler, logFinder)

	// 同一个 handler 需要响应不同的请求方式，把需要的都注册一遍。
	methods := handler.SupportedHttpMethods()
	for i := 0; i < len(methods); i++ {
		method := strings.ToLower(methods[i])
		switch method {
		case "get":
			engine.router.Get(path, handlerFunc)
		case "post":
			engine.router.Post(path, handlerFunc)
		case "put":
			engine.router.Put(path, handlerFunc)
		case "delete":
			engine.router.Delete(path, handlerFunc)
		case "patch":
			engine.router.Patch(path, handlerFunc)
		case "head":
			engine.router.Head(path, handlerFunc)
		case "trace":
			engine.router.Trace(path, handlerFunc)
		case "connect":
			engine.router.Connect(path, handlerFunc)
		case "options":
			engine.router.Options(path, handlerFunc)
		}
	}

	return ApiSetup{engine, handler}
}
