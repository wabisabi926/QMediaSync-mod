package helpers

import "strconv"

func Int64toInt(i64 int64) int {
	strInt64 := strconv.FormatInt(i64, 10)
	i16, _ := strconv.Atoi(strInt64)
	return i16
}

func IntToString(i16 int) string {
	return strconv.Itoa(i16)
}

func Int64ToString(i64 int64) string {
	return strconv.FormatInt(i64, 10)
}
