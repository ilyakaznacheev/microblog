package main

import (
	"io/ioutil"

	"github.com/graph-gophers/graphql-go"
)

func getSchema(filename string, rh *repoHandler) (*graphql.Schema, error) {
	// todo: add to config file
	schemaFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	schemaRaw := string(schemaFile)

	return graphql.MustParseSchema(schemaRaw, newResolver(rh)), nil
}
