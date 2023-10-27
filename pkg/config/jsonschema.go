/*
Copyright © 2020-2023 The k3d Author(s)

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/xeipuuv/gojsonschema"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
)

// ValidateSchemaFile takes a filepath, reads the file and validates it against a JSON schema
func ValidateSchemaFile(filepath string, schema []byte) error {
	l.Log().Debugf("Validating file %s against default JSONSchema...", filepath)

	fileContents, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("Failed to read file %s: %+v", filepath, err)
	}

	var content map[string]interface{}
	if err := yaml.Unmarshal(fileContents, &content); err != nil {
		return fmt.Errorf("Failed to unmarshal the content of %s to a map: %+v", filepath, err)
	}

	return ValidateSchema(content, schema)
}

// ValidateSchema validates a YAML construct (non-struct representation) against a JSON Schema
func ValidateSchema(content interface{}, schemaJSON []byte) error {
	contentYaml, err := yaml.Marshal(content)
	if err != nil {
		return err
	}
	contentJSON, err := yaml.YAMLToJSON(contentYaml)
	if err != nil {
		return err
	}

	return ValidateSchemaJSON(contentJSON, schemaJSON)
}

func ValidateSchemaJSON(contentJSON []byte, schemaJSON []byte) error {
	if bytes.Equal(contentJSON, []byte("null")) {
		contentJSON = []byte("{}") // non-json yaml struct
	}

	configLoader := gojsonschema.NewBytesLoader(contentJSON)
	schemaLoader := gojsonschema.NewBytesLoader(schemaJSON)

	result, err := gojsonschema.Validate(schemaLoader, configLoader)
	if err != nil {
		return fmt.Errorf("failed to validate config: %w", err)
	}

	l.Log().Debugf("JSON Schema Validation Result: %+v", result)

	if !result.Valid() {
		var sb strings.Builder
		for _, desc := range result.Errors() {
			sb.WriteString(fmt.Sprintf("- %s\n", desc))
		}
		return errors.New(sb.String())
	}

	return nil
}
