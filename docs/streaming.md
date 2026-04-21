# 流式输出

本文描述如何基于 slimapi 返回 **Server-Sent Events（SSE）** 与 **Newline Delimited JSON（NDJSON）** 形式的流式 HTTP 响应。
二者均由 `webapi` 包提供类型，由 `slimapi` 的响应写入逻辑按 SlimAPI 信封规则序列化每一段输出。

## API 方法注册

API 方法的**有且仅有一个返回值**且类型为下列之一时，即表示该方法使用流式响应：

| 返回类型                   | 说明                                                                                                             |
| -------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `webapi.EventStream[DATA]` | HTTP `Content-Type` 为 `text/event-stream` ，按 SSE 规范写出多段 `data:` ，并在流末尾发送固定格式的 `END` 事件。 |
| `webapi.NdJson[DATA]`      | HTTP `Content-Type` 为 `application/x-ndjson` ，每行一条 JSON ，行与行之间用换行分隔。                           |

若自行组装 `ApiHandler` 且使用 `webapi.NewBasicApiMethodRegister`，必须在选项中开启 `SupportStreamingResponse: true` ，否则注册阶段会拒绝上述返回类型。
使用 `slimapi.NewSlimApiHandler` 时，该开关已默认开启，无需额外配置。

`EventStream` 与 `NdJson` 基于 Go 1.23 版本引入了迭代器语法实现：

```go
type EventStream[DATA any] func(yield func(data DATA, err error) bool)
type NdJson[DATA any] func(yield func(data DATA, err error) bool)
```

示例（仅演示，省略业务逻辑）：

```go
type Item struct {
	Step int    `json:"step"`
	Text string `json:"text"`
}

func (Methods) EventStreamDemo() webapi.EventStream[Item] {
	return func(yield func(data Item, err error) bool) {
		for i := 1; i <= 3; i++ {
			if !yield(Item{Step: i, Text: "..."}, nil) {
				return
			}
		}
		// 若某次 yield 携带非 nil 的 error，下一段 JSON 会反映为错误信封。
	}
}

func (Methods) NdJsonDemo() webapi.NdJson[Item] {
	return func(yield func(data Item, err error) bool) {
		for i := 1; i <= 3; i++ {
			if !yield(Item{Step: i, Text: "..."}, nil) {
				return
			}
		}
	}
}
```

> 客户端的使用见下文《通过 SlimApiInvoker 访问流式 API》节。

---

## 数据格式

流式输出**不是**整段响应一个大 JSON，而是**多次**写出与常规接口相同的信封结构（由 `ApiResponseWriter` 与 `BuildResponse` 生成），每一段对应一次 `yield` 的结果（以及可选的 `error` 信息）。

非流式接口的典型形态为：

```json
{
    "Code": 0,
    "Message": "",
    "Data": <dynamic>
}
```

流式场景下，每一段仍遵循上述字段约定；出现错误时，返回一段的 `Code` 可为非 0 的结果，同时结束当前输出流。错误处理的具体规则与 [SlimAPI 错误处理](slim-api.md#输出值与错误处理)一致。

### Server-Sent Events

`webapi.EventStream[DATA]` 在 HTTP 层表现为 `Content-Type: text/event-stream`  格式的数据。

每一段业务数据在 SSE 中占一行 `data:` ，其后为一行 SlimAPI 标准格式的 JSON，并以两个换行结束该事件，例如：
```
data: {"Code":0,"Message":"","Data":{"Step":1}}

data: {"Code":0,"Message":"","Data":{"Step":2}}

```

流正常结束时，框架会再写入固定的结束事件，事件名固定为 `END` ，`data` 中的 `Code` 固定为 `1000` （定义在常量 `webapi.EventStreamEndCode` ）：

```
event: END
data: {"Code":1000,"Message":"","Data":null}

```

浏览器侧可使用 `EventSource` 订阅默认消息与名为 `END` 的自定义事件；结束事件到达后应关闭连接。

### ND-JSON

`webapi.NdJson[DATA]` 在 HTTP 层表现为 `Content-Type: application/x-ndjson` 格式的数据。

每一段信封为一行 JSON ，行尾换行；例如：
```
{"Code":0,"Message":"","Data":{"Step":1}}
{"Code":0,"Message":"","Data":{"Step":2}}

```

与 SSE 不同，NDJSON 没有由协议规定的“最后一行结束标记”；HTTP 响应体结束即表示流结束。读取端应按行缓冲解析 JSON ，并处理最后一行可能未以换行结束的情况。

---

## 通过 SlimApiInvoker 访问流式 API

`slimapi.SlimApiInvoker` 提供了用于调用流式API的方法。

- **`DoRawStream`**：返回 `iter.Seq2[webapi.ApiResponse[TData], error]`。每一项的第一分量是一段完整信封；第二分量为读流或 JSON 解析错误。非流式 JSON 响应时，序列中通常只有一项。
- **`MustDoStream`**：在 `DoRawStream` 之上封装，返回 `iter.Seq[TData]`，仅在每段 `Code == 0` 时产出 `Data`；任一段 `Code != 0` 会 **panic** 为 `errx.BizError`。

下面演示迭代器的典型用法（泛型参数需换成你的请求类型与每段 `Data` 类型）。

`DoRawStream`：需要自行判断 `error` 与每一段 `Code`。

```go
invoker := slimapi.NewSlimApiInvoker[MyReq, MyChunk]("http://localhost:15000/api/MyStream")

for resp, err := range invoker.DoRawStream(MyReq{ /* 字段 */ }) {
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return fmt.Errorf("stream chunk: code=%d msg=%s", resp.Code, resp.Message)
	}

	_ = resp.Data
}
```

`MustDoStream`：只关心成功的 `Data`，错误段通过 panic 暴露（与 `MustDo` 风格一致）。

```go
invoker := slimapi.NewSlimApiInvoker[MyReq, MyChunk]("http://localhost:15000/api/MyStream")

for chunk := range invoker.MustDoStream(MyReq{ /* 字段 */ }) {
	_ = chunk
}
```
