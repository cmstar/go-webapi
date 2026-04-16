# SlimAuth

SlimAuth 协议是 [SlimAPI](slim-api.md) 附带签名校验的扩展。它在 SlimAPI 的基础上，为每个请求添加基于 HMAC-SHA256 的签名校验。

`slimauth` 包通过替换 SlimAPI 的部分[管线组件](architecture.md#定制与扩展以-slimauth-为例)来实现这一扩展。

> 本文档的协议部分也可参考 [GoDoc](https://pkg.go.dev/github.com/cmstar/go-webapi/slimauth#pkg-overview)。

## 限制

与 SlimAPI 相比，SlimAuth **不支持 `multipart/form-data`** 类型的请求。

## 使用 Authorization 头

每个 API 调用者会被分配一组配对的 key-secret。key 用于标识调用者身份，secret 用于生成签名。

签名信息通过 HTTP `Authorization` 头传递：

```
Authorization: SLIM-AUTH Key={key}, Sign={sign}, Timestamp={timestamp}, Version=1
```

| 参数      | 说明                                                                        |
| --------- | --------------------------------------------------------------------------- |
| Scheme    | 固定为 `SLIM-AUTH`（可通过 `SlimAuthApiHandlerOption.AuthScheme` 自定义）。 |
| Key       | 请求方的标识。                                                              |
| Sign      | 基于请求内容和 secret 生成的签名。                                          |
| Timestamp | 生成签名时的 UNIX 时间戳（秒）。                                            |
| Version   | 签名算法版本，当前固定为 1。可省略，默认为 1。                              |

参数间由逗号隔开，顺序不做要求，参数名称前的空白字符会被忽略。

### 通过 URL 参数传递

当不方便定制请求头时（例如 JSONP 请求），也可以将 Authorization 头的值放在 URL 的 `~auth` 参数上（需 URL 编码）。`~auth` 参数**不参与签名计算**。

如果同时提供了 `~auth` 参数和 `Authorization` 头，只读取请求头。

## 签名算法

字符集统一使用 UTF-8。签名使用 HMAC-SHA256 算法，通过 secret 对待签名串进行计算。

### 待签名串格式

```
TIMESTAMP
METHOD
PATH
QUERY_VALUES
BODY_VALUES（可选）
END
```

每部分间用换行符（`\n`）分割：

1. **TIMESTAMP** —— UNIX 时间戳，须与 Authorization 头中的 Timestamp 一致。
2. **METHOD** —— HTTP 方法，如 `GET`、`POST`。
3. **PATH** —— 请求路径。没有路径时使用 `/`。例如 `http://temp.org/the/path/` 的路径为 `/the/path/`。
4. **QUERY_VALUES** —— URL 参数值的拼接结果：
   - 按参数名称的 UTF-8 字节序升序排列（使用稳定排序）。
   - 将排序后的参数值紧密拼接（无分隔符）。
   - 若参数没有值（如 `?a=` 或 `?a`），用参数名称代替。
   - 没有 query string 时为空字符串。
   - `~auth` 参数不参与签名。
5. **BODY_VALUES**（仅 POST/PUT/PATCH 请求）——
   - `application/x-www-form-urlencoded`：处理方式同 QUERY_VALUES。
   - `application/json`：JSON 原文，不做任何修改。
   - GET 请求时省略此部分（包含换行符）。
6. **END** —— 固定字符串 `END`，末尾没有换行。

> 注意：UTF-8 字节序下，英文大写字母在小写字母前面（如 `X` 排在 `a` 前面）。

### 示例

以下示例中，Timestamp 固定为 `1662439087`，Key 为 `my_key`，Secret 为 `my_secret`。

#### 示例 1：带参数的 POST 请求

请求：

```
POST http://temp.org/my/path?a&c=3&b=2&z=4&X=%E4%B8%AD%E6%96%87&a=1&b=
Content-Type: application/x-www-form-urlencoded

p1=11&p3=33&p2=22
```

QUERY 部分计算：
- 参数按 UTF-8 字节序排列：`[X, a, a, b, b, c, z]`。
- 对应的值：`[中文, (空→用名称a代替), 1, 2, (空→用名称b代替), 3, 4]`。
- 拼接结果：`中文a12b34`。

BODY 部分（表单格式，同 QUERY 规则）：`112233`。

待签名串：

```
1662439087
POST
/my/path
中文a12b34
112233
END
```

签名结果：`b3baa63839877585cc05495810fb10267317df2fceda2eddcb92a740f78d1ba5`

#### 示例 2：空白 GET 请求

请求：`GET http://temp.org`

待签名串：

```
1662439087
GET
/

END
```

GET 请求没有 BODY 部分；QUERY 为空字符串。

签名结果：`980b8715cefc0b98ae2b0788ce849308757554fbe685a05a43e6bc31fb0d0a4c`

#### 示例 3：JSON POST 请求

请求：

```
POST http://temp.org/p/?x=1&y=2
Content-Type: application/json

{"key":"value"}
```

待签名串：

```
1662439087
POST
/p/
12
{"key":"value"}
END
```

JSON body 原样拼接，不做修改。

签名结果：`ce0906df79291d516bb443adbc6099b39f36c006696150202e4e41ffe7dab211`

---

## 服务端使用

### 创建 Handler

```go
handler := slimauth.NewSlimAuthApiHandler(slimauth.SlimAuthApiHandlerOption{
    Name: "my-auth-api",
    SecretFinder: func(accessKey string) string {
        // 根据 key 查找对应的 secret。
        // 返回空字符串表示 key 不存在。
        return findSecret(accessKey)
    },
})

handler.RegisterMethods(Methods{})

e := webapi.NewEngine()
e.Handle("/api/{~method}", handler, logFinder)
```

#### SlimAuthApiHandlerOption

| 字段           | 说明                                                                       |
| -------------- | -------------------------------------------------------------------------- |
| `Name`         | Handler 名称，用于日志分区。                                               |
| `AuthScheme`   | Authorization 头的 scheme 部分。为空时使用默认值 `SLIM-AUTH`。             |
| `SecretFinder` | **必填**。根据 accessKey 查找 secret 的函数。返回空字符串表示 key 不存在。 |
| `TimeChecker`  | 时间戳校验函数。为 `nil` 时使用 `DefaultTimeChecker`。                     |

#### SecretFinderFunc

```go
type SecretFinderFunc func(accessKey string) string
```

根据请求方提供的 `accessKey` 查找对应的 `secret`。返回空字符串表示该 key 未绑定。若查找过程出错，可直接 panic。

#### TimeCheckerFunc

用于校验签名中的时间戳是否在允许范围内：

| 预定义实现                   | 说明                                                  |
| ---------------------------- | ----------------------------------------------------- |
| `DefaultTimeChecker`         | 要求时间戳与服务器时间误差在 **5 分钟**（300 秒）内。 |
| `MaxDeviationTimeChecker(n)` | 自定义最大允许误差（秒）。                            |
| `NoTimeChecker`              | 不校验时间戳。                                        |

### 在方法中获取认证信息

API 方法可以直接声明 `Authorization` 或 `*Authorization` 类型的参数，框架会自动注入：

```go
func (Methods) WhoAmI(auth slimauth.Authorization) string {
    return auth.Key
}
```

`Authorization` 结构体：

```go
type Authorization struct {
    AuthScheme string // Authorization 头的 Scheme 部分。
    Key        string // 请求方的标识。
    Sign       string // 签名。
    Timestamp  int64  // UNIX 时间戳（秒）。
    Version    int    // 算法版本。
}
```

### 通过 ApiState 获取

也可以在方法内通过 `ApiState` 获取：

```go
func (Methods) WhoAmI(state *webapi.ApiState) string {
    auth := slimauth.MustGetBufferedAuthorization(state)
    return auth.Key
}
```

提供两个获取函数：
- `GetBufferedAuthorization(state)` —— 返回 `(Authorization, bool)`，获取失败时返回 `ok=false` 。
- `MustGetBufferedAuthorization(state)` —— `GetBufferedAuthorization` 的 panic 版本，获取失败时 panic。

---

## 客户端调用

### SlimAuthInvoker

`SlimAuthInvoker` 在 `SlimApiInvoker` 基础上自动完成签名计算：

```go
invoker := slimauth.NewSlimAuthInvoker[MyParam, MyResult](slimauth.SlimAuthInvokerOp{
    Uri:    "http://localhost:15001/api/Plus",
    Key:    "my_key",
    Secret: "my_secret",
})

result, err := invoker.Do(MyParam{A: 1, B: 2})
```

`SlimAuthInvokerOp` 选项：

| 字段         | 说明                                             |
| ------------ | ------------------------------------------------ |
| `Uri`        | 目标 URL。                                       |
| `Key`        | SlimAuth 的 accessKey。                          |
| `Secret`     | SlimAuth 的 secret。                             |
| `AuthScheme` | Authorization 的 scheme 部分，为空时使用默认值。 |

`SlimAuthInvoker` 内嵌了 `*SlimApiInvoker`，因此继承了 `Do`、`DoRaw`、`MustDo`、`MustDoRaw` 等全部方法。

### 手动签名

如需手动签名（不使用 Invoker），可以使用底层函数：

```go
// 计算签名并直接设置 http.Request 的 Authorization 头。
signResult := slimauth.AppendSign(request, "my_key", "my_secret", "", time.Now().Unix())
if signResult.Cause != nil {
    // 签名失败。
}

// 或者仅计算签名，不修改请求。
signResult = slimauth.Sign(request, true, "my_secret", time.Now().Unix())
fmt.Println(signResult.Sign)
```

`HmacSha256(secret, data)` 可用于直接计算 HMAC-SHA256，返回小写 HEX 字符串。
