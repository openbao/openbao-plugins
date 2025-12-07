package generator

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/puppetlabs/errawr-gen/pkg/doc"
	"github.com/puppetlabs/errawr-gen/pkg/lang/golang"
)

type Generator interface {
	Generate(pkg string, document *doc.Document, output io.Writer) error
}

type Language string

var (
	LanguageGo Language = "go"
)

type Config struct {
	Package        string
	OutputPath     string
	OutputLanguage Language
	InputPath      string
}

func Generate(conf Config) error {
	if len(conf.Package) == 0 {
		if pkg := os.Getenv("GOPACKAGE"); len(pkg) != 0 {
			conf.Package = pkg
		} else {
			return fmt.Errorf("package name could not be determined; specify one")
		}
	}

	var generator Generator

	switch conf.OutputLanguage {
	case LanguageGo, "":
		generator = golang.NewGenerator()
	default:
		return fmt.Errorf("language %q is not supported", conf.OutputLanguage)
	}

	var input, output *os.File
	var err error

	if len(conf.InputPath) > 0 && conf.InputPath != "-" {
		input, err = os.Open(conf.InputPath)
		if err != nil {
			return fmt.Errorf("could not open input file: %+v", err)
		}
		defer input.Close()
	} else {
		input = os.Stdin
	}

	y, err := ioutil.ReadAll(input)
	if err != nil {
		return fmt.Errorf("could not read file: %+v", err)
	}

	document, err := doc.New(string(y))
	if err != nil {
		return err
	}

	if len(conf.OutputPath) > 0 && conf.OutputPath != "-" {
		output, err = os.Create(conf.OutputPath)
		if err != nil {
			return fmt.Errorf("could not open output file: %+v", err)
		}
		defer output.Close()
	} else {
		output = os.Stdout
	}

	if err := generator.Generate(conf.Package, document, output); err != nil {
		return fmt.Errorf("failed to generate Go file: %+v", err)
	}

	return nil
}
