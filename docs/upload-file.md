# 接收文件

本文描述如何基于 slimapi 框架接收 `multipart/form-data` 类型的请求。

## 方法的输入参数

可以使用下面的类型接收 `multipart/form-data` 格式的请求中的文件：

| 目标类型                | 行为                             |
| ----------------------- | -------------------------------- |
| `*multipart.FileHeader` | Golang 标准库定义的文件类型。    |
| `*slimapi.FilePart`     | 内嵌 `*multipart.FileHeader` ，可视同 FileHeader 。 |
| `[]byte`                | 直接读取文件的内容。             |

`multipart/form-data` 类型的请求种的每个部分会根据 `name` 值（大小写不敏感）匹配到参数表的字段上，并自动进行类型转换。

可通过两种类型的字段接收对应的 part ：
- `*multipart.FileHeader`/`*slimapi.FilePart` 会拿到 Go 标准库解析后的原始结果。
- `[]byte` 会直接读取 part 的数据部分，在数据不大，又不需要读取 `filename` 等信息时使用起来较为方便。

和其他参数一样，如果请求不包含对应名称的部分，则参数值为类型的默认值，即 nil 。

示例代码：
```go
import "mime/multipart"

type ApiProvider struct{}

func (ApiProvider) UploadMultipleIcon(request struct {
	Num     int
	Str     string
	Icon1   *multipart.FileHeader
	Icon2   []byte
}) {
	// impl
}
```

对应的请求可以是：
```
POST http://temp.org/UploadMultipleIcon
Content-Type: multipart/form-data; boundary=TheBoundary

--TheBoundary
Content-Disposition: form-data; name="Num"

42
--TheBoundary
Content-Disposition: form-data; name="Str"

a string value
--TheBoundary
Content-Disposition: form-data; name="icon1"; filename="1.png"
Content-Type: image/png

{BinaryData}
--TheBoundary
Content-Disposition: form-data; name="icon2"; filename="2.png"
Content-Type: image/png

{BinaryData}
--TheBoundary--
```

### 通过 JSON 在传文件的同时传递复杂参数

如果需要在接收文件的同时，接收复杂结构的参数，可以使用具有 `Content-Type: application/json` 的分部。

例如下面的方法，在接收一个复杂类型的参数 `Complex` 的同时，接收一个文件 `Icon`：
```go
import "mime/multipart"

func (ApiProvider) UploadComplex(request struct {
	Simple  int
	Complex struct {
		IntValue    int
		StringValue string
		Array       []int
	}
	Icon *multipart.FileHeader
}) {
	// impl
}
```

对应的请求可以是：
```
POST http://temp.org/UploadComplex
Content-Type: multipart/form-data; boundary=TheBoundary

--TheBoundary
Content-Disposition: form-data; name="Simple"

42
--TheBoundary
Content-Disposition: form-data; name="Complex"; filename="blob"
Content-Type: application/json

{
    "IntValue": 123,
    "StringValue": "vv",
    "Array": [1, 2, 3]
}
--TheBoundary
Content-Disposition: form-data; name="Icon"; filename="icon.png"
Content-Type: image/png

{BinaryFileData}
--TheBoundary--
```

注意：这里的 `Complex` 部分需要给定一个有效但无意义的 `filename` ，否则 Go 的标准库会直接默认此部分的 `Content-Type` 为 `text/plain` 。

上面的请求，等同于在上传了文件的同时，额外传递了下面的数据：

```json
{
    "Simple": 42,
    "Complex": {
        "IntValue": 123,
        "StringValue": "vv",
        "Array": [1, 2, 3]
    }
}
```
