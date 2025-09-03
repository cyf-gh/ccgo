package st_comn_err

import (
	"github.com/kpango/glg"
)

func Exsit( err error ) bool {
	if err != nil {
		glg.Error( err )
		return true
	}
	return false
}