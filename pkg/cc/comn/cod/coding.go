package st_comn_cod

func BinToUTF8( b []byte ) string {
	bytes := b
	utf := ""
	for len(bytes)>0 {
		ch, size := utf8.DecodeRune(bytes)
		bytes = bytes[size:]
		utf += fmt.Sprint( "$c", ch )
	}
	return utf
}

