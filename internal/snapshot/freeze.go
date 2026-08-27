// Package snapshot 在冻结播出表时深拷贝现场证据并计算内容哈希。
package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"task278-broadcastslot/internal/model"
)

// Payload 是冻结版本中保存的不可变证据包。
type Payload struct {
	Entries      []model.ProgramEntry         `json:"entries"`
	Attributions []model.SlotAttribution      `json:"attributions"`
	Conflicts    []model.AttributionConflict  `json:"conflicts"`
	Verdicts     []model.SlotVerdict          `json:"verdicts"`
	Citations    []model.SourceCitation       `json:"citations"`
}

// Freeze 深拷贝后序列化为 JSON，并计算 SHA-256 hex。后续改 live 切片不得影响已冻结字节。
func Freeze(live Payload) (payloadJSON []byte, contentHash string, err error) {
	cloned := Clone(live)
	raw, err := json.Marshal(cloned)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(raw)
	return raw, hex.EncodeToString(sum[:]), nil
}

// Clone 对五类切片做值拷贝，切断与 live 底层数组的共享。
func Clone(live Payload) Payload {
	return Payload{
		Entries:      cloneEntries(live.Entries),
		Attributions: cloneAttrs(live.Attributions),
		Conflicts:    cloneConflicts(live.Conflicts),
		Verdicts:     cloneVerdicts(live.Verdicts),
		Citations:    cloneCitations(live.Citations),
	}
}

// Parse 把存库 JSON 还原为 Payload，供 GetVersion 原样返回，禁止用 live 数据重算。
func Parse(raw []byte) (Payload, error) {
	var p Payload
	if len(raw) == 0 {
		return p, nil
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return Payload{}, err
	}
	return p, nil
}

func cloneEntries(in []model.ProgramEntry) []model.ProgramEntry {
	if in == nil {
		return nil
	}
	out := make([]model.ProgramEntry, len(in))
	copy(out, in)
	return out
}

func cloneAttrs(in []model.SlotAttribution) []model.SlotAttribution {
	if in == nil {
		return nil
	}
	out := make([]model.SlotAttribution, len(in))
	copy(out, in)
	return out
}

func cloneConflicts(in []model.AttributionConflict) []model.AttributionConflict {
	if in == nil {
		return nil
	}
	out := make([]model.AttributionConflict, len(in))
	copy(out, in)
	return out
}

func cloneVerdicts(in []model.SlotVerdict) []model.SlotVerdict {
	if in == nil {
		return nil
	}
	out := make([]model.SlotVerdict, len(in))
	copy(out, in)
	return out
}

func cloneCitations(in []model.SourceCitation) []model.SourceCitation {
	if in == nil {
		return nil
	}
	out := make([]model.SourceCitation, len(in))
	copy(out, in)
	return out
}
