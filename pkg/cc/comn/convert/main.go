package convert

import "strconv"

func Atoi64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
