# webapi

[![GoDoc](https://godoc.org/github.com/cmstar/go-conv?status.svg)](https://pkg.go.dev/github.com/cmstar/go-webapi)
[![Go](https://github.com/cmstar/go-webapi/workflows/Go/badge.svg)](https://github.com/cmstar/go-webapi/actions?query=workflow%3AGo)
[![codecov](https://codecov.io/gh/cmstar/go-webapi/branch/master/graph/badge.svg)](https://codecov.io/gh/cmstar/go-webapi)
[![License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat)](https://opensource.org/licenses/MIT)

这是[SlimWebApi](https://github.com/cmstar/SlimWebApi)的 Go 版。 SlimAPI 通信协议详见[godoc-SlimAPI通信协议](https://pkg.go.dev/github.com/cmstar/go-webapi/slimapi#pkg-overview)。

## 快速使用

安装：
```
go get -u github.com/cmstar/go-webapi@latest
```

上代码：
```go
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cmstar/go-errx"
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
	e.Handle("/api/{~method}", slim, logFinder)

	// 启动。
	err := http.ListenAndServe(":15001", e)
	if err != nil {
		logger.Log(logx.LevelFatal, err.Error())
	}
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

// 参数也可以是 *webapi.ApiState ，可通过其访问到当前请求的上下文。
// 也可以同时搭配 struct 型的参数。
func (Methods) Headers(state *webapi.ApiState, req struct{ NoMeaning bool }) map[string][]string {
	// RawRequest 是标准库中，当前请求的 *http.Request 。
	return state.RawRequest.Header
}

// 支持至多两个返回值，第一个返回值对应输出的 Data 字段；第二个返回值必须是 error 。详见《错误处理》节。
func (Methods) Err(req struct {
	BizErr bool
	Value  string
}) (string, error) {
	if req.BizErr {
		return req.Value, errx.NewBizError(12345, "your message", nil)
	}
	return "", fmt.Errorf("not a biz-error: %v", req.Value)
}
```

跑起来，现在可以调用 `Methods` 上的方法了。方法名称和参数都是大小写不敏感的。

```
GET http://localhost:15001/api/plus?a=11&b=22

=> {
    "Code": 0,
    "Message": "",
    "Data": 33
}

---
# 以 JSON 格式请求。
POST http://localhost:15001/api/plus
Content-Type: application/json

{"a":11, "b":22}

=> {
    "Code": 0,
    "Message": "",
    "Data": 33
}

---
GET http://localhost:15001/api/time

=> {
    "Code": 0,
    "Message": "",
    "Data": "2022-03-06 23:16"
}

---
GET http://localhost:15001/api/headers

=> {
    "Code": 0,
    "Message": "",
    "Data": {
        "Accept": ["text/html,application/xhtml+xml"],
        "Accept-Encoding": ["gzip, deflate, br"],
        "Connection": ["keep-alive"],
        "User-Agent": ["Mozilla/5.0 ..."],
        ...
    }
}

---
# BizError 会以 Code + Message 的方式体现在输出上。
GET http://localhost:15001/api/err?bizErr=1&value=my-value

=> {
    "Code": 12345,
    "Message": "your message",
    "Data": "my-value"
}

---
# 非 BizError 均表现为 internal error 。
GET http://localhost:15001/api/err?bizErr=false&value=my-value

=> {
    "Code": 500,
    "Message": "internal error",
    "Data": ""
}
```

## 错误处理

表示 WebAPI 的方法支持0-2个返回值（详见[GoDoc](https://pkg.go.dev/github.com/cmstar/go-webapi#ApiMethodRegister)）。

当方法返回：
- 没有 `error` 返回值或返回的 `error` 为 `nil`：表示调用成功，输出的 `Code=0` 。
- 返回 `errx.BizError`：输出 `Code=BizError.Code(), Message=BizError.Message()` 。
- 返回不是 `errx.BizError` 的 `error`：统一输出 `Code=500, Message=internal error` 。
- 方法 `panic`：统一输出 `Code=500, Message=internal error` 。

> `BizError` 的详细说明，参考[go-errx库](https://github.com/cmstar/go-errx#bizerror)。
