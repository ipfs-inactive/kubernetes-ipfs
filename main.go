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

// The k8s deployment name; this is what we use to select pods and scale the deployment.
var DEPLOYMENT_NAME = "go-ipfs-stress"

// Summary is
type Summary struct {
	Start      time.Time
	End        time.Time
	Successes  int
	Failures   int
	TestsToRun int
	TestsRan   int
	Timeouts   int
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
	EndNode    int         `yaml:"end_node"`
	CMD        string      `yaml:"cmd"`
	Timeout    int         `yaml:"timeout"`
	Outputs    []Output    `yaml:"outputs"`
	Inputs     []string    `yaml:"inputs"`
	Assertions []Assertion `yaml:"assertions"`
}

// Config is
type Config struct {
	Nodes         int           `yaml:"nodes"`
	Times         int           `yaml:"times"`
	GraceShutdown time.Duration `yaml:"grace_shutdown"`
	Expected      Expected      `yaml:"expected"`
}

// Expected is
type Expected struct {
	Successes int `yaml:"successes"`
	Failures  int `yaml:"failures"`
	Timeouts  int `yaml:"timeouts"`
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
	Status struct {
		Phase string `json:"phase"`
	} `json:"status"`
}

// GetPodsOutput is
type GetPodsOutput struct {
	Items []Pod `json:"items"`
}

func fatal(i interface{}) {
	fmt.Fprintln(os.Stderr, i)
	os.Exit(1)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: ", os.Args[0], "<testfile>")
		os.Exit(1)
	}
	filePath := os.Args[1]
	debug("## Loading " + filePath)

	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		fatal(err)
	}

	var test Test
	var summary Summary

	err = yaml.Unmarshal([]byte(fileData), &test)
	if err != nil {
		fatal(err)
	}

	debug("Configuration:")
	debugSpew(test)

	summary.TestsToRun = test.Config.Times
	summary.Start = time.Now()

	for i := 0; i < test.Config.Times; i++ {
		color.Cyan("## Running test '" + test.Name + "'")
		if err != nil {
			fatal(err)
		}

		// We'll check for running pods.
		// In the event we ask the controller to scale, and the pods are just still starting
		// e.g. If someone cancels the scale-up and restarts right after, then it'll just keep
		// on doing the same thing.
		running_nodes, err := getRunningPods()
		if err != nil {
			fatal(err)
		}
		if test.Config.Nodes > running_nodes {
			fmt.Println("Not enough nodes running. Scaling up...")
			err := scaleTo(test.Config.Nodes)
			if err != nil {
				fatal(err)
			}
		}
		pods, err := getPods() // Get the pod list after a scale-up
		color.Cyan("## Using " + strconv.Itoa(test.Config.Nodes) + " nodes for this test")
		env := make([]string, 0)
		for _, step := range test.Steps {
			color.Blue("### Running step '" + step.Name + "' on node " + strconv.Itoa(step.OnNode))
			if len(step.Inputs) != 0 {
				for _, input := range step.Inputs {
					color.Blue("### Getting variable " + input)
				}
			}
			color.Magenta("$ " + step.CMD)
			// Test whether we want to run this test in parallel
			if step.EndNode != 0 {
				numNodes := step.EndNode - step.OnNode + 1
				color.Magenta("Running parallel on " + strconv.Itoa(numNodes) + " nodes.")
				// Initialize a channel with depth of number of nodes we're testing on simultaneously
				outputStrings := make(chan []string, numNodes)
				outputErr := make(chan bool, numNodes)
				for j := step.OnNode; j < step.EndNode; j++ {
					// Hand this channel to the pod runner and let it fill the queue
					runInPodAsync(pods.Items[step.OnNode-1].Metadata.Name, step.CMD, env, step.Timeout, outputStrings, outputErr)
				}
				// Iterate through the queue to pull out results one-by-one
				// These may be out of order, but is there a better way to do this? Do we need them in order?
				// TODO: Find a way to reduce the duplicated code here.
				for j := step.OnNode; j < step.EndNode; j++ {
					out := <-outputStrings
					err := <-outputErr
					if err {
						summary.Timeouts++
						continue // skip handling the output or other assertions since it timed out.
					}
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
			}
			out, err := runInPod(pods.Items[step.OnNode-1].Metadata.Name, step.CMD, env, step.Timeout)
			if err {
				summary.Timeouts++
				continue
			}
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
	os.Exit(evaluateOutcome(summary, test.Config.Expected)) // Returns success on all tests to OS; this allows for test scripting.
}

func getPods() (*GetPodsOutput, error) {
	// Only return pods that match our deployment.
	cmd := exec.Command("kubectl", "get", "pods", "--output=json", "--selector=run="+DEPLOYMENT_NAME)

	out := new(bytes.Buffer)
	errout := new(bytes.Buffer)
	cmd.Stdout = out
	cmd.Stderr = errout

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("get pods error: %s %s %s", err, errout.String(), out.String())
	}

	pods := new(GetPodsOutput)
	err = json.Unmarshal(out.Bytes(), pods)
	if err != nil {
		return nil, err
	}

	return pods, nil
}

func getRunningPods() (int, error) {
	pods, err := getPods()
	if err != nil {
		return 0, fmt.Errorf("%s\n", err)
	}
	current_number_running := 0
	for _, pod := range pods.Items {
		if pod.Status.Phase == "Running" {
			current_number_running++
		}
	}
	return current_number_running, nil
}

// Scale the k8s deployment to the size required for the tests.
func scaleTo(number int) error {
	fmt.Printf("Scaling in progress...\n")
	cmd := exec.Command("kubectl", "scale", "--replicas="+strconv.Itoa(number), "deployment/"+DEPLOYMENT_NAME)
	errbuf := new(bytes.Buffer)
	cmd.Stderr = errbuf
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf(errbuf.String())
	}
	// Wait until the pods are in "ready" state
	number_running := 0
	for number_running < number {
		number_running, err = getRunningPods()
		if err != nil {
			return err
		}
		fmt.Printf("\tContainers running (current/target): (%d/%d)\n", number_running, number)
		time.Sleep(time.Duration(3) * time.Second)
	}
	fmt.Println("Scale complete")
	return nil
}

func runInPodAsync(name string, cmdToRun string, env []string, timeout int, chanStrings chan []string, chanTimeout chan bool) {
	go func() {
		var lines []string
		defer func() {
			chanStrings <- lines
		}()
		envString := ""
		for _, e := range env {
			envString += e + " "
		}
		if envString != "" {
			envString = envString + "&& "
		}
		cmd := exec.Command("kubectl", "exec", name, "--", "bash", "-c", envString+cmdToRun)
		var out bytes.Buffer
		var errout bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &errout
		cmd.Start()
		timeout_reached := false

		// Handle timeouts
		if timeout != 0 {
			timer := time.AfterFunc(time.Duration(timeout)*time.Second, func() {
				cmd.Process.Kill()
				timeout_reached = true
				color.Set(color.FgRed)
				fmt.Println("Command timed out after", timeout, "seconds")
				color.Unset()
			})
			cmd.Wait()
			timer.Stop()
		} else {
			cmd.Wait()
		}

		if errout.String() != "" {
			fmt.Println(errout.String())
		}
		lines = strings.Split(out.String(), "\n")
		// Feed our output into the channel.
		chanStrings <- lines[:len(lines)-1]
		chanTimeout <- timeout_reached
	}()
}

func runInPod(name string, cmdToRun string, env []string, timeout int) ([]string, bool) {
	envString := ""
	for _, e := range env {
		envString += e + " "
	}
	if envString != "" {
		envString = envString + "&& "
	}
	cmd := exec.Command("kubectl", "exec", name, "--", "bash", "-c", envString+cmdToRun)
	var out bytes.Buffer
	var errout bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errout
	cmd.Start()
	timeout_reached := false

	// Handle timeouts
	if timeout != 0 {
		timer := time.AfterFunc(time.Duration(timeout)*time.Second, func() {
			cmd.Process.Kill()
			timeout_reached = true
			color.Set(color.FgRed)
			fmt.Println("Command timed out after", timeout, "seconds")
			color.Unset()
		})
		cmd.Wait()
		timer.Stop()
	} else {
		cmd.Wait()
	}

	if errout.String() != "" {
		fmt.Println(errout.String())
	}
	lines := strings.Split(out.String(), "\n")
	return lines[:len(lines)-1], timeout_reached
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
	timeouts := strconv.Itoa(summary.Timeouts)
	fmt.Println("== Successes: " + successes + "/" + failures + " (success/failure)")
	fmt.Println("== Timeouts: " + timeouts)

	// Get the grafana service dynamically; this will work even for real k8s deployments instead of just minikube
	var port_out bytes.Buffer
	port_cmd := exec.Command("kubectl", "get", "service", "grafana", "--namespace=monitoring", "-o", "jsonpath='{.spec.ports[0].nodePort}'")
	port_cmd.Stdout = &port_out
	port_cmd.Run()
	// Ignore this error for now... We handle it in address_cmd

	var address_out bytes.Buffer
	address_cmd := exec.Command("kubectl", "get", "nodes", "-o", "jsonpath='{.items[0].status.addresses[?(@.type == \"InternalIP\")].address}'")
	address_cmd.Stdout = &address_out
	if address_cmd.Run() != nil {
		// Use fallback address, we weren't able to get the
		address := strings.Replace(address_out.String(), "'", "", -1)
		port := strings.Replace(port_out.String(), "'", "", -1)

		metricsLink := fmt.Sprintf("http://%s:%s", address, port)
		metricsLink += "/dashboard/db/kubernetes-pod-resources?from=" + unixToStr(summary.Start.Unix()) + "&to=" + unixToStr(summary.End.Unix())
		fmt.Println("==")
		fmt.Println("== Metrics: " + metricsLink)
	}
}

func evaluateOutcome(summary Summary, expected Expected) int {
	if summary.Successes != expected.Successes || summary.Failures != expected.Failures || summary.Timeouts != expected.Timeouts {
		color.Set(color.FgRed)
		fmt.Println("Expectations were not met")
		color.Unset()
		return 1
	}

	color.Set(color.FgGreen)
	fmt.Println("Expectations were met")
	color.Unset()
	return 0
}

func unixToStr(i int64) string {
	return strconv.FormatInt(i, 10) + "000"
}
