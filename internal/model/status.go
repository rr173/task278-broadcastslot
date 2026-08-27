package model

// 证据批次状态。封存后只读。
const (
	BatchOrganizing     = "organizing"
	BatchPendingAlign   = "pending_align"
	BatchPendingVerdict = "pending_verdict"
	BatchPublished      = "published"
	BatchSealed         = "sealed"
)

// 节目条目状态。
const (
	EntryRaw      = "raw"
	EntryAligned  = "aligned"
	EntryConflict = "conflict"
	EntryExcluded = "excluded"
)

// 时段归属状态。
const (
	AttrCandidate     = "candidate"
	AttrFeasible      = "feasible"
	AttrClockConflict = "clock_conflict"
	AttrConfirmed     = "confirmed"
	AttrRejected      = "rejected"
)

// 播出表版本状态。
const (
	VersionDraft      = "draft"
	VersionShared     = "shared"
	VersionFrozen     = "frozen"
	VersionSuperseded = "superseded"
)

// 冲突种类。
const (
	ConflictCallsignOverlap = "callsign_overlap"
	ConflictAdDelayed       = "ad_delayed"
)

// 裁决结论。
const (
	DecisionConfirmed = "confirmed"
	DecisionRejected  = "rejected"
)

// 广告延播阈值：同页广告印刷开始晚于条目超过 5 分钟。
const AdDelayThresholdMS int64 = 300000

// CanTransitionBatch 判断批次状态机是否允许 from → to。
func CanTransitionBatch(from, to string) bool {
	if from == to {
		return false
	}
	if from == BatchSealed {
		return false
	}
	switch from {
	case BatchOrganizing:
		return to == BatchPendingAlign
	case BatchPendingAlign:
		return to == BatchPendingVerdict
	case BatchPendingVerdict:
		return to == BatchPublished
	case BatchPublished:
		return to == BatchSealed
	default:
		return false
	}
}

// ValidBatchStatus 是否为已知批次状态。
func ValidBatchStatus(s string) bool {
	switch s {
	case BatchOrganizing, BatchPendingAlign, BatchPendingVerdict, BatchPublished, BatchSealed:
		return true
	default:
		return false
	}
}

// ValidEntryStatus 是否为已知条目状态。
func ValidEntryStatus(s string) bool {
	switch s {
	case EntryRaw, EntryAligned, EntryConflict, EntryExcluded:
		return true
	default:
		return false
	}
}

// ValidAttrStatus 是否为已知归属状态。
func ValidAttrStatus(s string) bool {
	switch s {
	case AttrCandidate, AttrFeasible, AttrClockConflict, AttrConfirmed, AttrRejected:
		return true
	default:
		return false
	}
}

// ValidVersionStatus 是否为已知版本状态。
func ValidVersionStatus(s string) bool {
	switch s {
	case VersionDraft, VersionShared, VersionFrozen, VersionSuperseded:
		return true
	default:
		return false
	}
}

// ValidDecision 是否为确认或否决。
func ValidDecision(d string) bool {
	return d == DecisionConfirmed || d == DecisionRejected
}
