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

// Package main implements a command line tool for performing structured
// operations on yaml files.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"sigs.k8s.io/structured-merge-diff/typed"
)

type options struct {
	schemaPath string
	typeName   string

	validatePath string
}

func (o *options) addFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.schemaPath, "schema", "", "Path to the schema file for this operation. Required.")
	fs.StringVar(&o.typeName, "type-name", "", "Name of type in the schema to use. If empty, the first type in the schema will be used.")
	fs.StringVar(&o.validatePath, "validate", "", "Path to a file to validate against the schema.")
}

type operation interface {
	execute() error
}

type operationBase struct {
	parser   *typed.Parser
	typeName string
}

// resolve turns options in to an operation that can be executed.
func (o *options) resolve() (operation, error) {
	var base operationBase
	if o.schemaPath == "" {
		return nil, errors.New("a schema is required")
	}
	b, err := ioutil.ReadFile(o.schemaPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read schema %q: %v", o.schemaPath, err)
	}
	base.parser, err = typed.NewParser(typed.YAMLObject(b))
	if err != nil {
		return nil, fmt.Errorf("schema %q has errors:\n%v", o.schemaPath, err)
	}

	if o.typeName == "" {
		types := base.parser.Schema.Types
		if len(types) == 0 {
			return nil, errors.New("no types were given in the schema")
		}
		base.typeName = types[0].Name
	} else {
		base.typeName = o.typeName
	}

	switch {
	case o.validatePath != "":
		return validation{base, o.validatePath}, nil
	}
	return nil, errors.New("no operation requested")
}

type validation struct {
	operationBase

	fileToValidate string
}

func (v validation) execute() error {
	b, err := ioutil.ReadFile(v.fileToValidate)
	if err != nil {
		return fmt.Errorf("unable to read file %q: %v", v.fileToValidate, err)
	}
	_, err = v.parser.FromYAML(typed.YAMLObject(b), v.typeName)
	if err != nil {
		return fmt.Errorf("unable to validate file %q:\n%v", v.fileToValidate, err)
	}
	return nil
}

func main() {
	var o options
	o.addFlags(flag.CommandLine)
	flag.Parse()

	op, err := o.resolve()
	if err != nil {
		log.Fatalf("Couldn't resolve options: %v", err)
	}

	err = op.execute()
	if err != nil {
		log.Fatalf("Couldn't execute operation: %v", err)
	}
}
