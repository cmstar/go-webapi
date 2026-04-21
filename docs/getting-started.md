# Getting Started

本文档是 go-webapi 的入门教程，将带你从零搭建一个可运行的 SlimAPI 服务。

## 前置条件

- Go 1.24 或更高版本。

## 安装

```bash
go get -u github.com/cmstar/go-webapi@latest
```

## 完整示例

下面是一个完整的服务端程序：

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
	// 1. 创建 SlimAPI 的 Handler。
	slim := slimapi.NewSlimApiHandler("demo")

	// 2. 注册 API 方法——将 Methods 结构体上的所有导出方法注册为 WebAPI。
	slim.RegisterMethods(Methods{})

	// 3. 初始化日志。
	logger := logx.NewStdLogger(nil)
	logFinder := logx.NewSingleLoggerLogFinder(logger)

	// 4. 创建引擎并注册路由。
	e := webapi.NewEngine()
	e.Handle("/api/{~method}", slim, logFinder)

	// 5. 启动 HTTP 服务。
	err := http.ListenAndServe(":15001", e)
	if err != nil {
		logger.Log(logx.LevelFatal, err.Error())
	}
}

type Methods struct{}

// 两数相加。
func (Methods) Plus(req struct {
	A int
	B int
}) int {
	return req.A + req.B
}

// 获取当前时间。
func (Methods) Time() struct {
	Time string
} {
	return struct {
		Time string
	}{
		Time: time.Now().Format("2006-01-02 15:04"),
	}
}

// 获取请求头。演示如何使用 *webapi.ApiState 参数。
func (Methods) Headers(state *webapi.ApiState) map[string][]string {
	return state.RawRequest.Header
}

// 演示错误处理。
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

下面逐步说明。

### 第 1 步：创建 Handler

```go
slim := slimapi.NewSlimApiHandler("demo")
```

`NewSlimApiHandler` 创建一个实现 SlimAPI 协议的 `ApiHandler`。参数 `"demo"` 是 Handler 的名称，用于日志分区。

### 第 2 步：注册方法

```go
slim.RegisterMethods(Methods{})
```

`RegisterMethods` 将 `Methods` 结构体上的所有导出方法注册为 WebAPI。每个方法对应一个 API 接口，方法名就是接口名。

### 第 3 步：初始化日志

```go
logger := logx.NewStdLogger(nil)
logFinder := logx.NewSingleLoggerLogFinder(logger)
```

框架通过 [go-logx](https://github.com/cmstar/go-logx) 库进行日志记录。这个库对日志操作做了抽象，定义了一组日志接口和语义，并没有提供完整的日志输出实现。
这里使用了其内置的 `NewStdLogger(nil)` 创建一个基于 `stdout` 的日志记录器。在正式的项目里，通常需要自己实现 `logx.Logger` 接口，将其对接到自己的日志底层。

### 第 4 步：路由注册

```go
e := webapi.NewEngine()
e.Handle("/api/{~method}", slim, logFinder)
```

`ApiEngine` 是基于 [chi](https://github.com/go-chi/chi) 的 HTTP 服务器，使用 `chi` 的路由语法。

`{~method}` 是框架约定的参数名，表示 Web API 方法的名称。若注册了 `struct` 类型——上文的 `Methods`——框架会自动从中提取公开方法，作为方法名称。

### 第 5 步：启动服务

```go
err := http.ListenAndServe(":15001", e)
```

## 调用示例

服务启动后，可以通过 HTTP 调用注册的方法。方法名称和参数都是**大小写不敏感**的。

### GET 请求

无参数的方法：
```
GET http://localhost:15001/api/time
```

响应：

```json
{"Code":0,"Message":"","Data":{"Time":"2022-03-06 23:16"}}
```

通过 query-string 传递参数：
```
GET http://localhost:15001/api/plus?a=11&b=22
```

响应：

```json
{"Code":0,"Message":"","Data":33}
```

### 使用 POST 请求

JSON 格式：
```
POST http://localhost:15001/api/plus
Content-Type: application/json

{"a":11,"b":22}
```

表单格式：
```
POST http://localhost:15001/api/plus
Content-Type: application/x-www-form-urlencoded

a=11&b=22
```

上面的请求均得到相同的响应：

```json
{"Code":0,"Message":"","Data":33}
```

### 调用使用了 `*webapi.ApiState` 参数的方法

```
GET http://localhost:15001/api/headers
```

响应：

```json
{
    "Code": 0,
    "Message": "",
    "Data": {
        "Accept": ["text/html,application/xhtml+xml"],
        "User-Agent": ["Mozilla/5.0 ..."]
    }
}
```

### 错误处理

返回 `errx.BizError` 时，错误码和消息会体现在响应中：

```
GET http://localhost:15001/api/err?bizErr=true&value=my-value
```

```json
{"Code":12345,"Message":"your message","Data":"my-value"}
```

返回非 `BizError` 的 `error` 或 panic 时，统一返回 500：

```
GET http://localhost:15001/api/err?bizErr=false&value=my-value
```

```json
{"Code":500,"Message":"internal error","Data":""}
```

## 方法注册规则

### 名称约定

通过 `RegisterMethods` 注册方法时：
```go
slim.RegisterMethods(Methods{})
```

只有大写字母开头（ exported ）的方法才会被注册：

```go
type Methods struct{}

func (Methods) Plus(...)    // 会被注册。
func (Methods) plus(...)    // 不会被注册（ unexported ）。
```

方法名称有一组基于双下划线（`__`）的约定：
| 方法名        | 注册的 API 名称 | 说明                                |
| ------------- | --------------- | ----------------------------------- |
| `Plus`        | `Plus`          | 原样注册。                          |
| `GetName__13` | `13`            | 双下划线后的部分作为 API 名称。     |
| `Do____a_B`   | `__a_B`         | 首次双下划线后的部分作为 API 名称。 |
| `Internal__`  | （不注册）      | 双下划线后无有效名称，方法被忽略。  |
| `Helper____`  | （不注册）      | 双下划线后仅有下划线，被忽略。      |

```go
func (Methods) GetName__13(...)    // 会被注册为 API 名称 "13"。
```

此时该方法请求地址变成了：
```
GET http://localhost:15001/api/13
```

### 方法入参约束

注册为 Web API 的方法支持以下参数类型：

| 参数类型           | 说明                                                                                                     |
| ------------------ | -------------------------------------------------------------------------------------------------------- |
| struct             | 字段对应请求参数，字段名大小写不敏感。需兼容 JSON 序列化。                                               |
| `*webapi.ApiState` | 访问当前请求的完整上下文。如在需要时，可通过 `state.RawRequest` 字段访问当前请求的 `http.Request` 对象。 |

两种类型可以同时使用。方法参数表中**同一种类型只能出现一次**，注意：所有未被单独说明的 `struct` 均属于同一种类型。

### 方法返回值约束

请求 Web API 时，固定返回下面的格式：

```json
{
	"Code": 0,
	"Message": "",
	"Data": <dynamic>
}
```

注册为 Web API 的方法支持 0-2 个返回值，返回非 `error` 的数据必须能被序列化为 JSON 格式。

当被调用点的方法未返回错误也未 panic 时，响应的 `Code` 字段为0；否则为对应的错误码。详细的错误处理规则参见 [SlimAPI - 错误处理](slim-api.md#错误处理)。

## 下一步

- [SlimAPI 协议详解](slim-api.md) —— 请求/响应格式、参数处理等完整说明。
  - [接收文件](upload-file.md) —— 描述 SlimAPI 如何通过 `multipart/form-data` 类型的请求传递文件、简单类型和 JSON 数据。
  - [流式输出](streaming.md) —— 描述如何使用 SSE（Server-Sent Events）与 ‌ND-JSON‌（Newline-Delimited JSON）格式的流式响应。
- [框架设计与扩展](architecture.md) —— 管线模型、核心接口、如何定制和扩展框架。
- [SlimAuth](slim-auth.md) —— 带签名校验的 SlimAPI 扩展。
