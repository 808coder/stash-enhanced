package sqlite

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

func durationToTinyIntFn(str string) (int64, error) {
	splits := strings.Split(str, ":")

	if len(splits) > 3 {
		return 0, nil
	}

	seconds := 0
	factor := 1
	for len(splits) > 0 {
		// pop the last split
		var thisSplit string
		thisSplit, splits = splits[len(splits)-1], splits[:len(splits)-1]

		thisInt, err := strconv.Atoi(thisSplit)
		if err != nil {
			return 0, nil
		}

		seconds += factor * thisInt
		factor *= 60
	}

	return int64(seconds), nil
}

func basenameFn(str string) (string, error) {
	return filepath.Base(str), nil
}

// lowerUnicode lower-cases s with Unicode case folding, unlike SQLite's
// lower() and LIKE, which fold ASCII only.
func lowerUnicode(s string) string {
	return strings.ToLower(s)
}

// lowerUnicodeFn is lowerUnicode as an SQL function. NULL yields an empty
// string.
func lowerUnicodeFn(v interface{}) (string, error) {
	switch v := v.(type) {
	case nil:
		return "", nil
	case string:
		return lowerUnicode(v), nil
	case []byte:
		return lowerUnicode(string(v)), nil
	default:
		return lowerUnicode(fmt.Sprint(v)), nil
	}
}
