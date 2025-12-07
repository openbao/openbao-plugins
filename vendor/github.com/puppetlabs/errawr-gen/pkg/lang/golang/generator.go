package golang

import (
	"io"

	"github.com/puppetlabs/errawr-gen/pkg/doc"
)

type Generator struct{}

func (g *Generator) Generate(pkg string, document *doc.Document, output io.Writer) error {
	return NewFileGenerator(pkg, document).Render(output)
}

func NewGenerator() *Generator {
	return &Generator{}
}
