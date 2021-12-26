package webapitest

import (
	"fmt"
	"strings"

	"github.com/cmstar/go-logx"
)

// NewLogRecorder 创建一个 LogRecorder 的新实例。
func NewLogRecorder() *LogRecorder {
	return &LogRecorder{
		buf: &strings.Builder{},
	}
}

// LogRecorder 实现 logx.Logger ，将全部日志追加记录在一个字符串上，每个日志末尾追加一个换行。
// 每个日志的字符串拼接格式为，格式化使用 fmt.Sprintf() ：
//   level={LEVEL} message={MESSAGE} KEY1=VALUE1 KEY2=VALUE2 ...
type LogRecorder struct {
	buf *strings.Builder
	m   []map[string]string
}

var _ logx.Logger = (*LogRecorder)(nil)

// Log 实现 Logger.Log() 。
func (l *LogRecorder) Log(level logx.Level, message string, keyValues ...interface{}) error {
	m := make(map[string]string)
	l.m = append(l.m, m)

	lv := logx.LevelToString(level)
	l.buf.WriteString("level=")
	l.buf.WriteString(lv)
	m["level"] = lv

	l.buf.WriteString(" message=")
	l.buf.WriteString(message)
	m["message"] = message

	length := len(keyValues)
	for i := 0; i < length-1; i += 2 {
		k := fmt.Sprintf("%v", keyValues[i])
		v := fmt.Sprintf("%v", keyValues[i+1])

		l.buf.WriteByte(' ')
		l.buf.WriteString(k)
		l.buf.WriteByte('=')
		l.buf.WriteString(v)

		m[k] = v
	}

	if length%2 != 0 {
		v := fmt.Sprintf("%v", keyValues[length-1])
		l.buf.WriteString(" UNKNOWN=")
		l.buf.WriteString(v)
		m["UNKNOWN"] = v
	}

	l.buf.WriteByte('\n')
	return nil
}

// Log 实现 Logger.LogFn() 。
func (l *LogRecorder) LogFn(level logx.Level, messageFactory func() (string, []interface{})) error {
	m, kv := messageFactory()
	return l.Log(level, m, kv...)
}

// String 返回当前记录的完整日志。
func (l *LogRecorder) String() string {
	if l.buf == nil {
		return ""
	}
	return l.buf.String()
}

// Map 返回结构化日志。每条日志使用一个 map 记录。
func (l *LogRecorder) Map() []map[string]string {
	return l.m
}
