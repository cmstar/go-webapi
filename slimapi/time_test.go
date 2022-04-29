package slimapi

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTime(t *testing.T) {
	// 2022-11-03 07:30:06
	in := time.Date(2022, 11, 03, 7, 30, 6, 0, time.UTC)
	tm := Time(in)

	t.Run("Time", func(t *testing.T) {
		assert.Equal(t, in, tm.Time())
	})

	t.Run("String", func(t *testing.T) {
		assert.Equal(t, `2022-11-03 07:30:06`, tm.String())
	})

	t.Run("Marshal", func(t *testing.T) {
		j, _ := json.Marshal(tm)
		assert.Equal(t, `"2022-11-03 07:30:06"`, string(j))
	})
}

func Test_parseTime(t *testing.T) {
	t.Run("short", func(t *testing.T) {
		v, err := parseTime(`2022-11-03 07:30:06`)
		assert.Nil(t, err)
		assert.Equal(t, time.Date(2022, 11, 03, 7, 30, 6, 0, time.UTC), v)
	})

	t.Run("long", func(t *testing.T) {
		v, err := parseTime(`2022-11-03 07:30:06.112233`)
		assert.Nil(t, err)
		assert.Equal(t, time.Date(2022, 11, 03, 7, 30, 6, int(112233*time.Microsecond), time.UTC), v)
	})

	t.Run("rfc3339", func(t *testing.T) {
		v, err := parseTime(`2022-11-03T07:30:06.112233Z`)
		assert.Nil(t, err)
		assert.Equal(t, time.Date(2022, 11, 03, 7, 30, 6, int(112233*time.Microsecond), time.UTC), v)
	})

	t.Run("err", func(t *testing.T) {
		v, err := parseTime(`x`)
		assert.NotNil(t, err)
		assert.Equal(t, time.Time{}, v)
		assert.Equal(t, `parsing time "x" as "2006-01-02 15:04:05.999999": cannot parse "x" as "2006"`, err.Error())
	})
}
