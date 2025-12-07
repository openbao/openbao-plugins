package doc

import (
	"fmt"

	"github.com/puppetlabs/errawr-gen/pkg/errors"
	"github.com/xeipuuv/gojsonschema"
	yaml "gopkg.in/yaml.v2"
)

type DocumentVersionFragment struct {
	Version string `yaml:"version" json:"version"`
}

type DocumentDomain struct {
	Key   string `yaml:"key" json:"key"`
	Title string `yaml:"title" json:"title"`
}

type DocumentErrorDescriptionFragment struct {
	Friendly  string `yaml:"friendly" json:"friendly"`
	Technical string `yaml:"technical" json:"technical"`
}

type DocumentErrorDescription DocumentErrorDescriptionFragment

func (dd *DocumentErrorDescription) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var f DocumentErrorDescriptionFragment
	if err := unmarshal(&f); err != nil {
		var s string
		if err := unmarshal(&s); err != nil {
			return err
		}

		*dd = DocumentErrorDescription{
			Friendly:  s,
			Technical: s,
		}
	} else {
		*dd = DocumentErrorDescription(f)
	}

	return nil
}

type DocumentErrorArgumentFragment struct {
	Type        string      `yaml:"type" json:"type"`
	Description string      `yaml:"description" json:"description"`
	Validators  []string    `yaml:"validators" json:"validators"`
	Default     interface{} `yaml:"default" json:"default"`
}

type DocumentErrorArgument DocumentErrorArgumentFragment

func (dea *DocumentErrorArgument) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var f DocumentErrorArgumentFragment
	if err := unmarshal(&f); err != nil {
		return err
	}

	*dea = DocumentErrorArgument(f)

	if dea.Validators == nil {
		dea.Validators = []string{}
	}

	if dea.Type == "" {
		dea.Type = "string"
	}

	return nil
}

func (dea *DocumentErrorArgument) IsOptional() bool {
	return dea != nil && dea.Default != nil
}

type DocumentErrorArgumentItem struct {
	Name     string
	Argument *DocumentErrorArgument
}

type DocumentErrorArguments map[string]*DocumentErrorArgument

type DocumentErrorHTTPMetadataHeader []string

func (dehmh *DocumentErrorHTTPMetadataHeader) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var f []string
	if err := unmarshal(&f); err != nil {
		var s string
		if err := unmarshal(&s); err != nil {
			return err
		}

		*dehmh = []string{s}
	} else {
		*dehmh = f
	}

	return nil
}

type DocumentErrorHTTPMetadataHeaderItem struct {
	Name   string
	Values DocumentErrorHTTPMetadataHeader
}

type DocumentErrorHTTPMetadataHeaders map[string]DocumentErrorHTTPMetadataHeader

type DocumentErrorHTTPMetadataFragment struct {
	Status         int                                   `json:"status"`
	Headers        DocumentErrorHTTPMetadataHeaders      `json:"headers,omitempty"`
	OrderedHeaders []DocumentErrorHTTPMetadataHeaderItem `json:"-"`
}

type DocumentErrorHTTPMetadata DocumentErrorHTTPMetadataFragment

func (dehm *DocumentErrorHTTPMetadata) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var f DocumentErrorHTTPMetadataFragment
	if err := unmarshal(&f); err != nil {
		return err
	}

	*dehm = DocumentErrorHTTPMetadata(f)

	if dehm.Headers == nil {
		dehm.Headers = DocumentErrorHTTPMetadataHeaders{}
	}

	var fi struct {
		Status    int           `yaml:"status"`
		HeadersIt yaml.MapSlice `yaml:"headers"`
	}
	if err := unmarshal(&fi); err != nil {
		return err
	}

	for _, item := range fi.HeadersIt {
		name := item.Key.(string)

		dehm.OrderedHeaders = append(dehm.OrderedHeaders, DocumentErrorHTTPMetadataHeaderItem{
			Name:   name,
			Values: dehm.Headers[name],
		})
	}

	return nil
}

type DocumentErrorMetadata struct {
	HTTP *DocumentErrorHTTPMetadata `yaml:"http" json:"http"`
}

type DocumentErrorFragment struct {
	Title            string                      `json:"title"`
	Traits           []string                    `json:"traits"`
	Sensitivity      string                      `json:"sensitivity,omitempty"`
	Description      *DocumentErrorDescription   `json:"description"`
	Arguments        DocumentErrorArguments      `json:"arguments"`
	OrderedArguments []DocumentErrorArgumentItem `json:"-"`
	Metadata         DocumentErrorMetadata       `json:"metadata"`
}

type DocumentError DocumentErrorFragment

func (de *DocumentError) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var f DocumentErrorFragment
	if err := unmarshal(&f); err != nil {
		return err
	}

	*de = DocumentError(f)

	if de.Traits == nil {
		de.Traits = []string{}
	}

	if de.Arguments == nil {
		de.Arguments = DocumentErrorArguments{}
	}

	// Extract the iteration order.
	var fi struct {
		Title       string                    `yaml:"title"`
		Traits      []string                  `yaml:"traits"`
		Sensitivity string                    `yaml:"sensitivity"`
		Description *DocumentErrorDescription `yaml:"description"`
		ArgumentsIt yaml.MapSlice             `yaml:"arguments"`
		Metadata    *DocumentErrorMetadata    `yaml:"metadata"`
	}
	if err := unmarshal(&fi); err != nil {
		return err
	}

	for _, item := range fi.ArgumentsIt {
		name := item.Key.(string)

		de.OrderedArguments = append(de.OrderedArguments, DocumentErrorArgumentItem{
			Name:     name,
			Argument: de.Arguments[name],
		})
	}

	return nil
}

type DocumentErrors map[string]DocumentError

type DocumentSection struct {
	Title  string         `yaml:"title" json:"title"`
	Errors DocumentErrors `yaml:"errors" json:"errors"`
}

type DocumentSections map[string]DocumentSection

type Document struct {
	Version  int              `yaml:"version" json:"version"`
	Domain   DocumentDomain   `yaml:"domain" json:"domain"`
	Sections DocumentSections `yaml:"sections" json:"sections"`
}

func New(data string) (*Document, error) {
	// Pull out the document version.
	var version DocumentVersionFragment
	if err := yaml.Unmarshal([]byte(data), &version); err != nil {
		return nil, errors.NewDocNoVersionError().WithCause(err)
	}

	if version.Version != "1" {
		return nil, errors.NewDocUnknownVersionError(version.Version, []string{"1"})
	}

	var document Document
	if err := yaml.UnmarshalStrict([]byte(data), &document); err != nil {
		return nil, errors.NewDocParseError().WithCause(err)
	}

	result, err := schema.Validate(gojsonschema.NewGoLoader(document))
	if err != nil {
		return nil, errors.NewDocValidationError().WithCause(err)
	} else if !result.Valid() {
		errs := make([]string, len(result.Errors()))
		for i, err := range result.Errors() {
			errs[i] = fmt.Sprintf("%s", err)
		}

		return nil, errors.NewDocValidationErrorBuilder().WithErrors(errs).Build()
	}

	return &document, nil
}
