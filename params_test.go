package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
)

// test that various replacements succeed
func TestReplacement(t *testing.T) {
	template, err := ioutil.ReadFile("tests/add-and-gc-template.yml")
	if err != nil {
		t.Fatal(err)
	}
	target, err := ioutil.ReadFile("tests/add-and-gc.yml")
	if err != nil {
		t.Fatal(err)
	}

	params := Params{"NUM_NODES": "1",
		"NUM_TIMES": "10",
		"FILE_SIZE": "10",
		"ON_NODE":   "1",
	}

	processedTemplate, err := replaceParams(template, params)

	if err != nil {
		t.Log(err)
	}

	if bytes.Compare(processedTemplate, target) != 0 {
		t.Fatal("Template with replaced params not equal to target test.")
	}
}

// test that bad replacement fails
func TestBadReplacement(t *testing.T) {
	template, err := ioutil.ReadFile("tests/add-and-gc-template.yml")
	if err != nil {
		t.Fatal(err)
	}
	target, err := ioutil.ReadFile("tests/add-and-gc.yml")
	if err != nil {
		t.Fatal(err)
	}

	params := Params{"NUM_NODES": "1",
		"NUM_TIMES": "10",
		"FILE_SIZE": "10",
		"ON_NODE":   "0",
	}

	processedTemplate, err := replaceParams(template, params)

	if err != nil {
		t.Log(err)
	}

	if bytes.Compare(processedTemplate, target) == 0 {
		t.Fatal("This test should have failed (wrong replacement value).")
	}
}

// test that replacement using config file succeeds
func TestConfigReplacement(t *testing.T) {
	template, err := ioutil.ReadFile("tests/add-and-gc-template.yml")
	if err != nil {
		t.Fatal(err)
	}
	target, err := ioutil.ReadFile("tests/add-and-gc.yml")
	if err != nil {
		t.Fatal(err)
	}

	config, err := loadConfigFile("testutils/add-and-gc-config.yml")
	if err != nil {
		t.Fatal(err)
	}

	processedTemplate, err := replaceParams(template, config.Params)

	if err != nil {
		t.Log(err)
	}

	if bytes.Compare(processedTemplate, target) != 0 {
		t.Fatal("Template with replaced params not equal to target test.")
	}
}

// test that parameter addition works as expected
func TestParamsOverride(t *testing.T) {
	template, err := ioutil.ReadFile("tests/add-and-gc-template.yml")
	if err != nil {
		t.Fatal(err)
	}
	target, err := ioutil.ReadFile("tests/add-and-gc.yml")
	if err != nil {
		t.Fatal(err)
	}

	params1 := Params{"NUM_NODES": "1",
		"NUM_TIMES": "10",
		"ON_NODE":   "0",
	}
	params2 := Params{"NUM_NODES": "1",
		"FILE_SIZE": "10",
		"ON_NODE":   "1",
	}

	config := TestConfig{Params: make(Params)}
	config.addParams(params1)
	config.addParams(params2)

	processedTemplate, err := replaceParams(template, config.Params)

	if err != nil {
		t.Log(err)
	}

	if bytes.Compare(processedTemplate, target) != 0 {
		t.Fatal("Template with replaced params not equal to target test.")
	}
}

// test that missing parameters fail replacement
func TestUndeclaredParams(t *testing.T) {
	template, err := ioutil.ReadFile("tests/add-and-gc-template.yml")
	if err != nil {
		t.Fatal(err)
	}

	params := Params{"NUM_NODES": "1",
		"NUM_TIMES": "10",
		"FILE_SIZE": "10",
	}

	_, err = replaceParams(template, params)

	if err == nil {
		t.Fatal("Param replacement should have failed (missing value).")
	}
	if err.Error() != fmt.Sprintf("Parameter ON_NODE not specified") {
		t.Fatal(fmt.Sprintf("Failed with different error than intended: %s", err))
	}
}
