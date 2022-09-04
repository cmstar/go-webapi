package slimauth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/cmstar/go-webapi"
)

/* 当前文件提供签名算法的实现。 */

// Authorization 记录 SlimAuth 协议规定的 HTTP Authorization 头的内容。
type Authorization struct {
	Key       string // 请求方的标识。
	Sign      string // 签名。
	Timestamp int    // 生成签名时的 UNIX 时间戳，单位是秒。
	Version   int    // 算法版本。在 Authorization 头未给出时，默认为 [DEFAULT_AUTHORIZATION_VERSION] 。
}

// BuildAuthorizationHeader 返回用于 HTTP 的 Authorization 头的值。
// 若 [Authorization.Version] 为 0 ，则 Version 部分被省略。
func BuildAuthorizationHeader(auth Authorization) string {
	b := new(strings.Builder)
	b.WriteString(AuthorizationScheme)

	b.WriteString(" Key=")
	b.WriteString(auth.Key)

	b.WriteString(", Sign=")
	b.WriteString(auth.Sign)

	b.WriteString(", Timestamp=")
	b.WriteString(strconv.Itoa(auth.Timestamp))

	if auth.Version != 0 {
		b.WriteString(", Version=")
		b.WriteString(strconv.Itoa(auth.Version))
	}

	res := b.String()
	return res
}

// ParseAuthorizationHeader 解析 Authorization 头。
//
// 格式为：
//
//	Authorization: Scheme Key=value_of_key, Sign=value_of_sign, Timestamp=unix_timestamp, Version=1
//
// 说明：
//   - 每个 Key 前的空格被忽略。
//   - key-value 对的顺序不做要求。
//   - Scheme 必须是 SLIM-AUTH 。
//   - Timestamp 签名时的 UNIX 时间戳，单位是秒。
//   - Version 可省略，省略时默认为 1 。
func ParseAuthorizationHeader(r *http.Request) (Authorization, error) {
	auth := Authorization{}

	headers, ok := r.Header[HttpHeaderAuthorization]
	if !ok {
		return auth, fmt.Errorf("missing the Authorization header")
	}

	if len(headers) > 1 {
		return auth, fmt.Errorf("more than one Authorization headers found")
	}

	// Read <Scheme> part.
	header := headers[0]
	idx := strings.Index(header, " ")
	if idx <= 0 {
		return auth, fmt.Errorf("Authorization scheme error")
	}

	scheme := header[:idx]
	if scheme != AuthorizationScheme {
		return auth, fmt.Errorf("Authorization scheme error")
	}

	// Read params.
	parts := strings.Split(header[idx+1:], ",")
	hasVersion := false
	for _, part := range parts {
		part = strings.TrimSpace(part)
		kv := strings.Split(part, "=")

		switch kv[0] {
		case "Key":
			auth.Key = kv[1]

		case "Sign":
			auth.Sign = kv[1]

		case "Version":
			v, err := strconv.Atoi(kv[1])
			if err != nil {
				return auth, fmt.Errorf("Authorization version error: %w", err)
			}
			auth.Version = v
			hasVersion = true

		case "Timestamp":
			v, err := strconv.Atoi(kv[1])
			if err != nil {
				return auth, fmt.Errorf("Authorization timestamp error: %w", err)
			}
			auth.Timestamp = v
		}
	}

	if !hasVersion {
		auth.Version = DefaultSignVersion
	}

	return auth, nil
}

// HmacSha256 计算 hmac-sha256 ，返回小写的 HEX 格式。
func HmacSha256(secret, data []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write(data)
	hash := hex.EncodeToString(h.Sum(nil))
	return hash
}

// 表示签名执行的结果和错误原因（当有错误时）。
type SignResult struct {
	Sign  string         // 签名成功时，为签名的值。
	Type  SignResultType // 签名结果。
	Cause error          // 签名失败时，记录原因。
}

// 签名结果。
type SignResultType int

const (
	SignResultType_OK                     SignResultType = iota // 签名成功。
	SignResultType_MissingContentType                           // 当 POST 请求缺少 Content-Type 头时给定此错误。
	SignResultType_UnsupportedContentType                       // 当有 POST 请求有 Content-Type 头，但类型不受支持时给定此错误。
	SignResultType_InvalidFormData                              // 表单请求的表单数据格式缺失或不正确。
)

// AppendSign 计算请求的签名，并将其赋值到请求的 Authorization 头。
// 调用此方法后， [http.Request.Body] 会被读取并重新置换为新的 [bytes.Buffer] 。
//   - r 需要计算签名的请求。
//   - accessKey 对应 Authorization 头中的 Key 字段的值。
//   - secret HMAC-SHA256 的密钥，使用 UTF-8 字符集。
//   - timestamp UNIX 时间戳，对应 Authorization 头的 Timestamp 字段的值。
func AppendSign(r *http.Request, accessKey, secret string, timestamp int) SignResult {
	// 追加 Authorization 头的请求基本上是用来发送的，而不是服务器接收到的。
	// 这种情况下 HTTP body 需要是可用的，故总是设置参数 rewind=true 。
	res := Sign(r, true, secret, timestamp)
	if res.Type != SignResultType_OK {
		return res
	}

	auth := BuildAuthorizationHeader(Authorization{
		Key:       accessKey,
		Sign:      res.Sign,
		Timestamp: timestamp,
		Version:   DefaultSignVersion,
	})

	r.Header.Set(HttpHeaderAuthorization, auth)
	return res
}

// Sign 计算给定的请求的签名。
//   - r 需要计算签名的请求。。
//   - rewindBody 指定是否需要重用 [http.Request.Body] 。
//     若为 true ，则读取完 body 后，它会被替换为新的、可重读的 [bytes.Buffer] 。
//     此设置在表单请求下不会生效，表单解析后应通过 [http.Request.Form]/[http.Request.PostForm] 访问。
//   - secret HMAC-SHA256 的密钥，使用 UTF-8 字符集。
//   - timestamp UNIX 时间戳，对应 Authorization 头的 Timestamp 字段的值。
func Sign(r *http.Request, rewindBody bool, secret string, timestamp int) SignResult {
	data, typ, err := buildDataToSign(r, rewindBody, timestamp)
	if typ != SignResultType_OK {
		return SignResult{
			Type:  typ,
			Cause: err,
		}
	}

	hash := HmacSha256([]byte(secret), data)
	return SignResult{
		Sign: hash,
	}
}

// 构建用于签名的串，各部分末尾带一个换行符（ \n ）分割，依次为：
//   - TIMESTAMP UNIX 时间戳，需和 Authorization 头里的一样。
//   - METHOD 是 HTTP 请求的 METHOD ，如 GET/POST 。
//   - PATH 请求的路径，包含开头的 / ，比如请求地址是 http://temp.org/the/path/ 则路径为 /the/path/ ；如果没有路径部分，使用 / 。
//   - QUERY 是 URL 的参数表，按参数名称字典顺序升序，然后将值部分紧密拼接起来（无分隔符）。没有参数时，使用一个空字符串。
//   - BODY 若是表单类型，则处理方式同 QUERY ；若是 JSON 请求，则为 JSON 原文。 GET 请求时此部分省略（包含换行符）。
//   - 最后一行固定是“END”。
func buildDataToSign(r *http.Request, rewindBody bool, timestamp int) ([]byte, SignResultType, error) {
	buf := new(bytes.Buffer)

	// TIMESTAMP
	buf.WriteString(strconv.Itoa(timestamp))
	buf.WriteRune('\n')

	// METHOD
	buf.WriteString(r.Method)
	buf.WriteRune('\n')

	// PATH
	if r.URL.Path == "" {
		buf.WriteRune('/')
	} else {
		buf.WriteString(r.URL.Path)
	}
	buf.WriteRune('\n')

	// QUERY
	appendQueryWithNewLine(buf, r.URL.Query())

	// BODY
	if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
		contentType, ok := r.Header[webapi.HttpHeaderContentType]
		if !ok {
			err := fmt.Errorf("missing Content-Type")
			return nil, SignResultType_MissingContentType, err
		}

		switch contentType[0] {
		case webapi.ContentTypeForm:
			err := r.ParseForm()
			if err != nil {
				return nil, SignResultType_InvalidFormData, err
			}
			appendQueryWithNewLine(buf, r.PostForm)

		case webapi.ContentTypeJson:
			if rewindBody {
				data, err := repeatableReadBody(r)
				if err != nil {
					panic(err)
				}
				buf.Write(data)
			} else {
				io.Copy(buf, r.Body)
			}
			buf.WriteRune('\n')

		default:
			err := fmt.Errorf("unsupported Content-Type: %s", contentType[0])
			return nil, SignResultType_UnsupportedContentType, err
		}
	}

	// END
	buf.WriteString("END")

	return buf.Bytes(), SignResultType_OK, nil
}

func appendQueryWithNewLine(buf *bytes.Buffer, values url.Values) {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		for _, v := range values[k] {
			buf.WriteString(v)
		}
	}
	buf.WriteRune('\n')
}

// 读取整个 [http.Request.Body] 并返回读取到数据。
// 读取完毕后，原 body 会被关闭， Body 字段被替换为新的、未被读取的 [bytes.Buffer] ，其包含读取到数据。
// 此方法用于处理 body 的重复读取。
func repeatableReadBody(r *http.Request) ([]byte, error) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	err = r.Body.Close()
	if err != nil {
		return nil, err
	}

	r.Body = io.NopCloser(bytes.NewBuffer(data))
	return data, nil
}
