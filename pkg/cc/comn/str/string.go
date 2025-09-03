package st_comn_str

func WrapWith( raw string, left string, right string ) string {
	return left + raw + right
}

func End( raw string ) byte {
	ei := EndIndex(raw)
	return raw[ei]
}

func EndIndex( raw string ) int {
	return len(raw) - 1
}