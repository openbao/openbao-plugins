// +build tools

package main

import (
	"log"

	"github.com/puppetlabs/errawr-gen/pkg/generator"
)

func main() {
	err := generator.Generate(generator.Config{
		InputPath:  "errors.yml",
		OutputPath: "generate_errors.go",
	})
	if err != nil {
		log.Fatalln(err)
	}
}
