package model

// EvidenceBatch 是一次节目单复核批次。
type EvidenceBatch struct {
	ID        int64  `json:"id"`
	Code      string `json:"code"`
	Station   string `json:"station"`
	AirDate   string `json:"air_date"`
	Timezone  string `json:"timezone"`
	DriftPPM  float64 `json:"drift_ppm"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	SealedAt  string `json:"sealed_at,omitempty"`
}

// Sealed 批次是否已封存。
func (b *EvidenceBatch) Sealed() bool {
	return b != nil && b.Status == BatchSealed
}

// ProgramEntry 是一份残页节目条目。
type ProgramEntry struct {
	ID             int64  `json:"id"`
	BatchID        int64  `json:"batch_id"`
	Fingerprint    string `json:"fingerprint"`
	Title          string `json:"title"`
	Callsign       string `json:"callsign"`
	PrintedStartMS int64  `json:"printed_start_ms"`
	PrintedEndMS   int64  `json:"printed_end_ms"`
	PageID         string `json:"page_id"`
	Transmitter    string `json:"transmitter"`
	Status         string `json:"status"`
}

// SameIdentity 比较构成条目指纹的字段以及发射机。
func (e ProgramEntry) SameIdentity(other ProgramEntry) bool {
	return e.Title == other.Title &&
		e.Callsign == other.Callsign &&
		e.PrintedStartMS == other.PrintedStartMS &&
		e.PrintedEndMS == other.PrintedEndMS &&
		e.PageID == other.PageID &&
		e.Transmitter == other.Transmitter
}

// StationClip 是录音台呼片段。
type StationClip struct {
	ID       int64  `json:"id"`
	BatchID  int64  `json:"batch_id"`
	ClipNo   int64  `json:"clip_no"`
	Callsign string `json:"callsign"`
	OffsetMS int64  `json:"offset_ms"`
	Source   string `json:"source"`
	Status   string `json:"status"`
}

// NewspaperAd 是报纸广告时刻。
type NewspaperAd struct {
	ID             int64  `json:"id"`
	BatchID        int64  `json:"batch_id"`
	AdNo           int64  `json:"ad_no"`
	PrintedStartMS int64  `json:"printed_start_ms"`
	PageID         string `json:"page_id"`
	Edition        string `json:"edition"`
	Note           string `json:"note"`
}

// SourceCitation 是来源互引边。
type SourceCitation struct {
	ID      int64  `json:"id"`
	BatchID int64  `json:"batch_id"`
	FromRef string `json:"from_ref"`
	ToRef   string `json:"to_ref"`
	Kind    string `json:"kind"`
}

// ClockCorrection 是一次印刷时刻到播出 UTC 的校正记录。
type ClockCorrection struct {
	ID          int64  `json:"id"`
	BatchID     int64  `json:"batch_id"`
	SubjectKind string `json:"subject_kind"`
	SubjectID   int64  `json:"subject_id"`
	PrintedMS   int64  `json:"printed_ms"`
	UTCMS       int64  `json:"utc_ms"`
	Method      string `json:"method"`
	AppliedAt   string `json:"applied_at"`
}

// SlotAttribution 是条目到播出时段的归属。
type SlotAttribution struct {
	ID         int64  `json:"id"`
	BatchID    int64  `json:"batch_id"`
	EntryID    int64  `json:"entry_id"`
	ClipID     int64  `json:"clip_id"`
	UTCStartMS int64  `json:"utc_start_ms"`
	UTCEndMS   int64  `json:"utc_end_ms"`
	Status     string `json:"status"`
	DelayMS    int64  `json:"delay_ms"`
}

// AttributionConflict 是对齐阶段检出的冲突。
type AttributionConflict struct {
	ID           int64  `json:"id"`
	BatchID      int64  `json:"batch_id"`
	LeftEntryID  int64  `json:"left_entry_id"`
	RightEntryID int64  `json:"right_entry_id"`
	Kind         string `json:"kind"`
	Detail       string `json:"detail"`
}

// SlotVerdict 是对归属的确认或否决。
type SlotVerdict struct {
	ID              int64  `json:"id"`
	BatchID         int64  `json:"batch_id"`
	EntryID         int64  `json:"entry_id"`
	Decision        string `json:"decision"`
	Reviewer        string `json:"reviewer"`
	Note            string `json:"note"`
	ExpectedVersion int64  `json:"expected_version"`
}

// ScheduleVersion 是一次播出表快照版本。
type ScheduleVersion struct {
	ID          int64  `json:"id"`
	BatchID     int64  `json:"batch_id"`
	Version     int64  `json:"version"`
	Status      string `json:"status"`
	Sealed      bool   `json:"sealed"`
	Payload     string `json:"payload,omitempty"`
	ContentHash string `json:"content_hash"`
	CreatedAt   string `json:"created_at"`
}

// Frozen 版本是否已冻结（payload 不可改）。
func (v *ScheduleVersion) Frozen() bool {
	return v != nil && (v.Status == VersionFrozen || v.Sealed)
}

// Stats 是全库计数。
type Stats struct {
	Batches         int `json:"batches"`
	Entries         int `json:"entries"`
	Clips           int `json:"clips"`
	Ads             int `json:"ads"`
	Citations       int `json:"citations"`
	Corrections     int `json:"corrections"`
	Attributions    int `json:"attributions"`
	Conflicts       int `json:"conflicts"`
	Verdicts        int `json:"verdicts"`
	Versions        int `json:"versions"`
	SealedBatches   int `json:"sealed_batches"`
	FrozenVersions  int `json:"frozen_versions"`
}

// RefEntry 构造条目标识。
func RefEntry(id int64) string {
	return formatRef("entry", id)
}

// RefClip 构造台呼片段标识（按 clip_no）。
func RefClip(clipNo int64) string {
	return formatRef("clip", clipNo)
}

// RefAd 构造广告标识（按 ad_no）。
func RefAd(adNo int64) string {
	return formatRef("ad", adNo)
}
