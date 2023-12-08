package slimapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"reflect"
	"strconv"
	"strings"

	"github.com/cmstar/go-conv"
	"github.com/cmstar/go-webapi"
)

// FilePart 用于封装一个 *multipart.FileHeader ，用于 [Conv] 对象进行类型转换，
// 以支持 multipart/form-data 方式的参数及文件上传。
type FilePart struct {
	*multipart.FileHeader        // 原始的 FileHeader 。
	content               []byte // 数据部分被读取后，存储在这里。
	jsonValue             any    // 对于 application/json 类型的数据， 读取并 json.Unmarshal 然后存储在这里。
}

var _ json.Marshaler = (*FilePart)(nil)

// NewFilePart 创建一个 FilePart 。
//
// 如果 part 带有 HTTP 头 Content-Type:application/json ，则 JSON 内容会被读取并校验其格式。
// 若读取或校验失败，返回对应的错误。
func NewFilePart(fh *multipart.FileHeader) (*FilePart, error) {
	f := &FilePart{
		FileHeader: fh,
	}

	if f.IsJson() {
		content, err := f.ReadAll()
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(content, &f.jsonValue)
		if err != nil {
			err = fmt.Errorf("unmarshal JSON part '%s': %w", f.Filename, err)
			return nil, err
		}
	}

	return f, nil
}

// ContentType 返货当前 part 的 Content-Type 。
// 若没有此头部，返回空字符串；若此头部重复多次，仅返回第一个值。
func (x *FilePart) ContentType() string {
	types := x.Header[webapi.HttpHeaderContentType]
	if len(types) > 0 {
		return types[0]
	}
	return ""
}

// IsJson 判断当前 part 是否具有 HTTP 头 Content-Type:application/json 。
func (x *FilePart) IsJson() bool {
	return x.ContentType() == webapi.ContentTypeJson
}

// ReadAll 读取当前 part 的全部数据。
// 此方法可被重复调用。
func (x *FilePart) ReadAll() (res []byte, err error) {
	if x.content != nil {
		return x.content, nil
	}

	f, err := x.Open()
	if err != nil {
		err = fmt.Errorf("open file part '%s': %w", x.Filename, err)
		return
	}
	defer f.Close()

	res, err = io.ReadAll(f)
	if err != nil {
		err = fmt.Errorf("read file part '%s': %w", x.Filename, err)
	}

	x.content = res
	return
}

// JsonValue 返回当前 part 的 JSON 数据的 json.Unmarshal 反序列化结果。
// 若当前 part 不具有 Content-Type:application/json ，则 panic 。
func (x *FilePart) JsonValue() any {
	if !x.IsJson() {
		panic(fmt.Sprintf("require Content-Type: application/json, got %s", x.ContentType()))
	}
	return x.jsonValue
}

// MarshalJSON 实现 json.Marshaler 。它以 JSON 形式返回对于当前 FilePart 的描述信息。
func (x *FilePart) MarshalJSON() ([]byte, error) {
	// json 包没有开放字符串转义的方法。只能重复调用 Marshal ，有点费事。
	escapeString := func(v string) []byte {
		res, _ := json.Marshal(v)
		return res
	}

	// 对于非 JSON 数据： {"$FileName":"name","ContentType":"type","Size":123} ；
	// 对于 JSON 数据： {"$FileName":"name","ContentType":"type","Size":123,"Data":{jsonValue 的序列化结果}} 。
	//
	// 这里重新序列化 jsonValue ，有两个作用：
	// 1. 移除原文 JSON 里的空白。
	// 2. 使输出的 JSON key 有序。
	//
	// $FileName 以 $ 开头，以避免和一般的参数混淆。
	buf := new(bytes.Buffer)
	buf.WriteString(`{"$FileName":`)
	buf.Write(escapeString(x.Filename))
	buf.WriteString(`,"ContentType":`)
	buf.Write(escapeString(x.ContentType()))
	buf.WriteString(`,"Size":`)
	buf.Write([]byte(strconv.Itoa(int(x.Size))))

	if x.IsJson() {
		v, err := json.Marshal(x.jsonValue)
		if err != nil {
			panic(err)
		}
		buf.WriteString(`,"Data":`)
		buf.Write(v)
	}

	buf.WriteString(`}`)
	return buf.Bytes(), nil
}

var _convConf = conv.Config{
	FieldMatcherCreator: &conv.SimpleMatcherCreator{
		Conf: conv.SimpleMatcherConfig{
			CaseInsensitive: true,
		},
	},
	StringToTime:   ParseTime,
	StringSplitter: func(v string) []string { return strings.Split(v, "~") },
}

// 提供 Conv 变量内部使用。
// Go 不支持变量初始化时引用自己，出现 _converters 依赖 Conv ， Conv 又依赖 _converters 则不能编译。
// 为解决此问题，改为单向引用： Conv -> _converters -> _internalConv 。
var _internalConv = conv.Conv{Conf: _convConf}

// Conv 是用于 SlimAPI 的 [conv.Conv] 实例，它支持：
//   - 使用大小写不敏感（case-insensitive）的方式处理字段。
//   - 支持 SlimAPI 规定的时间格式 yyyyMMdd HH:mm:ss 。
//   - 支持字符串到数组的转换，使用 ~ 分割，如将 "1~2~3" 转为 [1, 2, 3] 。
//
// 特别的，用于支持 multipart/form-data 的文件上传，如果输入 [*FilePart] ：
//   - 目标值类型是 [*multipart.FileHeader] 时，原样返回输入值，不做转换。
//   - 目标值时 []byte 时，将其数据读取出来。
//   - 目标值时其他类型时，若此分部的 Content-Type 为 application/json ，则将其内容作为 JSON 读取，并将此 JSON 反序列化到目标值。
var Conv = func() conv.Conv {
	_convConf.CustomConverters = func() []conv.ConvertFunc {
		var (
			typFileHeader = reflect.TypeOf(&multipart.FileHeader{})
			typByteSlice  = reflect.TypeOf([]byte{})
		)

		// *FilePart -> *multipart.FileHeader 原样返回被封装值。
		fileHeaderToFileHeader := func(value interface{}, typ reflect.Type) (result interface{}, err error) {
			if typ != typFileHeader {
				return
			}

			f, ok := value.(*FilePart)
			if !ok {
				err = fmt.Errorf("require *slimapi.FilePart, got %T", value)
				return
			}

			return f.FileHeader, nil
		}

		// *FilePart -> []byte
		fileHeaderToBytes := func(value interface{}, typ reflect.Type) (result interface{}, err error) {
			if typ != typByteSlice {
				return
			}

			f, ok := value.(*FilePart)
			if !ok {
				return
			}

			return f.ReadAll()
		}

		// *FilePart as JSON -> any
		jsonFileHeaderToAny := func(value interface{}, typ reflect.Type) (result interface{}, err error) {
			f, ok := value.(*FilePart)
			if !ok || !f.IsJson() {
				return
			}

			jsonValue := f.JsonValue()
			result, err = _internalConv.ConvertType(jsonValue, typ)
			return
		}

		return []conv.ConvertFunc{
			fileHeaderToFileHeader,
			fileHeaderToBytes,
			jsonFileHeaderToAny,
		}
	}()

	return conv.Conv{Conf: _convConf}
}()
