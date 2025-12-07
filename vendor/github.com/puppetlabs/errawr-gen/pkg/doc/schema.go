package doc

import (
	"io/ioutil"

	"github.com/xeipuuv/gojsonschema"
)

var schema *gojsonschema.Schema

func init() {
	f, err := assets.Open("schemas/v1/errors.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}

	schema, err = gojsonschema.NewSchema(gojsonschema.NewBytesLoader(b))
	if err != nil {
		panic(err)
	}
}
