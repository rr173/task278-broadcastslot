// Package evidence 计算节目条目指纹，并判定同指纹导入是幂等还是冲突。
package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	"task278-broadcastslot/internal/model"
)

const fingerprintHexLen = 16

// Fingerprint 计算 SHA-256(title|callsign|printed_start|printed_end|page_id) 的 hex 前 16 位。
func Fingerprint(title, callsign string, printedStart, printedEnd int64, pageID string) string {
	raw := title + "|" + callsign + "|" +
		strconv.FormatInt(printedStart, 10) + "|" +
		strconv.FormatInt(printedEnd, 10) + "|" +
		pageID
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])[:fingerprintHexLen]
}

// Fill 为条目写入指纹。标题为空则拒绝。
func Fill(entry *model.ProgramEntry) error {
	if entry == nil {
		return model.ErrInvalidID
	}
	if entry.Title == "" {
		return model.ErrEmptyTitle
	}
	entry.Fingerprint = Fingerprint(
		entry.Title,
		entry.Callsign,
		entry.PrintedStartMS,
		entry.PrintedEndMS,
		entry.PageID,
	)
	return nil
}

// Resolve 同指纹同字段返回已有行；同指纹不同字段返回 ErrDuplicateFingerprint。
// existing 为 nil 表示库中尚无该指纹，应新建 incoming。
func Resolve(existing *model.ProgramEntry, incoming model.ProgramEntry) (*model.ProgramEntry, error) {
	if incoming.Fingerprint == "" {
		if err := Fill(&incoming); err != nil {
			return nil, err
		}
	}
	if existing == nil {
		cp := incoming
		return &cp, nil
	}
	if existing.Fingerprint != incoming.Fingerprint {
		return nil, model.ErrDuplicateFingerprint
	}
	if existing.SameIdentity(incoming) {
		return existing, nil
	}
	return nil, model.ErrDuplicateFingerprint
}
