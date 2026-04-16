# SlimAPI

SlimAPI 是一个基于 HTTP 的 WebAPI 通信协议。
- 自适应多种不同格式的输入。
- 固定使用 JSON 输出。

`slimapi` 包基于 `webapi` 的[管线模型](architecture.md)，提供了 SlimAPI 协议的完整实现。

> 本文档的协议部分也可参考 [GoDoc](https://pkg.go.dev/github.com/cmstar/go-webapi/slimapi#pkg-overview)。

## 请求格式

SlimAPI 支持 GET 和 POST 两种 HTTP 方法，通过 `Content-Type` 头区分请求数据的格式：

| Content-Type                        | 格式                           |
| ----------------------------------- | ------------------------------ |
| （GET 请求）                        | 参数放在 URL query string 上。 |
| `application/x-www-form-urlencoded` | 表单格式。                     |
| `multipart/form-data`               | 表单格式（支持文件上传）。     |
| `application/json`                  | JSON 格式。                    |

在 [Getting Started](getting-started.md) 中已演示了 GET / POST+JSON / POST+表单 格式的用法。

而 `multipart/form-data` 格式较为灵活，详见 [接收文件](upload-file.md) 。

---

## 方法名解析

SlimAPI 支持三种方式指定目标 API 方法名称。

> 所有方式下，方法名称都是**大小写不敏感**的。

### 形式 1：URL 路由

最常见的方式，将方法名编排到 URL 路径中：

```
http://domain/api/Plus
```

对应注册路由时使用 `{~method}` 占位符：

```go
e.Handle("/api/{~method}", handler, logFinder)
```

### 形式 2：命名元参数

```
http://domain/api?~method=METHOD&~format=FORMAT&~callback=CALLBACK
```

以 `~` 开头的参数是 API 框架的**元参数**：

| 参数        | 必填 | 说明                                                                 |
| ----------- | ---- | -------------------------------------------------------------------- |
| `~method`   | 是   | 目标方法名称。                                                       |
| `~format`   | 否   | 请求格式，可选值：`get`、`post`、`json`。优先级高于 `Content-Type`。 |
| `~callback` | 否   | JSONP 回调函数名称。指定后返回 JSONP 格式。                          |

`~format` 的可选值：
- `get` —— 默认值，使用 GET 方式处理参数。
- `post` —— 等同于 `Content-Type: application/x-www-form-urlencoded`。
- `json` —— 等同于 `Content-Type: application/json`。
- `plain` —— 指定响应的 Content-Type 为 `text/plain`（可与上述值组合，如 `~format=json,plain`）。

### 形式 3：紧凑格式

将元参数值直接追加在 URL 后面，省略参数名：

```
http://domain/api?METHOD.FORMAT(CALLBACK)
```

其中 `.FORMAT` 和 `(CALLBACK)` 是可选的。例如：
- `http://domain/api?Plus` —— 仅指定方法名。
- `http://domain/api?Plus.json` —— 指定方法名和格式。
- `http://domain/api?Plus(myCallback)` —— 指定方法名和 JSONP 回调。

---

## 响应格式

SlimAPI 的 HTTP 状态码总是 200，具体结果通过 JSON 信封中的 `Code` 字段判定：

```json
{
    "Code": 0,
    "Message": "",
    "Data": <dynamic>
}
```

| 字段      | 说明                               |
| --------- | ---------------------------------- |
| `Code`    | 0 表示成功；非 0 表示错误。        |
| `Message` | 错误描述，`Code` 为 0 时通常为空。 |
| `Data`    | 返回的数据，类型取决于具体 API。   |

### 错误码约定

| 范围       | 含义                                                           |
| ---------- | -------------------------------------------------------------- |
| 0          | 成功。                                                         |
| 400        | 请求参数或报文错误。                                           |
| 500        | 服务端内部错误。                                               |
| 其他 1-999 | 与 HTTP 状态码重合区域，通常不使用。                           |
| 1000-9999  | 用于表示通信协议约定的错误，比如权限验证失败、签名校验错误等。 |
| 10000 之后 | 表示具体的业务错误。                                           |

### JSONP

指定 `~callback` 参数后，响应为 JSONP 格式，Content-Type 变为 `text/javascript`：

```javascript
myCallback({"Code":0,"Message":"","Data":3})
```

### 纯文本

通过 `~format=plain`（或 `~format=json,plain`），可将响应的 Content-Type 设为 `text/plain`，body 内容不变。

---

## 输出值与错误处理

API 方法支持 0-2 个返回值：

| 返回值数 | 规则                                 |
| -------- | ------------------------------------ |
| 0 个     | 无返回值，`Data` 为 `null`。         |
| 1 个     | 可以是任意受支持类型，或 `error`。   |
| 2 个     | 第一个是数据，第二个必须是 `error`。 |

错误处理规则：

| 情况                                     | 输出                                                               |
| ---------------------------------------- | ------------------------------------------------------------------ |
| 没有 `error` 返回值，或 `error` 为 `nil` | `Code=0`，正常返回。                                               |
| 返回 `errx.BizError`                     | `Code=BizError.Code()`，`Message=BizError.Message()`。             |
| 返回其他 `error`                         | `Code=500`，`Message="internal error"`（具体错误仅记录在日志中）。 |
| 方法 panic                               | `Code=500`，`Message="internal error"`。                           |

> `BizError` 的详细说明参考 [go-errx 库](https://github.com/cmstar/go-errx#bizerror)。

---

## 输出值

### struct 参数

API 方法的参数通常是一个 struct，struct 的每个导出字段对应一个请求参数。字段名称**大小写不敏感**。

```go
func (Methods) Plus(req struct {
    A int
    B int
}) int {
    return req.A + req.B
}
```

### `*webapi.ApiState` 参数

方法可以声明 `*webapi.ApiState` 类型的参数，以访问当前请求的完整上下文：

```go
func (Methods) Headers(state *webapi.ApiState) map[string][]string {
    return state.RawRequest.Header
}
```

`*ApiState` 可与 struct 参数同时使用。但注意：**方法参数表中同一种类型只能出现一次**。

### 接收文件

详见 [接收文件](upload-file.md) 。

### 参数的合并

对于 GET 请求，参数来自 URL query string。

对于 POST 请求，参数来源按格式有所不同，但遵循统一的合并规则：
- URL 上的参数（query）总是会被读取。
- 表单（含 multipart/form-data 格式）参数与 query 合并，同名参数的值用逗号拼接。
- JSON 参数与 query 合并，同名参数以 JSON 的值为准。

例子1：
```
POST /api/MyMethod?a=v1&b=2
Content-Type: application/x-www-form-urlencoded

a=v2&c=3
```

结果等同于于 `{"a":"v1,v2","b":2,"c":3}`。

例子2：
```
POST /api/MyMethod?a=v1&b=2
Content-Type: application/json

{"a":"v2","c":3}
```

结果等同于于 `{"a":"v2","b":2,"c":3}`。

### 类型转换

`slimapi.Conv` 是 SlimAPI 使用的类型转换器（基于 [go-conv](https://github.com/cmstar/go-conv) 库），具有以下特性：

- **大小写不敏感**——字段匹配忽略大小写。
- **数组分隔符**——在 GET/表单格式下，字符串使用 `~` 分隔转为数组。例如 `"1~2~3"` 转为 `[1, 2, 3]`。
- **类型自动转换**——当目标类型与输入值的类型不同时，会尝试进行自动转换。例如 GET 参数 `a=123` 可使用 `A int` 接收，也可以使用 `A string` 接收。
- **时间格式**——支持 SlimAPI 规定的 `yyyy-MM-dd HH:mm:ss` 格式（UTC），也兼容 RFC3339。
- **FilePart 支持**——详见上一节。

---

## 客户端调用：SlimApiInvoker

`slimapi.SlimApiInvoker[TParam, TData]` 是一个泛型 HTTP 客户端，用于调用 SlimAPI 接口。

```go
invoker := slimapi.NewSlimApiInvoker[MyParam, MyResult]("http://localhost:15001/api/Plus")

// Do：Code=0 时返回 Data，否则返回 errx.BizError。
result, err := invoker.Do(MyParam{A: 1, B: 2})

// DoRaw：返回原始的 ApiResponse，不判断 Code。
resp, err := invoker.DoRaw(MyParam{A: 1, B: 2})
```

请求总是以 POST + `Content-Type: application/json` 方式发送。

如需在请求前做额外处理（如添加自定义 Header），可设置 `RequestSetup`：

```go
invoker.RequestSetup = func(r *http.Request) error {
    r.Header.Set("X-Custom", "value")
    return nil
}
```

每个方法都有对应的 panic 版本（`MustDo`、`MustDoRaw`）。
