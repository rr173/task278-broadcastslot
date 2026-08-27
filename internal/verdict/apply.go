// Package verdict 按乐观版本确认或否决时段归属。
package verdict

import "task278-broadcastslot/internal/model"

// Apply 校验 expected_version 与当前最新播出表版本一致，且 decision 合法。
// 版本不匹配返回 ErrVersionConflict；不写库，由 service 在持锁后落盘。
func Apply(currentVersion, expectedVersion int64, decision string) error {
	if currentVersion != expectedVersion {
		return model.ErrVersionConflict
	}
	if !model.ValidDecision(decision) {
		return model.ErrInvalidDecision
	}
	return nil
}

// NextAttrStatus 把裁决映射到归属状态。
func NextAttrStatus(decision string) (string, error) {
	switch decision {
	case model.DecisionConfirmed:
		return model.AttrConfirmed, nil
	case model.DecisionRejected:
		return model.AttrRejected, nil
	default:
		return "", model.ErrInvalidDecision
	}
}
