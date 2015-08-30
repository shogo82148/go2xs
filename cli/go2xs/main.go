package main

import (
	"flag"

	"github.com/shogo82148/go2xs"
)

func main() {
	var name string
	flag.StringVar(&name, "name", "", "library name")
	flag.Parse()
	gen := go2xs.NewGenerator()
	for _, f := range flag.Args() {
		gen.ParseFile(f)
	}
	gen.Generate()
	gen.Output(name)
}
