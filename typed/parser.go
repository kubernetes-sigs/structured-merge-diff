/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package typed

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"
	"sigs.k8s.io/structured-merge-diff/schema"
	"sigs.k8s.io/structured-merge-diff/value"
)

// YAMLObject is an object encoded in YAML.
type YAMLObject string

// Parser implements YAMLParser and allows introspecting the schema.
type Parser struct {
	Schema schema.Schema
}

// create builds an unvalidated parser.
func create(schema YAMLObject) (*Parser, error) {
	p := Parser{}
	err := yaml.Unmarshal([]byte(schema), &p.Schema)
	return &p, err
}

func createOrDie(schema YAMLObject) *Parser {
	p, err := create(schema)
	if err != nil {
		panic(fmt.Errorf("failed to create parser: %v", err))
	}
	return p
}

var ssParser = createOrDie(YAMLObject(schema.SchemaSchemaYAML))

// NewParser will build a YAMLParser from a schema. The schema is validated.
func NewParser(schema YAMLObject) (*Parser, error) {
	_, err := ssParser.FromYAML(schema, "schema")
	if err != nil {
		return nil, fmt.Errorf("unable to validate schema: %v", err)
	}
	return create(schema)
}

func (p *Parser) NewEmpty(typename string) (TypedValue, error) {
	return p.FromYAML(YAMLObject("{}"), typename)
}

func (p *Parser) FromYAML(object YAMLObject, typename string) (TypedValue, error) {
	v, err := value.FromYAML([]byte(object))
	if err != nil {
		return TypedValue{}, err
	}
	return AsTyped(v, &p.Schema, typename)
}
