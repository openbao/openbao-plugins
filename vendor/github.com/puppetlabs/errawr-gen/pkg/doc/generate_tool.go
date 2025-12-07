// +build tools

package main

import (
	"log"
	"os"

	"github.com/puppetlabs/errawr-gen/pkg/util/fs"
	"github.com/shurcooL/vfsgen"
)

var h = fs.DirWithoutModTimes("assets")

func main() {
	err := vfsgen.Generate(h, vfsgen.Options{
		Filename:    "generate_assets.go",
		PackageName: os.Getenv("GOPACKAGE"),
	})
	if err != nil {
		log.Fatalln(err)
	}
}
