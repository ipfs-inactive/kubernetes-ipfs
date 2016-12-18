package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/color"

	yaml "gopkg.in/yaml.v2"
)

// DEBUG decides if we should have debug output enabled or not
var DEBUG = false

// Summary is
type Summary struct {
	Start      time.Time
	End        time.Time
	Successes  int
	Failures   int
	TestsToRun int
	TestsRan   int
}

// Output is
type Output struct {
	Line   int    `yaml:"line"`
	SaveTo string `yaml:"save_to"`
}

// Assertion is
type Assertion struct {
	Line            int    `yaml:"line"`
	ShouldBeEqualTo string `yaml:"should_be_equal_to"`
}

// Step is
type Step struct {
	Name       string      `yaml:"name"`
	OnNode     int         `yaml:"on_node"`
	CMD        string      `yaml:"cmd"`
	Outputs    []Output    `yaml:"outputs"`
	Inputs     []string    `yaml:"inputs"`
	Assertions []Assertion `yaml:"assertions"`
}

// Config is
type Config struct {
	Nodes         int           `yaml:"nodes"`
	Times         int           `yaml:"times"`
	GraceShutdown time.Duration `yaml:"grace_shutdown"`
}

// Test is
type Test struct {
	Name   string `yaml:"name"`
	Config Config `yaml:"config"`
	Steps  []Step `yaml:"steps"`
}

// Pod is
type Pod struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
}

// GetPodsOutput is
type GetPodsOutput struct {
	Items []Pod `json:"items"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: kubernetes-ipfs <testfile>")
		os.Exit(1)
	}
	filePath := os.Args[1]
	debug("## Loading " + filePath)
	fileData, err := ioutil.ReadFile(filePath)
	handleErr(err)
	test := Test{}
	summary := Summary{Successes: 0, Failures: 0, TestsRan: 0}
	err = yaml.Unmarshal([]byte(fileData), &test)
	debug("Configuration:")
	debugSpew(test)
	handleErr(err)
	summary.TestsToRun = test.Config.Times
	summary.Start = time.Now()
	for i := 0; i < test.Config.Times; i++ {
		color.Cyan("## Running test '" + test.Name + "'")
		color.Cyan("## Starting " + strconv.Itoa(test.Config.Nodes) + " nodes for this test")
		pods := getPods()
		debug("First pod: " + pods.Items[0].Metadata.Name)
		debug("Second pod: " + pods.Items[1].Metadata.Name)
		env := make([]string, 0)
		for _, step := range test.Steps {
			color.Blue("### Running step '" + step.Name + "' on node " + strconv.Itoa(step.OnNode))
			if len(step.Inputs) != 0 {
				for _, input := range step.Inputs {
					color.Blue("### Getting variable " + input)
				}
			}
			color.Magenta("$ " + step.CMD)
			out := runInPod(pods.Items[step.OnNode-1].Metadata.Name, step.CMD, env)
			if len(step.Outputs) != 0 {
				for index, output := range step.Outputs {
					color.Magenta("### Saving output from line " + strconv.Itoa(output.Line) + " to variable " + output.SaveTo)
					line := out[index]
					env = append(env, output.SaveTo+"="+line)
				}
			}
			if len(step.Assertions) != 0 {
				for _, assertion := range step.Assertions {
					lineToAssert := out[assertion.Line]
					value := ""
					for _, e := range env {
						if strings.Contains(e, assertion.ShouldBeEqualTo) {
							value = e[len(assertion.ShouldBeEqualTo)+1:]
							break
						}
					}
					if lineToAssert != value {
						color.Set(color.FgRed)
						fmt.Println("Assertion failed!")
						fmt.Println("Actual value=" + value)
						fmt.Println("Expected value=" + lineToAssert)
						color.Unset()
						summary.Failures = summary.Failures + 1
					} else {
						summary.Successes = summary.Successes + 1
						color.Green("Assertion Passed")
					}
					fmt.Println()
				}
			}
		}
		summary.TestsRan = summary.TestsRan + 1
	}
	fmt.Println(time.Now().String())
	fmt.Println("Now waiting for " + test.Config.GraceShutdown.String() + " seconds before shutdown...")
	time.Sleep(test.Config.GraceShutdown * time.Second)
	summary.End = time.Now()
	printSummary(summary)
}

func getPods() GetPodsOutput {
	cmd := exec.Command("kubectl", "get", "pods", "--output=json")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	handleErr(err)
	pods := GetPodsOutput{}
	err = json.Unmarshal(out.Bytes(), &pods)
	handleErr(err)
	return pods
}

func runInPod(name string, cmdToRun string, env []string) []string {
	envString := ""
	for _, e := range env {
		envString = e + " "
	}
	if envString != "" {
		envString = envString + "&& "
	}
	cmd := exec.Command("kubectl", "exec", name, "--", "bash", "-c", envString+cmdToRun)
	var out bytes.Buffer
	var errout bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errout
	err := cmd.Run()
	if errout.String() != "" {
		fmt.Println(errout.String())
	}
	handleErr(err)
	lines := strings.Split(out.String(), "\n")
	return lines[:len(lines)-1]
}

func handleErr(err interface{}) {
	if err != nil {
		panic(err)
	}
}

func debug(str string) {
	debugEnvVar := os.Getenv("DEBUG")
	if debugEnvVar != "" {
		fmt.Println(str)
	}
}

func debugSpew(thing interface{}) {
	debugEnvVar := os.Getenv("DEBUG")
	if debugEnvVar != "" {
		spew.Dump(thing)
	}
}

func printSummary(summary Summary) {
	fmt.Println("============================")
	fmt.Println("== Test Summary")
	fmt.Println("===============")
	fmt.Println("==")
	fmt.Println("== Started: " + summary.Start.String())
	fmt.Println("== Ended: " + summary.End.String())
	fmt.Println("==")
	successes := strconv.Itoa(summary.Successes)
	failures := strconv.Itoa(summary.Failures)
	fmt.Println("== Successes: " + successes + "/" + failures + " (success/failure)")
	// ?from=1482079531666&to=1482079674457
	metricsLink := "http://192.168.99.101:30594/dashboard/db/kubernetes-pod-resources?from=" + unixToStr(summary.Start.Unix()) + "&to=" + unixToStr(summary.End.Unix())
	fmt.Println("==")
	fmt.Println("== Metrics: " + metricsLink)
}

func unixToStr(i int64) string {
	return strconv.FormatInt(i, 10) + "000"
}
