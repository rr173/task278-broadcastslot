package clockfix

import "task278-broadcastslot/internal/model"

const hourMS int64 = 3600 * 1000

// OffsetMS 返回固定时区表中的毫秒偏移。未知时区返回 ErrUnknownTimezone。
// CST 与 Asia/Shanghai 均为 +8h，UTC 为 0。
func OffsetMS(tz string) (int64, error) {
	switch tz {
	case "CST", "Asia/Shanghai":
		return 8 * hourMS, nil
	case "UTC", "utc":
		return 0, nil
	default:
		return 0, model.WrapSentinel(model.ErrUnknownTimezone, tz)
	}
}

// KnownTimezone 是否落在固定表内。
func KnownTimezone(tz string) bool {
	_, err := OffsetMS(tz)
	return err == nil
}
