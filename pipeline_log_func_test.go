package webapi

import (
	"testing"

	"github.com/cmstar/go-logx"
	"github.com/cmstar/go-logx/logxtest"
	"github.com/stretchr/testify/assert"
)

func TestLogFuncPipeline(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		p := NewLogFuncPipeline()
		p.Log(&ApiState{}) // Nothing happens.
	})

	t.Run("values", func(t *testing.T) {
		f1 := func(state *ApiState) {
			state.LogMessage = append(state.LogMessage, "K1", "V1")
			state.LogMessage = append(state.LogMessage, "K2", "V2")
		}

		f2 := func(state *ApiState) {
			state.LogLevel = logx.LevelWarn
			state.LogMessage = append(state.LogMessage, "K3", "V3")
		}

		p := NewLogFuncPipeline(f1, f2)
		logger := logxtest.NewRecorder()
		p.Log(&ApiState{Logger: logger})

		assert.Len(t, logger.Messages, 1)
		msg := logger.Messages[0]
		assert.Equal(t, logx.LevelWarn, msg.Level)
		assert.Equal(t, []any{"K1", "V1", "K2", "V2", "K3", "V3"}, msg.KeyValues)
	})

	t.Run("default-level", func(t *testing.T) {
		f1 := func(state *ApiState) {
			state.LogMessage = append(state.LogMessage, "K", "V")
		}

		p := NewLogFuncPipeline(f1)
		logger := logxtest.NewRecorder()
		p.Log(&ApiState{Logger: logger})

		assert.Len(t, logger.Messages, 1)
		msg := logger.Messages[0]
		assert.Equal(t, logx.LevelInfo, msg.Level)
		assert.Equal(t, []any{"K", "V"}, msg.KeyValues)
	})
}
