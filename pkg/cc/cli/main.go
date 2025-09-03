package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/kpango/glg"
)

type (
	CliFunc     func([]string) error
	CliFuncPack struct {
		F     CliFunc
		Desc  string
		Group string
	}
)

var (
	CliFuncs map[string]*CliFuncPack
	groups   map[string]bool
	ir       *bufio.Reader
)

func init() {
	CliFuncs = make(map[string]*CliFuncPack)
	groups = make(map[string]bool)
	ir = bufio.NewReader(os.Stdin)
	initClis()
}

func proc() {
	defer func() {
		if r := recover(); r != nil {
			glg.Error(r)
		}
	}()
	print("cyf-cloud> ")
	input, e := ir.ReadString('\n')

	for input[0:1] == " " {
		input = strings.TrimPrefix(input, " ")
	}
	args := strings.Split(input, " ")

	args[len(args)-1] = strings.TrimRight(args[len(args)-1], "\n")
	args[len(args)-1] = strings.TrimRight(args[len(args)-1], "\r")

	f := CliFuncs[args[0]]
	// 是否有该命令
	if f == nil {
		fmt.Println("command \"" + args[0] + "\" does not exist")
		return
	}

	// 移除首个元素
	args = append(args[:0], args[1:]...)
	e = f.F(args)
	if e != nil {
		panic(e)
	}
}

func Run() {
	for {
		proc()
	}
}

func Register(name string, f *CliFuncPack) {
	if CliFuncs[name] != nil {
		glg.Warn("cli: " + name + " overwrote")
	}
	groups[f.Group] = true
	CliFuncs[name] = f
}
