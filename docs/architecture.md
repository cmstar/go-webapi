# 框架设计与扩展

本文档介绍 go-webapi 框架的整体设计，包括管线模型、核心接口，以及如何基于框架进行定制和扩展。

## 包关系

```
webapi         基础管线和接口定义。
  ↑
slimapi        SlimAPI 协议的实现。
  ↑
slimauth       基于 SlimAPI 扩展的签名校验协议。
```

- `webapi`（根包）定义了处理请求的管线模型和一组抽象接口，不绑定任何具体协议。
- `slimapi` 是框架默认提供的协议实现，基于 `webapi` 的接口实现了 [SlimAPI 通信协议](slim-api.md)。
- `slimauth` 在 `slimapi` 之上叠加了 HMAC-SHA256 签名校验，详见 [SlimAuth](slim-auth.md)。

如果需要实现自定义协议（如 JWT 认证、加密传输等），通常以 `slimapi` 为基座，替换其中的部分组件即可。

---

## 管线模型

每个 HTTP 请求的处理被分解为一条**管线**（pipeline）。贯穿整条管线的是 `[ApiState](#apistate)`，它承载了一个请求从开始到结束的所有状态数据。

整体流程体现在 `webapi.CreateHandlerFunc()` 函数中，它将一个 `ApiHandler` 接口的实例转换为标准的 `http.HandlerFunc`。

管线调用步骤如下：
```
HTTP Request
  │
  ▼
┌──────────────────────┐
│  ApiUserHostResolver │  FillUserHost：解析客户端 IP。
├──────────────────────┤
│  ApiNameResolver     │  FillMethod：解析 API 方法名称（可含签名校验、认证等前置逻辑）。
├──────────────────────┤
│  (GetMethod)         │  由 ApiMethodRegister.GetMethod() 根据 Name 检索 Method；失败则填 Error，跳过 Decode/Call。
├──────────────────────┤
│  ApiDecoder          │  Decode：从请求中构建方法参数。
├──────────────────────┤
│  ApiMethodCaller     │  Call：调用目标方法。
├──────────────────────┤
│  ApiResponseWriter   │  WriteResponse：写入 ApiState.Response* 字段，期间可调用 ApiResponseBuilder.BuildResponse() 。
├──────────────────────┤
|  (Response)          |  --> HTTP Response
├──────────────────────┤
│  ApiLogger           │  Log：输出请求日志。此步骤在输出 HTTP 响应流之后，但并不是异步执行的。
└──────────────────────┘
```

非标准管线步骤：
- **ApiMethodRegister**：仅在**初始化**阶段通过 `RegisterMethod` / `RegisterMethods` 注册方法；在**每个请求**里，框架在 `FillMethod` 之后调用 `GetMethod`，根据 `ApiState.Name` 解析出 `ApiState.Method`。若方法不存在，会设置错误并跳过后续的 `Decode` 与 `Call`。
- **ApiResponseBuilder**：在 `ApiResponseWriter.WriteResponse` 执行过程中被调用（例如将 `ApiMethodCaller.Call` 的结果交给 `BuildResponse`），用于组装待序列化的业务结果。

---

## 核心接口

### ApiHandler

`ApiHandler` 是管线中所有接口的**组合体**：

```go
type ApiHandler interface {
    ApiMethodRegister
    ApiUserHostResolver
    ApiNameResolver
    ApiDecoder
    ApiMethodCaller
    ApiResponseBuilder
    ApiResponseWriter
    ApiLogger

    Name() string
    SupportedHttpMethods() []string
}
```

它聚合了管线每个阶段的接口，加上 `Name()`（用于标识和日志分区）和 `SupportedHttpMethods()`（声明支持的 HTTP 方法）。

通常不需要从零实现 `ApiHandler`，而是通过 `ApiHandlerWrapper` 组装各个接口。

### ApiHandlerWrapper

`ApiHandlerWrapper` 是实现 `ApiHandler` 的脚手架，它的每个字段对应管线中的一个接口：

```go
type ApiHandlerWrapper struct {
    ApiMethodRegister
    ApiNameResolver
    ApiUserHostResolver
    ApiDecoder
    ApiMethodCaller
    ApiResponseBuilder
    ApiResponseWriter
    ApiLogger

    HandlerName string
    HttpMethods []string
}
```

这是框架最核心的组装机制。`slimapi.NewSlimApiHandler()` 返回的就是一个 `*ApiHandlerWrapper`，其中每个字段都已填充了 SlimAPI 协议的默认实现。当需要定制时，只要替换其中的目标字段即可。

### 管线接口一览

| 接口                  | 职责                                                                | 对应填充/处理 `ApiState` 字段         |
| --------------------- | ------------------------------------------------------------------- | ------------------------------------- |
| `ApiUserHostResolver` | 解析客户端 IP 地址。                                                | `UserHost`                            |
| `ApiNameResolver`     | 从请求中解析目标方法名称；非法请求可设置 `Error` 以跳过后续阶段。   | `Name`、`Error`（解析失败时）         |
| `ApiMethodRegister`   | 初始化时注册方法；请求时 `GetMethod` 按 `Name` 解析 `Method`。      | `Method`、`Error`（未能定位到方法时） |
| `ApiDecoder`          | 从请求中构建方法的参数。                                            | `Args`                                |
| `ApiMethodCaller`     | 调用方法，获取返回值。                                              | `Data`、`Error`                       |
| `ApiResponseBuilder`  | 将单次调用结果组装为待写出对象（通常由 `WriteResponse` 内部调用）。 | —                                     |
| `ApiResponseWriter`   | 根据管线结果填充响应字段，供后续写入 HTTP。                         | `ResponseBody`、`ResponseContentType` |
| `ApiLogger`           | 在响应体写出之后输出请求日志。                                      | `LogLevel`、`LogMessage`              |

每个接口均提供了对应的函数适配器（如 `ApiNameResolverFunc`、`ApiDecoderFunc`），可以直接用函数实现接口，无需定义结构体。

### ApiState

`ApiState` 是每个请求独立的状态对象，它贯穿整条管线。各阶段从中读取所需数据，并将处理结果写回。

关键字段包括：

| 字段                         | 说明                                                             |
| ---------------------------- | ---------------------------------------------------------------- |
| `RawRequest` / `RawResponse` | 原始的 `*http.Request` 和 `http.ResponseWriter`。                |
| `Query`                      | 按 ASP.NET 风格解析的 URL 参数。                                 |
| `Handler`                    | 当前的 `ApiHandler`。                                            |
| `Logger`                     | 当前请求的日志记录器。                                           |
| `Name`                       | API 方法名称。                                                   |
| `Method`                     | 已注册的目标方法。                                               |
| `Args`                       | 调用方法所需的参数。                                             |
| `Data`                       | 方法返回的非 error 值。                                          |
| `Error`                      | 管线中产生的错误（包括 panic 后被捕获的错误）。                  |
| `ResponseBody`               | HTTP 响应 body，是一个 `iter.Seq[[]byte]` 迭代器，支持流式输出。 |
| `ResponseContentType`        | HTTP 响应的 Content-Type。                                       |
| `LogLevel` / `LogMessage`    | 日志级别和内容缓冲。                                             |

#### CustomData：跨阶段通信

`ApiState` 提供 `SetCustomData(key, value)` 和 `GetCustomData(key)` 方法，用于在管线的不同阶段之间传递自定义数据。其原理类似 `context.WithValue`。

这在扩展协议中非常常用。例如：
- SlimAuth 的 `ApiNameResolver` 在校验签名后，将解析得到的 `Authorization` 存入 CustomData；后续的 `ApiDecoder` 和 `ApiLogger` 从中读取。
- 类似地，自定义协议可以在 `ApiNameResolver` 阶段完成认证，将认证结果存入 CustomData，供后续阶段使用。

和 `context.WithValue` 的用法一致，推荐使用**未导出的类型**作为 key，以避免不同包之间的 key 冲突，：

```go
var myKey = func() any { type t struct{}; return t{} }()

state.SetCustomData(myKey, authResult)
```

---

## 管线扩展机制

除了直接替换整个管线接口的实现，框架还提供了两种**管道**（pipeline），支持细粒度的扩展。

### ArgumentDecoderPipeline

`ArgumentDecoderPipeline` 实现了 `ApiDecoder` 接口，内部是一个 `ArgumentDecoder` 的有序列表。`NewArgumentDecoderPipeline()` 创建管道时，第一个元素固定是内置的 `ApiStateArgumentDecoder`（用于为 `*ApiState` 类型的参数赋值）。

```go
type ArgumentDecoder interface {
    DecodeArg(state *ApiState, index int, argType reflect.Type) (ok bool, v any, err error)
}
```

工作方式：对方法参数表中的每个参数，按顺序询问管道内的每个 `ArgumentDecoder`——谁能解析就由谁处理。这使得不同类型的参数可以由不同的解码器负责。

例如 SlimAuth 的 Decoder 管道：

```go
func NewSlimAuthApiDecoder() webapi.ApiDecoder {
    return webapi.NewArgumentDecoderPipeline(
        authorizationArgumentDecoder{}, // 解析 Authorization 类型的参数。
        slimapi.StructArgumentDecoder,  // 解析 struct 类型的参数。
    )
}
```

这里 `authorizationArgumentDecoder` 只负责处理 `Authorization` 类型的参数，其余 struct 参数交给 `slimapi.StructArgumentDecoder`。

### LogSetupPipeline

`LogSetupPipeline` 实现了 `ApiLogger` 接口，内部是一个 `LogSetup` 的有序列表。在输出日志时，依次执行每个 `LogSetup`，各自向 `ApiState.LogMessage` 追加日志字段。

```go
type LogSetup interface {
    Setup(state *ApiState)
}
```

例如 SlimAuth 在 SlimAPI 的日志管道中插入了 `LogAuthorization`，用于额外记录 AccessKey 和 Timestamp：

```go
func NewSlimAuthApiLogger() webapi.LogSetupPipeline {
    pipe := slimapi.NewSlimApiLogger()
    // 在 LogBody 前面插入 LogAuthorization。
    // ...（找到 LogBody 的位置，在其前面插入）
    return pipe
}
```

## 定制与扩展：以 SlimAuth 为例

SlimAuth 是框架自带的扩展协议，它很好地展示了如何基于 SlimAPI 定制自己的协议。其核心模式是：

1. **以 SlimAPI 为基座**，获取 `*ApiHandlerWrapper`。
2. **按需替换组件**。

```go
func NewSlimAuthApiHandler(op SlimAuthApiHandlerOption) *webapi.ApiHandlerWrapper {
    h := slimapi.NewSlimApiHandler(op.Name)
    h.ApiNameResolver = NewSlimAuthApiNameResolver(op.AuthScheme, op.SecretFinder, timeChecker)
    h.ApiDecoder = NewSlimAuthApiDecoder()
    h.ApiLogger = NewSlimAuthApiLogger()
    return h
}
```

SlimAuth 替换了三个组件，其余组件（`ApiMethodCaller`、`ApiResponseBuilder`、`ApiResponseWriter` 等）沿用 SlimAPI 的默认实现。

### 常见扩展模式

从 SlimAuth 及实际项目中的扩展实践，可以总结出以下通用模式：

#### 1. NameResolver 承担前置校验

认证/签名校验通常放在 `ApiNameResolver` 阶段。校验通过后，委托给 SlimAPI 的 resolver 解析方法名：

```go
func (x slimAuthApiNameResolver) FillMethod(state *webapi.ApiState) {
    x.verifySignature(state)   // 先做签名校验。
    x.raw.FillMethod(state)    // 再委托给 SlimAPI 解析方法名。
}
```

校验失败时，通过 panic 一个 `BadRequestError` 或 `errx.BizError` 中止管线。

#### 2. Decoder 注入协议特有类型

通过在 `ArgumentDecoderPipeline` 的前端插入自定义的 `ArgumentDecoder`，使 API 方法可以直接声明协议特有的参数类型。例如：

```go
// SlimAuth 允许方法直接声明 Authorization 参数。
func (Methods) MyApi(auth slimauth.Authorization, req struct{ Name string }) string {
    fmt.Println(auth.Key)
    return req.Name
}
```

自定义 `ArgumentDecoder` 只需要检查参数类型是否匹配，匹配则从 `ApiState.GetCustomData()` 中取出之前存好的认证结果并返回：

```go
func (d authorizationArgumentDecoder) DecodeArg(
    state *webapi.ApiState, index int, argType reflect.Type,
) (ok bool, v any, err error) {
    if argType != reflect.TypeOf(Authorization{}) {
        return false, nil, nil // 不是目标类型，跳过。
    }
    auth := MustGetBufferedAuthorization(state)
    return true, auth, nil
}
```

#### 3. Logger 插入协议字段

获取 SlimAPI 的 `LogSetupPipeline`，在合适的位置插入自定义的 `LogSetup`，以记录协议相关的信息。通常插入在 body 或 error 之前：

```go
pipe := slimapi.NewSlimApiLogger()
// 找到目标位置并插入。
```

#### 4. CustomData 是跨阶段通信的桥梁

NameResolver 阶段解析的认证/协议信息，通过 `ApiState.SetCustomData()` 存储，后续的 Decoder 和 Logger 通过 `GetCustomData()` 读取。这套机制使各阶段之间保持解耦。

### 你并非必须替换所有组件

不同的扩展需求可能只需要替换不同的组件：

| 场景                 | 需要替换的组件                                       |
| -------------------- | ---------------------------------------------------- |
| 新增认证/签名校验    | `ApiNameResolver` + `ApiDecoder` + `ApiLogger`       |
| 仅修改参数解析方式   | `ApiDecoder`                                         |
| 仅添加额外的日志字段 | `ApiLogger`                                          |
| 修改响应格式         | `ApiResponseWriter`（可能还有 `ApiResponseBuilder`） |

## ApiEngine 与路由

`ApiEngine` 是框架提供的 HTTP 服务器，基于 [chi](https://github.com/go-chi/chi) 路由库。它将 `ApiHandler` 注册到指定的 URL 路径上：

```go
e := webapi.NewEngine()
e.Handle("/api/{~method}", handler, logFinder)
```

`NewEngine()` 创建引擎时，内置了 chi 的 `RealIP` 和 `Recoverer` 中间件。

`Handle()` 方法会根据 `ApiHandler.SupportedHttpMethods()` 的返回值，自动在 chi 上注册对应的 HTTP 方法路由。

`ApiEngine` 本身实现了 `http.Handler`，可以直接传给 `http.ListenAndServe()`。
