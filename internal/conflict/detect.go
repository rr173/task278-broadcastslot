// Package conflict 检测台呼重叠、广告延播，以及来源引用成环。
package conflict

import (
	"fmt"

	"task278-broadcastslot/internal/model"
	"task278-broadcastslot/internal/sequence"
)

// Window 是一条已校正条目的检测窗口。
type Window struct {
	EntryID    int64
	Callsign   string
	PageID     string
	PrintedStart int64
	UTCStart   int64
	UTCEnd     int64
}

// Detect 扫描台呼重叠与广告延播，不修改输入切片。
func Detect(windows []Window, ads []model.NewspaperAd) []model.AttributionConflict {
	out := make([]model.AttributionConflict, 0)
	out = append(out, callsignOverlaps(windows)...)
	out = append(out, adDelayed(windows, ads)...)
	return out
}

func callsignOverlaps(windows []Window) []model.AttributionConflict {
	var found []model.AttributionConflict
	for i := 0; i < len(windows); i++ {
		for j := i + 1; j < len(windows); j++ {
			a, b := windows[i], windows[j]
			if a.Callsign == "" || a.Callsign != b.Callsign {
				continue
			}
			if !sequence.Overlaps(a.UTCStart, a.UTCEnd, b.UTCStart, b.UTCEnd) {
				continue
			}
			found = append(found, model.AttributionConflict{
				LeftEntryID:  a.EntryID,
				RightEntryID: b.EntryID,
				Kind:         model.ConflictCallsignOverlap,
				Detail: fmt.Sprintf("callsign %s utc [%d,%d) overlaps [%d,%d)",
					a.Callsign, a.UTCStart, a.UTCEnd, b.UTCStart, b.UTCEnd),
			})
		}
	}
	return found
}

func adDelayed(windows []Window, ads []model.NewspaperAd) []model.AttributionConflict {
	var found []model.AttributionConflict
	for _, w := range windows {
		if w.PageID == "" {
			continue
		}
		for _, ad := range ads {
			if ad.PageID != w.PageID {
				continue
			}
			delay := ad.PrintedStartMS - w.PrintedStart
			if delay <= model.AdDelayThresholdMS {
				continue
			}
			found = append(found, model.AttributionConflict{
				LeftEntryID:  w.EntryID,
				RightEntryID: 0,
				Kind:         model.ConflictAdDelayed,
				Detail: fmt.Sprintf("page %s ad printed %d is %dms later than entry printed %d",
					w.PageID, ad.PrintedStartMS, delay, w.PrintedStart),
			})
		}
	}
	return found
}

// Involves 冲突是否触及指定条目。
func Involves(c model.AttributionConflict, entryID int64) bool {
	return c.LeftEntryID == entryID || c.RightEntryID == entryID
}
