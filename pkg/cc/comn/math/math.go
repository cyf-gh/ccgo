package st_comn_math

func Abs( n int ) int {
	if n < 0 {
		return -n;
	} else {
		return n
	}
}

func IsEven( n int ) bool {
	return ( n%2 == 0 )
}