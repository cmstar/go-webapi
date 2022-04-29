package slimapi

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cmstar/go-conv"
)

const (
	// SlimAPI 协议里默认的时间格式： yyyyMMdd HH:mm:ss 。 API 输出 JSON 时使用此格式。
	TimeFormat = "2006-01-02 15:04:05"

	// SlimAPI 协议里默认的时间格式的微秒版本，解析时间时使用此格式。
	timeFormatNano = "2006-01-02 15:04:05.999999"
)

// Time 描述 SlimAPI 协议中的时间。
//
// 在 API response 中，使用此类型，可输出 SlimAPI 格式的时间。
// e.g.
//  type SomeResponse struct {
//      CreateTime time.Time    // 在 JSON 中使用默认的格式（RFC3339）输出。
//      UpdateTime slimapi.Time // 在 JSON 中使用 SlimAPI 规定的格式输出： yyyy-MM-dd HH:mm:ss 。
//  }
//
type Time time.Time

var _ json.Marshaler = (*Time)(nil)
var _ fmt.Stringer = (*Time)(nil)

// Time 将 slimapi.Time 转换到标准库的 time.Time 。
func (t Time) Time() time.Time {
	return time.Time(t)
}

// Implements fmt.Stringer.
func (t Time) String() string {
	return time.Time(t).Format(TimeFormat)
}

// Implements json.Marshaler.
func (t Time) MarshalJSON() ([]byte, error) {
	v := `"` + time.Time(t).Format(TimeFormat) + `"`
	return []byte(v), nil
}

/*
当前只实现 json.Marshaler ，不实现 Unmarshaler 。
目前 JSON 数据会转化到 map ，再从 map 通过 conv 转换，绕过了 json.Unmarshal 。
func (t *Time) UnmarshalJSON(b []byte) error {
	s := string(b)

	// 去头尾引号。
	if len(s) < len(TimeFormat) || s[0] != '"' || s[len(s)-1] != '"' {
		return fmt.Errorf(`cannot parse %s as %q`, s, TimeFormat)
	}
	s = s[1:]
	s = s[:len(s)-1]

	v, err := parseTime(s)
	if err != nil {
		return err
	}
	*t = Time(v)
	return nil
}
*/

// parseTime 解析 SlimAPI 的时间格式，默认 yyyyMMdd HH:mm:ss 时区为 UTC ，如果解析失败再用默认的格式（ RFC3339 ）处理。
func parseTime(v string) (time.Time, error) {
	t, err := time.Parse(timeFormatNano, v)
	if err == nil {
		return t.UTC(), nil
	}

	t, err2 := conv.DefaultStringToTime(v)
	if err2 == nil {
		return t, nil // 不做时区转换，像 RFC3339 这种格式是自带时区信息的。
	}

	// 错误信息以最初的格式为准。
	return time.Time{}, err
}
