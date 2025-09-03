package st_cmon_test

import (
	"testing"
)

func Assert(shoudBeTrue bool, t *testing.T, args ...interface{})  {
	if !shoudBeTrue {
		t.Fatal(args)
	}
}
