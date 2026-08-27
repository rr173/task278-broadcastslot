// Package model 定义历史广播节目单复核域的实体、状态机与哨兵错误。
package model

import (
	"errors"
	"fmt"
)

// 领域哨兵错误。HTTP 层必须用 errors.Is 映射，store 写入冲突必须用 %w 包装。
var (
	ErrBatchNotFound        = errors.New("batch not found")
	ErrDuplicateCode        = errors.New("duplicate batch code")
	ErrDuplicateFingerprint = errors.New("duplicate entry fingerprint")
	ErrVersionConflict      = errors.New("optimistic version conflict")
	ErrSourceCycle          = errors.New("source citation cycle")
	ErrSlotInverted         = errors.New("corrected slot inverted")
	ErrSealed               = errors.New("batch is sealed")
	ErrIllegalTransition    = errors.New("illegal status transition")
	ErrUnknownTimezone      = errors.New("unknown timezone")
	ErrFrozenVersion        = errors.New("frozen version payload is immutable")
	ErrInvalidDecision      = errors.New("invalid verdict decision")
	ErrEmptyCode            = errors.New("batch code is empty")
	ErrEmptyTitle           = errors.New("entry title is empty")
	ErrInvalidID            = errors.New("invalid identifier")
	ErrEntryNotFound        = errors.New("entry not found")
	ErrVersionNotFound      = errors.New("schedule version not found")
	ErrAttributionNotFound  = errors.New("attribution not found")
)

// WrapSentinel 用 %w 包装哨兵，保证 errors.Is 可识别。禁止改成 %v。
func WrapSentinel(sentinel error, detail string) error {
	if detail == "" {
		return sentinel
	}
	return fmt.Errorf("%s: %w", detail, sentinel)
}
