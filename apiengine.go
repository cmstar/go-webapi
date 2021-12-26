package webapi

import (
	"github.com/cmstar/go-logx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// ApiEngine 表示一个抽象的 HTTP 服务器，基于 ApiHandler 注册和管理 WebAPI 。
type ApiEngine struct {
	echo *echo.Echo
}

// NewEngine 创建一个 ApiEngine 实例，并完成初始化设置。
// 自动生成并绑定 echo 实例。
func NewEngine() *ApiEngine {
	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.Logger()) // TODO set log format
	return NewEngineFromEcho(e)
}

// NewEngineFromEcho 创建一个 ApiEngine 实例，并绑定给定的 echo 实例。
func NewEngineFromEcho(e *echo.Echo) *ApiEngine {
	engine := new(ApiEngine)
	engine.echo = e
	// TODO Handle panic via the echo middleware.
	return engine
}

// Start 在指定的地址开启 HTTP 服务，开始监听端口并响应请求。在完成各个 API 注册后，最后调用此方法开启服务。
//
// addr 地址格式为 IP:PORT ，监听来自于特定 IP ，对于特定端口的请求；若不指定 IP 地址，省略 IP 部分，格式为 :PORT 。
// 如“:12345”监听任何来源对于 12345 端口的请求，“127.0.0.1:12345”则仅监听本机。
//
func (engine *ApiEngine) Start(addr string) {
	engine.echo.Logger.Fatal(engine.echo.Start(addr))
}

// Handle 指定一个 ApiHandler ，响应对应 URL 路径下的请求。
// 通过 CreateHandlerFunc(handler, logFinder) 方法创建用于响应请求的过程。
// 返回 ApiSetup ，用于向 ApiHandler 注册 API 方法。
//
// path 为相对路径，以 / 开头。参考 https://echo.labstack.com/guide/routing/
//
func (engine *ApiEngine) Handle(path string, handler ApiHandler, logFinder logx.LogFinder) ApiSetup {
	handlerFunc := CreateHandlerFunc(handler, logFinder)

	// 同一个 handler 需要响应不同的请求方式，把需要的都注册一遍。
	methods := handler.SupportedHttpMethods()
	for i := 0; i < len(methods); i++ {
		switch methods[i] {
		case "GET":
			engine.echo.GET(path, handlerFunc)
		case "POST":
			engine.echo.POST(path, handlerFunc)
		case "PUT":
			engine.echo.PUT(path, handlerFunc)
		case "DELETE":
			engine.echo.DELETE(path, handlerFunc)
		}
	}

	return ApiSetup{engine, handler}
}
