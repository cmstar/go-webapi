# 破坏性更新

从旧版本升级时，请阅读高于当前版本、且不高于目标版本的各节。未覆盖的历史版本不代表没有破坏性变更。

> 本文档不包含 v0.8.3 之前的版本。

## v0.9.2

### JSONP 回调名称限制收紧

#### 变更内容

JSONP 回调名称现在必须匹配 `[A-Za-z_$][A-Za-z0-9_$]*`，即首字符只能是 ASCII 字母、下划线或美元符号，后续字符还可以是数字。非空且不符合规则的名称会产生 `bad callback` 错误，并以 JSON 格式返回，不再包装为 JSONP。

#### 影响范围

以前使用 `object.callback`、中文名称等回调名称的调用方会失败。内置 SlimAPI / SlimAuth 均受影响。

#### 迁移方式

改用符合上述规则的全局回调名称，例如 `appCallback_1`；原先使用对象方法时，可通过全局函数转发调用。

### 兼容性提醒：请求结束后清理 multipart 临时文件

#### 变更内容

`CreateHandlerFunc` 在请求处理结束时调用 `MultipartForm.RemoveAll()`，修复 chi 复制请求对象后，标准库未能清理上传临时文件的问题。

#### 影响范围

若代码依赖旧版本遗留的磁盘临时文件，在请求结束后或后台任务中继续调用 `FileHeader.Open()`，现在可能因文件已删除而失败。请求期间读取上传文件的正常用法不受影响。

#### 迁移方式

需要延后处理的文件，应在请求结束前复制到自行管理的存储中，并由业务代码负责其生命周期；不要将请求所属的临时文件作为持久存储。

---

## v0.9.1

### 客户端 IP 来源和 `RemoteAddr` 行为变化

#### 变更内容

`NewEngine()` 不再安装 chi 的 `RealIP` 中间件。旧中间件按 `True-Client-IP`、`X-Real-IP`、`X-Forwarded-For` 的优先级选取非空头，地址有效时会改写 `http.Request.RemoteAddr`。

现在 `NewBasicApiUserHostResolver` 默认只读取 `X-Forwarded-For` 的第一个 IP；没有有效值时回退到 `RemoteAddr`。解析结果填入 `ApiState.UserHost`，不会改写 `RawRequest.RemoteAddr`，也不再读取 `True-Client-IP` 或 `X-Real-IP`。XFF 地址会去除首尾空白、将 IPv4 映射地址还原为 IPv4，并移除 IPv6 zone。

#### 影响范围

- 代理只设置 `True-Client-IP` / `X-Real-IP` 时，默认 `UserHost` 可能变为代理地址；同时设置多个头时，选取的地址也可能改变。
- 日志、鉴权或限流代码若直接读取 `RawRequest.RemoteAddr`，现在通常得到连接对端地址，而不是旧中间件改写后的客户端 IP。
- 自行调用 `CreateHandlerFunc`、未使用旧引擎中间件的项目，现在默认解析器也会读取 XFF。
- 普通无参调用 `NewBasicApiUserHostResolver()` 仍可编译，但构造函数改为可变参数函数，不能再直接赋给 `func() webapi.ApiUserHostResolver` 类型的变量。

#### 迁移方式

1. 核对代理实际设置的头，使用默认模式时，由受信任的代理控制 XFF。需要沿用其他头的规则时，自定义 `ApiUserHostResolver`。
2. 业务需要解析后的客户端地址时读取 `state.UserHost`；需要 TCP 对端地址时读取 `RawRequest.RemoteAddr`。
3. 若希望默认解析器忽略代理头，显式配置：

```go
webapi.NewBasicApiUserHostResolver(webapi.BasicApiUserHostResolverOp{
	Source: webapi.RemoteAddr,
})
```

4. 若将构造函数作为无参工厂传递，用无参闭包包装调用。

---

## v0.9.0

从 v0.8.4 升到 v0.9.0 时，下列变更都会破坏兼容性。请按条目逐项处理。流式响应（`EventStream` / `NdJson`）是本版本新增能力，本身不是破坏性变更；只有下面这些旧 API / 旧行为的调整需要迁移。

### 最低 Go 版本提升到 1.24

#### 变更内容

v0.8.4 的 `go.mod` 要求 Go 1.18。v0.9.0 将最低版本提升到 Go 1.24（中间曾先升到 1.23 以使用迭代器）。

#### 影响范围

本模块及依赖它的项目需要使用 Go 1.24 或更高版本的工具链构建；旧版工具链若支持并启用了自动切换，可能自动获取符合要求的工具链。

#### 迁移方式

把开发、构建和 CI 工具链升到 Go 1.24 或更高版本，再执行 `go mod tidy`。仅部署已经编译好的程序时，通常不需要安装 Go 工具链。

### `ApiState.ResponseBody` 改为迭代器

#### 变更内容

`ApiState.ResponseBody` 的类型从 `io.Reader` 改为 `iter.Seq[[]byte]`。框架按迭代轮次写出 HTTP body；若 `http.ResponseWriter` 实现了 `http.Flusher`，每一轮写出后会立刻 flush。

在 v0.9.0 之前：

- `ResponseBody` 是 `io.Reader`，框架用一次性拷贝写出整个 body。
- 若值还实现了 `io.Closer`，写出结束后会自动 `Close()`。

从 v0.9.0 起：

- `ResponseBody` 是 `func(yield func([]byte) bool)`。每次 `yield` 一段数据；返回 `false` 表示中止迭代。
- 长度为 0 的片段不会写出，迭代继续。
- 框架不再根据 `io.Closer` 自动关闭资源，需要在迭代器内部自行释放。
- 每个非空片段写出后都会尝试 flush，普通非流式响应也适用，不限于 SSE / NDJSON。

#### 影响范围

自定义 `ApiResponseWriter`、直接读写 `ApiState.ResponseBody`，或把 `bytes.Buffer`、`strings.Reader`、文件等 `io.Reader` 赋给该字段的代码，升级后无法编译。

若中间件实现了 `http.Flusher`，并依赖缓冲整个响应或在处理器返回后修改尚未发送的响应头，也需要检查提前 flush 的影响。

只使用内置 SlimAPI / SlimAuth、且没有自行填充 `ResponseBody` 的项目，一般不必改业务代码。若要做流式接口，应使用 `webapi.EventStream` / `webapi.NdJson`，详见 [`streaming.md`](streaming.md)。

#### 迁移方式

一次性写出整段 body 时，把 `io.Reader` 换成只 `yield` 一次的迭代器：

```go
// 旧：
state.ResponseBody = bytes.NewReader(payload)

// 新：
state.ResponseBody = func(yield func([]byte) bool) {
	yield(payload)
}
```

原先依赖自动 `Close()` 的 `io.ReadCloser`，改为在迭代器里关闭：

```go
state.ResponseBody = func(yield func([]byte) bool) {
	defer file.Close()
	// 按块 yield(file 读出的数据)。
}
```

读取 `ResponseBody` 时，不要再用 `io.ReadAll`，改为迭代：

```go
var body []byte
for chunk := range state.ResponseBody {
	body = append(body, chunk...)
}
```

### `ApiResponseBuilder` 不再作为管线步骤

#### 变更内容

`ApiResponseBuilder.BuildResponse` 不再由框架在写出响应前自动调用，改为由 `ApiResponseWriter.WriteResponse`（或其他自定义逻辑）按需调用。方法签名和 `ApiState.Response` 字段一并调整。

在 v0.9.0 之前：

- 管线顺序为 `… → ApiMethodCaller → ApiResponseBuilder → ApiResponseWriter → ApiLogger`。
- `BuildResponse(state *ApiState)` 无返回值，结果写在 `ApiState.Response`（类型为 `*ApiResponse[any]`）。
- 另有 `ApiState.MustHaveResponse()`。
- `ApiResponseBuilderFunc` 的类型为 `func(state *ApiState)`。

从 v0.9.0 起：

- 管线中不再单独执行 `BuildResponse`。内置 SlimAPI 的 `WriteResponse` 会调用 `state.Handler.BuildResponse(state, callResult, callError)`。
- 新签名为 `BuildResponse(state *ApiState, callResult any, callError error) any`。返回值即待序列化的结果；返回 `nil` 表示无输出。默认实现返回 `ApiResponse[any]` 值，不再写入 `ApiState`。
- `ApiState.Response` 与 `MustHaveResponse()` 已删除。
- `ApiResponseBuilderFunc` 的类型改为 `func(state *ApiState, callResult any, callError error) any`。

#### 影响范围

实现了自定义 `ApiResponseBuilder`、`ApiResponseBuilderFunc` 或 `ApiResponseWriter`，或者读写 `ApiState.Response` / `MustHaveResponse()` 的代码需要修改。只使用 `NewSlimApiHandler` / `NewBasicApiResponseBuilder` 默认实现的项目，通常不必改业务方法。

#### 迁移方式

1. 自定义 Builder 改为接收 `callResult` / `callError`，并返回组装结果，不要再写 `state.Response`：

```go
// 旧：
func (x) BuildResponse(state *webapi.ApiState) {
	state.Response = &webapi.ApiResponse[any]{Data: state.Data}
}

// 新：
func (x) BuildResponse(state *webapi.ApiState, callResult any, callError error) any {
	return webapi.ApiResponse[any]{Data: callResult}
}
```

2. 自定义 `WriteResponse` 需要组装业务结果时，自行调用 Builder，例如 `state.Handler.BuildResponse(state, state.Data, state.Error)`。
3. 删除对 `ApiState.Response` 和 `MustHaveResponse()` 的引用。
4. 若使用 `ApiResponseBuilderFunc`，按新签名补上 `callResult`、`callError` 和返回值。

### `NewBasicApiMethodRegister` 必须传入选项

#### 变更内容

`NewBasicApiMethodRegister` 从无参改为必须传入 `BasicApiMethodRegisterOp`。新增字段 `SupportStreamingResponse`：为 `true` 时允许方法返回 `StreamingResponse`（`EventStream` / `NdJson`）；零值 `false` 时拒绝这类返回值，与 v0.8.4 的注册规则一致。

`slimapi.NewSlimApiHandler` 已改为传入 `SupportStreamingResponse: true`。自行调用 `NewBasicApiMethodRegister` 组装 `ApiHandler` 时不会自动打开该开关。

#### 影响范围

所有直接调用 `webapi.NewBasicApiMethodRegister()` 的代码升级后无法编译。只通过 `NewSlimApiHandler` / `NewSlimAuthApiHandler` 创建 Handler 的项目不受影响。

#### 迁移方式

按是否需要流式返回值补上选项：

```go
// 旧：
webapi.NewBasicApiMethodRegister()

// 新，行为与 v0.8.4 一致（不允许流式返回值）：
webapi.NewBasicApiMethodRegister(webapi.BasicApiMethodRegisterOp{})

// 新，需要注册 EventStream / NdJson：
webapi.NewBasicApiMethodRegister(webapi.BasicApiMethodRegisterOp{
	SupportStreamingResponse: true,
})
```

### `SlimApiInvoker` 的 `Do` / `DoRaw` 不再解析流式响应

#### 变更内容

v0.8.4 的 `Do`、`DoRaw`、`MustDo`、`MustDoRaw` 会把整个 HTTP body 当作一段 JSON 解析。v0.9.0 若检测到响应 `Content-Type` 为 `text/event-stream` 或 `application/x-ndjson`，`Do` / `DoRaw` 会返回错误，`MustDo` / `MustDoRaw` 则会 panic，提示改用 `DoRawStream` / `MustDoStream`。

返回 HTTP 200 的普通 JSON 接口调用方式不变。非 200 响应的变化见下一节。

#### 影响范围

用 `SlimApiInvoker` 调用 SSE / NDJSON 接口，却仍走 `Do` / `DoRaw` 的代码会得到错误，其 `Must` 版本会 panic，而不再尝试把整段流解析成单个 `ApiResponse`。`SlimAuthInvoker` 内嵌了该调用器，同样受影响。只调用返回 HTTP 200 的非流式 JSON 接口的项目不必修改。

#### 迁移方式

流式接口改用新方法：

- `DoRawStream`：返回 `iter.Seq2[webapi.ApiResponse[TData], error]`，逐段给出信封。
- `MustDoStream`：仅在每段 `Code == 0` 时产出 `Data`；任一段 `Code != 0` 会 panic 为 `errx.BizError`。

详见 [`streaming.md`](streaming.md)。

### 调用器只接受 HTTP 200 响应

#### 变更内容

旧版 `SlimApiInvoker` 不检查 HTTP 状态码，只要 body 能解析为 `ApiResponse` 就继续处理。现在调用器统一检查状态码，仅接受 HTTP 200；其余状态码会产生 `unexpected HTTP status ...` 错误，不再按业务信封解析，即使 body 是合法的 JSON。

#### 影响范围

`Do` / `DoRaw` 返回错误，`MustDo` / `MustDoRaw` 会 panic。`SlimAuthInvoker` 也受影响。例如，以 HTTP 400 或 500 携带业务错误信封的接口，过去可通过 `DoRaw` 读取 `Code`，现在得到 HTTP 状态错误；HTTP 201 等其他成功状态也不被接受。

#### 迁移方式

使用内置调用器的服务端应按 SlimAPI 协议返回 HTTP 200，并通过 `ApiResponse.Code` 表示业务结果。若目标服务必须使用其他 HTTP 状态码，需要通过自定义 HTTP 调用逻辑读取并处理响应。

### 响应写出期间的 panic 改为日志

#### 变更内容

框架现在会捕获 `ResponseBody` 迭代、`http.ResponseWriter.Write` 及 `Flush` 期间发生的 panic，将错误追加到 `ApiState.LogMessage` 的 `WriteResponseError` 字段，并将日志级别至少提升到 `warn`，随后继续调用 `ApiLogger.Log`。

这比 v0.8.4 的处理范围更广：v0.8.4 只处理复制响应体和关闭 Reader 时返回的 error，不会统一捕获这些操作自身的 panic。捕获 panic 后不会重新写出完整响应，客户端可能已经收到部分内容。

#### 影响范围

依赖外层 Recoverer、自行 `recover` 或 stderr panic 堆栈发现响应写出异常的代码，需要改用日志观察该错误。该变化仅针对实际写出阶段，不代表所有管线步骤的 panic 都按相同方式处理。

#### 迁移方式

检查 `WriteResponseError` 和响应完整性；相关测试改为检查日志以及收尾流程，而不是断言 panic 向外传播。自定义迭代器应在内部使用 `defer` 释放资源。

### `webapitest.NewStateForTest` 的返回类型变化

#### 变更内容

第二个返回值从 `*httptest.ResponseRecorder` 改为 `*webapitest.RecorderEx`。`RecorderEx` 内嵌 `httptest.ResponseRecorder`，并增加 `OnWrite` 回调，用于观察流式写出。

#### 影响范围

测试代码若把该返回值声明为 `*httptest.ResponseRecorder`，或传给只接受该类型的函数，升级后无法编译。`Code`、`Body`、`Header`、`Result()` 等常用成员仍可通过内嵌直接使用。

#### 迁移方式

把变量类型改为 `*webapitest.RecorderEx`，或依赖短变量声明自行推断。若必须得到 `*httptest.ResponseRecorder`，使用 `&rec.ResponseRecorder`。需要观察每一次写出时，设置 `rec.OnWrite`。

---

## v0.8.4

复制响应体或关闭响应 Reader 返回错误时，框架不再主动 panic，改为记录日志并继续完成收尾流程。

### 变更内容

在 v0.8.3 中，`CreateHandlerFunc` 在 `io.Copy` 复制 HTTP body 返回错误，或响应 Reader 的 `Close()` 返回错误时，会主动 `panic`（写出失败的常见原因是客户端已断开连接）。该 panic 会中断后续流程，最终由 `NewEngine()` 内置的 Recoverer 中间件捕获，并输出到 stderr。后果是：

- `ApiLogger.Log` 不会被调用，结构化日志丢失。
- 请求处理的控制权落到最外层中间件，业务侧无法按正常管线收尾。

在 v0.8.4 中，上述操作返回的错误会记入 `ApiState`：

- 向 `ApiState.LogMessage` 追加一对键值：`WriteResponseError` 与具体错误。
- 若当前 `ApiState.LogLevel` 低于 `warn`，则提升到 `warn`。
- 随后仍会调用 `ApiLogger.Log`，错误会出现在应用日志里，而不是 stderr 上的 panic 堆栈。

这里处理的是返回的 error。Reader、Writer 或 `Close()` 自身抛出的 panic 在 v0.8.4 中仍会向外传播。v0.9.0 改用迭代器后，框架不再自动关闭 Reader，并新增了写出阶段的 panic 捕获，见对应小节。

### 影响范围

以下用法会受到影响：

- 依赖复制响应体或关闭 Reader 返回错误后触发的 panic，再靠 Recoverer / stderr 发现问题。
- 在外层自行 `recover`，并假定上述操作返回错误一定会触发 panic。
- 自定义 `ApiLogger` 时假定上述操作返回错误后 `Log` 不会执行。

只使用内置 SlimAPI / SlimAuth、且没有依赖上述 panic 行为的项目，通常只需改日志关注点，不必改业务代码。

### 迁移方式

1. 去掉对复制响应体或关闭 Reader 返回错误后触发 panic 的依赖。
2. 改为观察应用日志。上述错误会产生 `WriteResponseError` 字段，日志级别至少为 `warn`。
3. 若实现了自定义 `ApiLogger`，请处理 `ApiState.LogMessage` 中的 `WriteResponseError`，不要在写出失败后跳过 `Log`。
4. 若测试曾断言上述操作返回错误会触发 panic，改为断言日志中包含 `WriteResponseError`，且请求处理正常结束。

---

## v0.8.3

### 自动关闭响应 Reader

#### 变更内容

框架在复制 `ApiState.ResponseBody` 后，若它实现了 `io.Closer`，会自动调用 `Close()`；此前不会自动关闭。v0.8.3 中，`Close()` 返回错误还会触发 panic。该自动关闭行为适用于 v0.8.3 / v0.8.4，v0.9.0 改用迭代器后不再适用。

#### 影响范围

将文件、其他 `io.ReadCloser` 或共享 Reader 交给框架后，若还打算在请求结束后继续读取、复用，可能遇到资源已关闭的问题。原先自行关闭资源的代码也需要检查重复关闭。

#### 迁移方式

明确响应 Reader 的资源归属，避免将仍需复用的共享资源直接交给框架。若必须由业务管理生命周期，可使用仅实现 `io.Reader` 的包装类型传递，并自行保证正确释放。

### 内置响应 Writer 保留已有响应体

#### 变更内容

内置 SlimAPI 响应 Writer 现在发现 `ApiState.ResponseBody != nil` 时会直接保留该响应体，不再重新序列化 `ApiState.Response` 覆盖它；`ResponseContentType` 为空时会补为 `text/plain`。`ApiResponseWriter` 的接口约定也改为保留已有的 `Response*` 字段。SlimAuth 复用该 Writer，同样受影响。

#### 影响范围

此前先填充临时响应体、再依赖内置 Writer 覆盖为业务 JSON 的代码，现在会输出预填充的内容。仅使用默认管线且没有提前设置响应体的项目不受影响。

#### 迁移方式

希望框架生成业务 JSON 时，不要提前设置 `ResponseBody`，或在进入 Writer 前将临时值清空；需要自定义响应时，同时设置响应体和正确的 `ResponseContentType`。自定义 Writer 也应遵循保留已有响应字段的接口约定。
