package cli

import (
	"io/ioutil"
	"os"

	"github.com/kpango/glg"
)

var (
	Banner string
)

func init() {
	Banner = ""
	b, e := ioutil.ReadFile("./banner.txt")
	if e != nil {
		glg.Fail("load banner")
		glg.Error(e)
	}
	Banner = string(b)
}

func initClis() {
	Register("echo", &CliFuncPack{echo, "Echo text", "basic"})
	Register("help", &CliFuncPack{help, "List all commands and descriptions", "basic"})
	Register("stop", &CliFuncPack{stop, "Abort application", "basic"})
	Register("banner", &CliFuncPack{PrintBanner, "Print application banner", "misc"})
}

func echo(ts []string) error {
	str := ""
	for _, t := range ts {
		str += t + " "
	}
	println(str)
	return nil
}

func help(ts []string) error {
	println("===")
	for g, _ := range groups {
		println("[" + g + "]")
		for n, p := range CliFuncs {
			if p.Group == g {
				println("\t" + n + " - " + p.Desc)
			}
		}
	}
	println("===")
	return nil
}

func stop(ts []string) error {
	print("server stopped\t")
	os.Exit(0)
	return nil
}

func PrintBanner(ts []string) error {
	print(Banner)
	println("")
	return nil
}
