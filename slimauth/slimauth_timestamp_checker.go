package slimauth

import (
	"fmt"
	"time"
)

// TimeCheckerFunc 用于校验签名信息中携带的时间戳的有效性。
// 若时间校验不通过，返回相关描述信息；否则返回 nil 表示校验通过。
// 若方法 panic ，其错误处理方式与普通的 API 方法一致。
type TimeCheckerFunc func(timestamp int64) error

var (
	// 不校验时间戳的 [TimeCheckerFunc] 。
	NoTimeChecker TimeCheckerFunc = func(timestamp int64) error {
		return nil
	}

	// 默认的时间戳校验 [TimeCheckerFunc] ：要求签名给定的时间戳与当前时间误差在 5 分钟内。
	DefaultTimeChecker TimeCheckerFunc = MaxDeviationTimeChecker(300)
)

// MaxDeviationTimeChecker 返回一个 [TimeCheckerFunc] ，
// 其校验给定的时间戳与当前时间的误差必须小于等于 maxDeviation ，单位为秒。
// maxDeviation 应为非负数，否则校验总是通过。
func MaxDeviationTimeChecker(maxDeviation int64) TimeCheckerFunc {
	return func(timestamp int64) error {
		now := time.Now().Unix()
		d := now - timestamp

		// ABS().
		if d < 0 {
			d = -d
		}

		if d > maxDeviation {
			return fmt.Errorf("the deviation of time should be less than %ds, the time is %d, got %d", maxDeviation, now, timestamp)
		}

		return nil
	}
}
