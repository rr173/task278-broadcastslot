// Package clockfix 把印刷本地时刻校正为播出 UTC：先扣时区，再按 ppm 漂移相对参考钟修正。
package clockfix

import "task278-broadcastslot/internal/model"

// Apply 实现 utc = printed - tzOffsetMs - int64(float64(printed-ref)*driftPPM/1e6)。
func Apply(printed, tzOffsetMS, ref int64, driftPPM float64) int64 {
	drift := int64(float64(printed-ref) * driftPPM / 1e6)
	return printed - tzOffsetMS - drift
}

// RefPrinted 取该批最早报纸广告 printed_start；无广告则返回 ok=false，调用方应以 printed 自身为 ref（漂移项为 0）。
func RefPrinted(ads []model.NewspaperAd) (int64, bool) {
	if len(ads) == 0 {
		return 0, false
	}
	min := ads[0].PrintedStartMS
	for i := 1; i < len(ads); i++ {
		if ads[i].PrintedStartMS < min {
			min = ads[i].PrintedStartMS
		}
	}
	return min, true
}

// CorrectOne 校正单个印刷时刻。无参考钟时 ref=printed，漂移项为 0。
func CorrectOne(printed int64, tz string, driftPPM float64, ref int64, hasRef bool) (int64, error) {
	off, err := OffsetMS(tz)
	if err != nil {
		return 0, err
	}
	useRef := printed
	if hasRef {
		useRef = ref
	}
	return Apply(printed, off, useRef, driftPPM), nil
}

// CorrectSlot 同时校正起止时刻；校正后 end<=start 返回 ErrSlotInverted。
func CorrectSlot(printedStart, printedEnd int64, tz string, driftPPM float64, ref int64, hasRef bool) (utcStart, utcEnd int64, err error) {
	utcStart, err = CorrectOne(printedStart, tz, driftPPM, ref, hasRef)
	if err != nil {
		return 0, 0, err
	}
	utcEnd, err = CorrectOne(printedEnd, tz, driftPPM, ref, hasRef)
	if err != nil {
		return 0, 0, err
	}
	if utcEnd <= utcStart {
		return 0, 0, model.ErrSlotInverted
	}
	return utcStart, utcEnd, nil
}
