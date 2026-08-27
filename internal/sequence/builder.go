// Package sequence 按校正后 utc_start 构造可行播出序列。
// Builder 复用 workBuf；对外返回前必须 copy 结果以及每个 Slot.Sources。
package sequence

import (
	"sort"

	"task278-broadcastslot/internal/model"
)

// Item 是构造序列的输入：一条已校正的节目条目。
type Item struct {
	EntryID     int64
	Callsign    string
	Transmitter string
	UTCStart    int64
	UTCEnd      int64
	Sources     []string
}

// Slot 是序列中的一个播出时段。
type Slot struct {
	EntryID     int64
	Callsign    string
	Transmitter string
	UTCStart    int64
	UTCEnd      int64
	Sources     []string
	Feasible    bool
}

// Overlaps 两时段开区间相交。
func Overlaps(aStart, aEnd, bStart, bEnd int64) bool {
	return aStart < bEnd && bStart < aEnd
}

// Builder 可复用内部缓冲，避免每次对齐都重新分配。
type Builder struct {
	workBuf []Slot
}

// Build 按 utc_start 排序后扫描同发射机重叠。先出现的可行，后出现的标为不可行。
// 返回切片与 Sources 均已 copy，调用方可继续复用 Builder。
func (b *Builder) Build(items []Item) []Slot {
	n := len(items)
	if cap(b.workBuf) < n {
		b.workBuf = make([]Slot, n)
	} else {
		b.workBuf = b.workBuf[:n]
	}
	for i, it := range items {
		src := it.Sources
		b.workBuf[i] = Slot{
			EntryID:     it.EntryID,
			Callsign:    it.Callsign,
			Transmitter: it.Transmitter,
			UTCStart:    it.UTCStart,
			UTCEnd:      it.UTCEnd,
			Sources:     src,
			Feasible:    true,
		}
	}
	sort.SliceStable(b.workBuf, func(i, j int) bool {
		if b.workBuf[i].UTCStart != b.workBuf[j].UTCStart {
			return b.workBuf[i].UTCStart < b.workBuf[j].UTCStart
		}
		return b.workBuf[i].EntryID < b.workBuf[j].EntryID
	})
	markTransmitterOverlap(b.workBuf)
	return cloneSlots(b.workBuf)
}

func markTransmitterOverlap(slots []Slot) {
	accepted := make([]int, 0, len(slots))
	for i := range slots {
		ok := true
		for _, j := range accepted {
			if slots[i].Transmitter == "" || slots[j].Transmitter == "" {
				continue
			}
			if slots[i].Transmitter != slots[j].Transmitter {
				continue
			}
			if Overlaps(slots[i].UTCStart, slots[i].UTCEnd, slots[j].UTCStart, slots[j].UTCEnd) {
				ok = false
				break
			}
		}
		slots[i].Feasible = ok
		if ok {
			accepted = append(accepted, i)
		}
	}
}

func cloneSlots(src []Slot) []Slot {
	out := make([]Slot, len(src))
	for i := range src {
		out[i] = src[i]
		if src[i].Sources != nil {
			cp := make([]string, len(src[i].Sources))
			copy(cp, src[i].Sources)
			out[i].Sources = cp
		}
	}
	return out
}

// CollectSources 把条目自身引用与匹配的台呼片段引用拼成来源列表。
func CollectSources(entry model.ProgramEntry, clips []model.StationClip) []string {
	out := []string{model.RefEntry(entry.ID)}
	for _, c := range clips {
		if c.Callsign == entry.Callsign {
			out = append(out, model.RefClip(c.ClipNo))
		}
	}
	return out
}
