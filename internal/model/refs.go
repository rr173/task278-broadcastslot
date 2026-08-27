package model

import (
	"fmt"
	"strconv"
	"strings"
)

func formatRef(kind string, n int64) string {
	return kind + ":" + strconv.FormatInt(n, 10)
}

// ParseRef 解析 "kind:id" 形式的来源引用。
func ParseRef(ref string) (kind string, id int64, err error) {
	kind, id, ok := splitRef(ref)
	if !ok {
		return "", 0, fmt.Errorf("%w: %s", ErrInvalidID, ref)
	}
	return kind, id, nil
}

func splitRef(ref string) (string, int64, bool) {
	kind, rest, found := strings.Cut(ref, ":")
	if !found || kind == "" || rest == "" {
		return "", 0, false
	}
	n, err := strconv.ParseInt(rest, 10, 64)
	if err != nil || n <= 0 {
		return "", 0, false
	}
	return kind, n, true
}
