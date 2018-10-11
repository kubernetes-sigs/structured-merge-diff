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

package framework

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"

	"sigs.k8s.io/structured-merge-diff/schema"
	"sigs.k8s.io/structured-merge-diff/typed"
	"sigs.k8s.io/structured-merge-diff/value"
)

// YAMLObject is an object encoded in YAML.
type YAMLObject string

// YAMLParser allows you to parse YAML into a TypeValue
type YAMLParser interface {
	FromYAML(object YAMLObject, typename string) (typed.TypedValue, error)
	FromYAMLOrDie(object YAMLObject, typename string) typed.TypedValue
}

type parser struct {
	schema schema.Schema
}

func parseSchema(object YAMLObject) (schema.Schema, error) {
	// Make sure the schema validates against the schema schema.
	var s schema.Schema
	err := yaml.Unmarshal([]byte(object), &s)
	return s, err
}

// NewParser will build a YAMLParser with a corresponding version and schema.
func NewParser(object YAMLObject) (YAMLParser, error) {
	ss, err := parseSchema(YAMLObject(schema.SchemaSchemaYAML))
	if err != nil {
		return nil, fmt.Errorf("unable to parse SchemaSchema: %v", err)
	}

	schemaParser := parser{
		schema: ss,
	}
	_, err = schemaParser.FromYAML(object, "schema")
	if err != nil {
		return nil, fmt.Errorf("unable to validate schema: %v", err)
	}

	s, err := parseSchema(object)
	if err != nil {
		return nil, fmt.Errorf("unable to parse schema: %v", err)
	}
	return &parser{schema: s}, nil
}

// NewParserOrDie either returns a YAMLParser or dies.
func NewParserOrDie(schema YAMLObject) YAMLParser {
	p, err := NewParser(schema)
	if err != nil {
		panic(fmt.Errorf("Failed to create parser: %v", err))
	}
	return p
}

func (p *parser) FromYAML(object YAMLObject, typename string) (typed.TypedValue, error) {
	v, err := value.FromYAML([]byte(object))
	if err != nil {
		return typed.TypedValue{}, err
	}
	return typed.AsTyped(v, &p.schema, typename)
}

func (p *parser) FromYAMLOrDie(object YAMLObject, typename string) typed.TypedValue {
	o, err := p.FromYAML(object, typename)
	if err != nil {
		panic(fmt.Errorf("Failed to parse YAML object: %v", err))
	}
	return o
}
