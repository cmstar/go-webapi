package webapitest

import (
	"testing"

	"github.com/cmstar/go-logx"
	"github.com/stretchr/testify/assert"
)

func TestLogRecorder(t *testing.T) {
	r := NewLogRecorder()
	assert := assert.New(t)
	assert.Empty(r.String())

	r.Log(logx.LevelDebug, "")
	r.Log(logx.LevelError, "msg")
	r.Log(logx.LevelInfo, "", "k1", "v1", "k2", 2, 3)
	r.Log(logx.LevelInfo, "msg", "k1", "v1")

	res := r.String()
	want := `level=DEBUG message=
level=ERROR message=msg
level=INFO message= k1=v1 k2=2 UNKNOWN=3
level=INFO message=msg k1=v1
`
	assert.Equal(want, res)

	checkMap := func(idx int, key, wantValue string) {
		m := r.m[idx]
		v := m[key]
		assert.Equal(wantValue, v)
	}

	checkMap(0, "level", "DEBUG")
	checkMap(0, "message", "")

	checkMap(1, "level", "ERROR")
	checkMap(1, "message", "msg")

	checkMap(2, "level", "INFO")
	checkMap(2, "message", "")
	checkMap(2, "k1", "v1")
	checkMap(2, "k2", "2")
	checkMap(2, "UNKNOWN", "3")

	checkMap(3, "level", "INFO")
	checkMap(3, "message", "msg")
	checkMap(3, "k1", "v1")
}
