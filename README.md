# webapi

[![Go](https://github.com/cmstar/go-webapi/workflows/Go/badge.svg)](https://github.com/cmstar/go-webapi/actions?query=workflow%3AGo)
[![codecov](https://codecov.io/gh/cmstar/go-webapi/branch/master/graph/badge.svg)](https://codecov.io/gh/cmstar/go-webapi)
[![License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat)](https://opensource.org/licenses/MIT)

这是[SlimWebApi](https://github.com/cmstar/SlimWebApi)的 Go 版。

待完善。

## 基础使用

```go
package main

import (
	"net/http"
	"time"

	"github.com/cmstar/go-logx"
	"github.com/cmstar/go-webapi"
	"github.com/cmstar/go-webapi/slimapi"
)

func main() {
	// 初始化 API 容器。
	slim := slimapi.NewSlimApiHandler("demo")

	// 注册 WebAPI 方法。
	slim.RegisterMethods(Methods{})

	// 初始化日志。 https://github.com/cmstar/go-logx
	logger := logx.NewStdLogger(nil)
	logFinder := logx.NewSingleLoggerLogFinder(logger)

	// 初始化引擎。
	e := webapi.NewEngine()

	// 注册路由，使用 chi 库的语法。 https://github.com/go-chi/chi
	e.Handle("/api/{~method}/", slim, logFinder)

	// 启动。
	logger.Log(logx.LevelFatal, http.ListenAndServe(":15001", e).Error())
}

// 用于承载 WebAPI 方法。
type Methods struct{}

// 方法必须是 exported ，即大写字母开头的。
func (Methods) Plus(req struct {
	A int // 参数首字母也必须是大写的。
	B int
}) int {
	return req.A + req.B
}

func (Methods) Time() string {
	return time.Now().Format("2006-01-02 15:04")
}
```

跑起来，现在可以调用 Plus 和 Time 方法了。方法名称和参数都是大小写不敏感的。

```
GET http://localhost:15001/api/plus/?a=11&b=22

=> {
    "Code": 0,
    "Message": "",
    "Data": 33
}
```

```
GET http://localhost:15001/api/time/?a=1&b=2

=> {
    "Code": 0,
    "Message": "",
    "Data": "2022-03-06 23:16"
}
```