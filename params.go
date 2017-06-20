package main

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// Params holds the parameter mappings specified  by user
type Params map[string]string

var pNamePattern = `([A-Za-z_][0-9A-Za-z_]*)`

var pNameRegex = regexp.MustCompile(`^` + pNamePattern + `$`)
var paramDecRegex = regexp.MustCompile(`{{ *` + pNamePattern + ` *}}`)
var paramRegex = regexp.MustCompile(pNamePattern + `=(.+?)`)

// specific Params -> String transformation
func (params *Params) String() string {
	return fmt.Sprint(*params)
}

// tell `flag` package how to parse/store param args
func (params *Params) Set(input string) error {
	// for each `--param` flag...
	for _, p := range strings.Split(input, ",") {
		// split params on ,
		pElems := strings.SplitN(p, "=", 2)
		// check number of elements in param arg
		switch len(pElems) {
		case 0:
			return fmt.Errorf("Missing argument to `--param` flag.")
		case 1:
			return fmt.Errorf("Missing value in parameter arg: %s", p)
		}
		// validate string as PName identifer
		pName, pValue := pElems[0], pElems[1]

		(*params)[pName] = pValue
	}
	return nil
}

/* resolve all param names (keys in `params` map) to their resp. values */
func replaceParams(fileData []byte, params Params) ([]byte, error) {
	// get all unique param declarations from test file
	matches := uniqueParamDeclarations(fileData)
	for pDeclaration, pName := range matches {
		// check whether the reference parameter is defined
		if pVal, ok := params[pName]; ok {
			// parameter found, replace all occurrences with its value
			fileData = bytes.Replace(fileData, []byte(pDeclaration), []byte(pVal), -1)
		} else {
			// parameter not found, fail
			return []byte{}, fmt.Errorf("Parameter %s not specified", pName)
		}
	}
	return fileData, nil
}

/* returns all unique parameter declarations in a test file */
func uniqueParamDeclarations(fileData []byte) map[string]string {
	// get all regex results
	regexResults := paramDecRegex.FindAllSubmatch(fileData, -1)
	// matchSet is map from '{{ <varName> }}' to '<varName>'
	matchSet := make(map[string]string)
	for _, result := range regexResults {
		matchSet[string(result[0])] = string(result[1])
	}
	return matchSet
}
