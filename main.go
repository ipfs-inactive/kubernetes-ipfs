package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
	"math/rand"

	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/color"

	yaml "gopkg.in/yaml.v2"
)

// DEBUG decides if we should have debug output enabled or not
var DEBUG = false

var DEPLOYMENT_NAME = "go-ipfs-stress"

const (
  RANDOM     = "RANDOM"
  SEQUENTIAL = "SEQUENTIAL"
  EVEN       = "EVEN"
  WEIGHTED   = "WEIGHTED"
) 

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
	Name        string      `yaml:"name"`

	/* Old style selection remains supported */
	OnNode      int         `yaml:"on_node"`
	EndNode     int         `yaml:"end_node"`
	/* New style selection */
	Selection   Selection   `yaml:"selection"`

	CMD         string      `yaml:"cmd"`
	Timeout     int         `yaml:"timeout"`
	Outputs     []Output    `yaml:"outputs"`
	Inputs      []string    `yaml:"inputs"`
	Assertions  []Assertion `yaml:"assertions"`
	WriteToFile string      `yaml:"write_to_file"`
}

// Selection is
type Selection struct {
  Range   Range   `yaml:range`
  Percent Percent `yaml:percent`
  Subset  Subset  `yaml:subset`
}

// Range is
type Range struct {
  Order  int `yaml:order`  /* RANDOM or SEQUENTIAL */
  Start  int `yaml:start`  /* Valid for SEQUENTIAL */
  End    int `yaml:end`    /* Valid for SEQUENTIAL */
  Number int `yaml:number` /* Valid for RANDOM */  
}

// Percent is
type Percent struct {
  Order   int `yaml:order`  /* RANDOM or SEQUENTIAL */
  Start   int `yaml:start`  /* Valid for SEQUENTIAL */
  End     int `yaml:end`    /* Valid for SEQUENTIAL */
  Percent int `yaml:number` /* Valid for RANDOM */  
}

// Subset is 
type Subset struct {
  Indices []int `yaml:indices`
}

// SubsetOld is
type SubsetOld struct {
	Numbers []int `yaml:"numbers"`
	Total   int   `yaml:"total"`
}

// Config is
type Config struct {
	Nodes           int             `yaml:"nodes"`
	Selector        string          `yaml:"selector"`
	Times           int             `yaml:"times"`
	GraceShutdown   time.Duration   `yaml:"grace_shutdown"`
	Expected        Expected        `yaml:"expected"`
	SubsetPartition SubsetPartition `yaml:subset_partition`
}

//SubsetParition is
type SubsetPartition struct {
  PartitionType    int   `yaml:partion_type`      /* Either EVEN or WEIGHTED */
  Order            int   `yaml:order`             /* Either RANDOM or SEQUENTIAL */
  Percents         []int `yaml:percents`          /* Valid for WEIGHTED */
  NumberPartitions int   `yaml:number_partitions` /* Valid for EVEN */
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
	var subsetPartition map[int][]int

	err = yaml.Unmarshal([]byte(fileData), &test)
	if err != nil {
		fatal(err)
	}

	debug("Configuration:")
	debugSpew(test)

	/* Include call to partition nodes into subsets if the partition field is included in the 
	   config.  subsetPartion is nil if it is not included in config.  Tests must include this 
	   in the config in order to use the subset selection method to choose nodes later on during
	   testing  */
	subsetPartition = partition(test)

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
		running_nodes, err := getRunningPods(&test.Config)
		if err != nil {
			fatal(err)
		}
		if test.Config.Nodes > running_nodes {
			fmt.Println("Not enough nodes running. Scaling up...")
			err := scaleTo(&test.Config)
			if err != nil {
				fatal(err)
			}
		}

		pods, err := getPods(&test.Config) // Get the pod list after a scale-up
		color.Cyan("## Using " + strconv.Itoa(test.Config.Nodes) + " nodes for this test")
		env := make([]string, 0)
		for _, step := range test.Steps {
			nodeIndices := selectNodes(&step)
			env = handleStep(*pods, &step, &summary, env, nodeIndices)
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

func getSubsetBounds(subset int, numSubsets int, numNodes int) (int, int) {
	var offset1 int
	if (((subset - 1) * numNodes) % numSubsets) > 0 {
		offset1 = 1
	} else {
		offset1 = 0
	}
	startNode := 1 + (subset-1)*numNodes/numSubsets + offset1

	var offset2 int
	if ((subset * numNodes) % numSubsets) > 0 {
		offset2 = 1
	} else {
		offset2 = 0
	}
	endNode := subset*numNodes/numSubsets + offset2

	return startNode, endNode
}

func handleStep(pods GetPodsOutput, step *Step, summary *Summary, env []string, nodeIndices []int) []string {
	color.Blue("### Running step %s on nodes %d to %d", step.Name, step.OnNode, step.EndNode)
	if len(step.Inputs) != 0 {
		for _, input := range step.Inputs {
			color.Blue("### Getting variable " + input)
		}
	}
	color.Magenta("$ %s", step.CMD)
	numNodes := len(nodeIndices)
	color.Magenta("Running parallel on %d nodes.", numNodes)

	// Initialize a channel with depth of number of nodes we're testing on simultaneously
	outputStrings := make(chan []string, numNodes)
	outputErr := make(chan bool, numNodes)
	for j := step.OnNode; j <= endNode; j++ {
		// Hand this channel to the pod runner and let it fill the queue
		runInPodAsync(pods.Items[step.OnNode-1].Metadata.Name, step.CMD, env, step.Timeout, outputStrings, outputErr)
	}
	// Iterate through the queue to pull out results one-by-one
	// These may be out of order, but is there a better way to do this? Do we need them in order?
	for j := step.OnNode; j <= endNode; j++ {
		out := <-outputStrings
		err := <-outputErr
		if err {
			summary.Timeouts++
			continue // skip handling the output or other assertions since it timed out.
		}
		if len(step.WriteToFile) != 0 {
			f, err := os.OpenFile(step.WriteToFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0664)
			if err != nil {
				color.Red("Failed to open output file: %s", err)
			} else {
				f.WriteString(strings.Join(out, "\n"))
			}
		}
		if len(step.Outputs) != 0 {
			for index, output := range step.Outputs {
				if index >= len(out) {
					color.Red("Not enough lines in output. Skipping")
					break
				}
				line := out[index]
				color.Magenta("### Saving output from line %d to variable %s: %s", output.Line, output.SaveTo, line)
				env = append(env, output.SaveTo+"=\""+line+"\"")
			}
		}
		if len(step.Assertions) != 0 {
			for _, assertion := range step.Assertions {
				if assertion.Line >= len(out) {
					color.Red("Not enough lines in output.Skipping assertions")
					break
				}
				lineToAssert := out[assertion.Line]
				value := ""
				// Find an env that matches the ShouldBeEqualTo variable
				// i.e. RESULT="abc abc" matches ShouldBeEqualTo: RESULT
				// value becomes then abc abc (without quotes)
				for _, e := range env {
					rex := regexp.MustCompile(
						fmt.Sprintf("^%s=\"(.*)\"$",
							assertion.ShouldBeEqualTo))
					found := rex.FindStringSubmatch(e)
					if len(found) == 2 && found[1] != "" {
						value = found[1]
						break
					}
				}
				// If nothing was found in the environment,
				// assume its a literal
				if value == "" {
					value = assertion.ShouldBeEqualTo
				}
				if lineToAssert != value {
					color.Set(color.FgRed)
					fmt.Println("Assertion failed!")
					fmt.Printf("Actual value=%s\n", lineToAssert)
					fmt.Printf("Expected value=%s\n\n", value)
					color.Unset()
					summary.Failures = summary.Failures + 1
				} else {
					summary.Successes = summary.Successes + 1
					color.Green("Assertion Passed")
				}
			}
		}
	}
	return env
}

func getPods(cfg *Config) (*GetPodsOutput, error) {
	// Only return pods that match our deployment.
	cmd := exec.Command("kubectl", "get", "pods", "--output=json", "--selector="+cfg.Selector)

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

func getRunningPods(cfg *Config) (int, error) {
	pods, err := getPods(cfg)
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
func scaleTo(cfg *Config) error {
	number := cfg.Nodes
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
		number_running, err = getRunningPods(cfg)
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
		cmd := exec.Command("kubectl", "exec", name, "-t", "--", "bash", "-c", envString+cmdToRun)
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
		chanStrings <- lines
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
	cmd := exec.Command("kubectl", "exec", name, "-t", "--", "bash", "-c", envString+cmdToRun)
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

func selectNodes(step Step, config Config) int[]{
	/* Handle OnNode EndNode syntax */
	if step.OnNode > 0 {
		if step.Selection != nil {
			fatal("Two node selection methods on test step")
		}
		if step.EndNode == 0 {
			step.EndNode = step.OnNode
			select := make(int[], 1)
			select[0] = step.OnNode
			return select
		}
		return makeRange(step.OnNode, step.EndNode)
	} else if step.Selection == nil {
		fatal("No selection method on test step")
	} else { /* Selection chooses nodes */
		if (step.Selection.Range == nil && step.Selection.Percent == nil 
			  && step.Selection.Subset == nil) {
			fatal("No selection method on test step")
		} else if (step.Selection.Range != nil) {
			return selectNodesRange(step, config)
		} else if (step.Selection.Percent != nil) {
			return selectNodesRange(step)
		} else /* Subset selection */ {
			return selectNodesSubset(step, subsetPartition)
		}
	}
}

func selectNodesRange(step Step, config Config) int[]{
	if step.Selection.Range.Order == SEQUENTIAL {
	  if step.Selection.Range.Start <=0 || 
			step.Selection.Range.End - step.Selection.Range.Start +1 > config.nodes {
				fatal("Invalid range")
		}
		return makeRange(step.Selection.Range.Start, step.Selection.Range.End)
	} else if step.Selection.Range.Order == RANDOM {
		if step.Selection.Range.Number > config.nodes {
			fatal("Invalid range")
		}
		return rand.Perm(config.nodes)[0:step.Selection.Range.Number]

	} else /* Invalid order */ {
			fatal("Invalid Selection format. Order must be SEQUENTIAL or RANDOM ")
	}
} 

func selectNodesPercent(step Step, config Config) int[] {
	var percent int
	percent = step.Selection.Percent.Percent
	if percent > 100 || percent < 0 {
		fatal("Invalid percent")
	}
	numNodes = int( (float64(percent) / 100.0) * float64(config.nodes) )
	if step.Selection.Percent.Order == SEQUENTIAL {
		if step.Selection.Percent.Start -1 + numNodes > config.nodes {
			fatal("Invalid start position")
		}
		return makeRange(step.Selection.Percent.Start, step.Selection.Percent.Start -1 + numNodes)
	} else if step.Selection.Percent.Order == RANDOM {
		return rand.Perm(config.nodes)[0:numNodes]
	} else /* Invalid order */ {
		fatal("Invalid Selection format. Order must be SEQUENTIAL or RANDOM ")
	}
}

func selectNodesSubset(step Step, subsetPartition map[int]int[]) int[]{
	if (subsetPartition == nil) {
		fatal("Subset specified without specifying partion in header")
	}
	selected := make(int[])
	for _, partition := step.Selection.Subset.Indices {
		selected = append(selected, subsetPartition[partition]...)
	}
	return selected
}


func partition(config Config) {
	if config.SubsetPartiton == nil {
		return nil
	} 
	partitionMap := make(map[int]int[])

	if config.SubsetPartition.Order == SEQUENTIAL {
		if config.SubsetPartition.PartitionType == EVEN {
			seqEvenPartition(&partitionMap, config.SubsetPartition.NumberPartitions, config.nodes)
			return partitionMap
		} else if config.SubsetPartition.PartitionType == WEIGHTED {
			seqWeightedPartition(&partitionMap, config.nodes)
		} else /* invalid partition type */{
				fatal("Invalid partition type")
		}
	} else if config.SubsetPartition.Order == RANDOM {
		if config.SubsetPartition.PartitionType == EVEN {
			randEvenPartition(&partitionMap)
			return partitionMap
		} else if config.SubsetPartition.PartitionType == WEIGHTED {
			randWeightedPartition(&partitionMap)
			return partitionMap
		} else /* invalid partition type */{
				fatal("Invalid partition type")
		}
	} else { /* invalid ordering */
		fatal("Partition has invalid order")
	}
}

func seqEvenPartition(partitionMap map[int]int[], numSubsets int, numNodes int) {
	for i := range numSubsets {
		startNode, endNode := getSubsetBounds(i, numSubsets, numNodes)
		partitionMap[i] = makeRange(startNode, endNode)
	}
}

func randEvenPartition(partitionMap map[int]int[], numSubsets int numNodes int) {
	sample := rand.Perm(numNodes)
	for i := range numSubsets {
		startNode, endNode := getSubsetBounds(i, numSubsets, numNodes)
		partitionMap[i] = sample[startNode -1:endNode -1]
	}
}

func weightedPartition(partitionMap map[int]int[], percents int[], numNodes int, random bool){
	/* Get all of the node nums for each partition, then spread 
	   out leftovers from rounding among the earliest subsets */
	if len(percents) > numNodes {
		fatal("No more partitions than number of nodes")
	}
	/* Calculate size of each partitions */
	partitionSize := make(int[])
	var size, sum, leftovers, acc int
	sum = 0
	for _, percent := range percents {
		size = int( (float64(percent) / 100.0) * float64(numNodes) )
		sum += size
		partitionSize = append(partitionSize, size)
	}

	/* Calculate indices of nodes in partition, accounting for rounding error */
	leftovers = numNodes - sum
	acc = 0
	var sample []int
	if random {
		sample = rand.Perm(numNodes)
	} else { /* sequential */
		sample = makeRange(1, numNodes)
	}
	for i,size := range partitionSize {
		if leftovers > 0 {
			leftovers -= 1
			size += 1
		}
		partitionMap[i] := sample[acc:size]
		acc = size
}


func seqWeightedPartition( partitionMap map[int]int[], percents int[], numNodes int) {
	weightedPartition(partitionMap map[int]int[], percents int[], numNodes int, false)
}

func randWeightedPartition( partitionMap map[int]int[], percents int[], numNodes int) {
	weightedPartition(partitionMap map[int]int[], percents int[], numNodes int, true)
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

funct makeRange(min, max int) []int {
	a := make([]int, max-min+1)
	for i := range a {
		a[i] = min + i
	}
	return a
}
