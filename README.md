# webapi

[![GoDoc](https://godoc.org/github.com/cmstar/go-webapi?status.svg)](https://pkg.go.dev/github.com/cmstar/go-webapi)
[![Go](https://github.com/cmstar/go-webapi/workflows/Go/badge.svg)](https://github.com/cmstar/go-webapi/actions?query=workflow%3AGo)
[![codecov](https://codecov.io/gh/cmstar/go-webapi/branch/master/graph/badge.svg)](https://codecov.io/gh/cmstar/go-webapi)
[![License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat)](https://opensource.org/licenses/MIT)
[![GoVersion](https://img.shields.io/github/go-mod/go-version/cmstar/go-webapi)](https://github.com/cmstar/go-webapi/blob/main/go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/cmstar/go-webapi)](https://goreportcard.com/report/github.com/cmstar/go-webapi)

Golang 开发的极轻量、傻瓜化 WebAPI 框架，将通信协议与业务代码完全解耦，让开发者专注于业务逻辑。

## 快速使用

安装：
```
go get -u github.com/cmstar/go-webapi@latest
```

上代码：
```go
package main

import (
	"net/http"

	"github.com/cmstar/go-logx"
	"github.com/cmstar/go-webapi"
	"github.com/cmstar/go-webapi/slimapi"
)

func main() {
	slim := slimapi.NewSlimApiHandler("demo")
	slim.RegisterMethods(Methods{})

	logger := logx.NewStdLogger(nil)
	logFinder := logx.NewSingleLoggerLogFinder(logger)

	e := webapi.NewEngine()
	e.Handle("/api/{~method}", slim, logFinder)
	http.ListenAndServe(":15001", e)
}

type Methods struct{}

func (Methods) Plus(req struct{ A, B int }) int {
	return req.A + req.B
}

func (Methods) Multiply(req struct{ A, B int }) int {
	return req.A * req.B
}
```

跑起来，现在可以调用 `Methods` 上的方法了。方法名称和参数都是大小写不敏感的。

```
GET http://localhost:15001/api/plus?a=11&b=22

=> {"Code":0,"Message":"","Data":33}

---

POST http://localhost:15001/api/multiply
Content-Type: application/json

{"a":11,"b":22}

=> {"Code":0,"Message":"","Data":242}
```

上面的示例中，业务代码 `Plus` 和 `Multiply` 没有耦合 HTTP 协议，通信部分完全由框架处理。

更完整的用法和说明，请参阅 [`docs/`](docs/) 目录。

## 其他语言的版本

- .net 版： [SlimAPI](https://pkg.go.dev/github.com/cmstar/go-webapi/slimapi)
