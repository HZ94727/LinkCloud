package controller

import (
	"strconv"
	"strings"
)

func parseBinaryStatus(raw string) (int8, bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, false, nil
	}

	value, err := strconv.ParseInt(trimmed, 10, 8)
	if err != nil {
		return 0, false, err
	}

	status := int8(value)
	if status != 0 && status != 1 {
		return 0, false, nil
	}

	return status, true, nil
}
