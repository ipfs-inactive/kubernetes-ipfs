package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
	"strconv"

	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/color"

	yaml "gopkg.in/yaml.v2"
)

// DEBUG decides if we should have debug output enabled or not
var DEBUG = false

var DEPLOYMENT_NAME = "go-ipfs-stress"

const (
	random     = "RANDOM"
	sequential = "SEQUENTIAL"
	even       = "EVEN"
	weighted   = "WEIGHTED"
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
	Line     int    `yaml:"line"`
	SaveTo   string `yaml:"save_to"`
	AppendTo string `yaml:"append_to"`
}

// Assertion is
type Assertion struct {
	Line            int    `yaml:"line"`
	ShouldBeEqualTo string `yaml:"should_be_equal_to"`
}

// For is the iteration structure
// determining how many iterations of a step are run
type For struct {
	/* BOUND to iterate from 1 to number, internal array variable name to iterate over array */
	IterStructure string `yaml:"iter_structure"` 
	Number        int    `yaml:"number"`        // Valid for BOUND
}

// Step is
type Step struct {
	Name string `yaml:"name"`

	/* Old style selection remains supported */
	OnNode      int         `yaml:"on_node"`
	EndNode     int         `yaml:"end_node"`
	/* New style selection */
	Selection   *Selection  `yaml:"selection"`
	For         *For 			  `yaml:"for"`

	CMD         string      `yaml:"cmd"`
	Timeout     int         `yaml:"timeout"`
	Outputs     []Output    `yaml:"outputs"`
	Inputs      []string    `yaml:"inputs"`
	Assertions  []Assertion `yaml:"assertions"`
	WriteToFile string      `yaml:"write_to_file"`
}

/* Selection is used to pick nodes for running commands
each step takes a selection object which allows
   tests to specify nodes in 2 ways

   1: Select nodes by range like with OnNode EndNode
      - Can choose Start and End
      - Can choose Number, and Number of random nodes will run
      **Example**
      selection:
        range:
          order: SEQUENTIAL
          start: 1
          end: 10


   2: Select percentage of nodes to run
      - Can choose Start, Percentage (run 30 % of nodes starting at node 2)
      - Can choose Percentage (run 30 % of nodes choosing at random)
      **Example**
      selection:
        percent:
          order: RANDOM
          percent: 33

	    In addition both percentage and range selections can specify
	 subsets of nodes over which the percents and ranges are calculated
	 The subset partitions are declared earlier in the config using a 
	 SubsetPartition.  A partition must be specified to use the subset
	 field within a selection.  Example 1 will run on 1 random node of 
	 subset 1 and subset 5, example 2 will run on 50% of the nodes in 
	 subset 3.
	    **Example 1**
	    selection:
	    	subset: [1, 5]
	    	range:
	    		order: RANDOM
	    		number: 1

	    **Example 2**
	    selection:
	    	subset: [3]
	    	percent:
	    		order: SEQUENTIAL
	    		start: 1
	    		percent: 50
*/
type Selection struct {
	Range   *Range   `yaml:"range"`
	Percent *Percent `yaml:"percent"`
	Subsets  []int  `yaml:"subset"`
}

/* Range is a selection method to choose a sequence of nodes.  The
   range can be sequential, in which case the start and end node
   index must be specified.  The range can also be random, in which
   case the number of nodes must be specified
*/
type Range struct {
	Order  string `yaml:"order"`  /* RANDOM or SEQUENTIAL */
	Start  int    `yaml:"start"`  /* Valid for SEQUENTIAL */
	End    int    `yaml:"end"`    /* Valid for SEQUENTIAL */
	Number int    `yaml:"number"` /* Valid for RANDOM */
}

/* Percent is a selection method used to choose a percentage of the
   total nodes.  Order can either be random or sequential.  Sequential
   percentages begin at a start node and choose the given number of 
   nodes in order from the start inclusive.  Random percentages simply
   choose a number of random nodes that make up the given percentage.
   Percentages truncate down.  For example 50% of 5 nodes selects 2.
*/
type Percent struct {
	Order   string `yaml:"order"`   /* RANDOM or SEQUENTIAL */
	Start   int    `yaml:"start"`   /* Valid for SEQUENTIAL */
	Percent int    `yaml:"percent"` /* Valid for RANDOM */
}



// Config is
type Config struct {
	Nodes           int              `yaml:"nodes"`
	Selector        string           `yaml:"selector"`
	Times           int              `yaml:"times"`
	GraceShutdown   time.Duration    `yaml:"grace_shutdown"`
	Expected        Expected         `yaml:"expected"`
	SubsetPartition *SubsetPartition `yaml:"subset_partition"`
}

/* SubsetParition controls the partitioning of the nodes into 
   distinct, non-overlapping subsets that can be specified by index
   in steps of the test to select nodes for running commands.
   SubsetPartition's PartitionType, or weighting specifies whether
   the partition splits the nodes into even groups or groups of 
   nodes proportional to different percentages.  SubsetPartitions Order
   specifies whether nodes to fill the different subsets will be chosen
   sequentially or at random.  Even partitions must specify the total
   number of partitions, and weighted partitions must specify a list of
   the different percentages of nodes in each partition.
*/
type SubsetPartition struct {
	PartitionType    string `yaml:"partition_type"`    /* Either EVEN or WEIGHTED */
	Order            string `yaml:"order"`             /* Either RANDOM or SEQUENTIAL */
	Percents         []int  `yaml:"percents"`          /* Valid for WEIGHTED */
	NumberPartitions int    `yaml:"number_partitions"` /* Valid for EVEN */
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
	
	var test Test
	var summary Summary
	var subsetPartition map[int][]int

	err := readTestFile(filePath, &test)
	if err != nil {
		fatal(err)
	}
	err = validate(&test, &subsetPartition)
	if err != nil {
		fatal(err)
	}
	RunTests(&summary, &test, subsetPartition)
	PrintResults(summary, test)
}

func readTestFile(filePath string, test *Test) error {
	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal([]byte(fileData), &test)
	if err != nil {
		return err
	}

	debug("Configuration:")
	debugSpew(test)
	return nil
}

func validate (test *Test, subsetPartition *map[int][]int) error {
	/* Include call to partition nodes into subsets if the partition field is included in the
	   config.  subsetPartion is nil if it is not included in config.  Tests must include this
	   in the config in order to use the subset selection method to choose nodes later on during
	   testing  */
	rand.Seed(time.Now().UTC().UnixNano())
	var err error
	*subsetPartition, err = partition(test.Config)
	if err != nil {
		color.Red("## Failed to parse subset partition: " + err.Error())
		return err
	}

	err = validateSelections(test.Steps, *subsetPartition, test.Config)
	if err != nil {
		color.Red("## Step selections did not validate")
		return err
	}
	return nil
}

func RunTests (summary *Summary, test *Test, subsetPartition map[int][]int) {
	summary.TestsToRun = test.Config.Times
	summary.Start = time.Now()
	var err error
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
		envArrays := make(map[string][]string)
		for _, step := range test.Steps {
			nodeIndices := selectNodes(step, test.Config, subsetPartition)
			fmt.Printf("The node indices %v\n", nodeIndices)
			env, envArrays = handleStep(*pods, &step, summary, env, envArrays, nodeIndices)
		}
		summary.TestsRan = summary.TestsRan + 1
	}
}

func PrintResults (summary Summary, test Test) {
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

func handleStep(pods GetPodsOutput, step *Step, summary *Summary, env []string, envArrays map[string][]string, nodeIndices []int) ([]string, map[string][]string) {
	color.Blue("### Running step %s on nodes %v", step.Name, nodeIndices)
	if len(step.Inputs) != 0 {
		for _, input := range step.Inputs {
			color.Blue("### Getting variable " + input)
		}
	}
	color.Magenta("$ %s", step.CMD)
	numNodes := len(nodeIndices)

	/* Determine number of iterations */
	var numIters int
	if step.For == nil {
		numIters = 1
	} else if step.For.IterStructure == "BOUND" {
		numIters = step.For.Number
	} else { /* Iterating over an array */
		numIters = len(envArrays[step.For.IterStructure])
	}

	/* Find all array variables used and add to environment */

  r, _ := regexp.Compile("([a-zA-Z_][a-zA-Z0-9_]*)\\[(%s|%i)\\]")
	regexRaw := r.FindAllStringSubmatch(step.CMD, -1)
	arrayVars := make([]string, 0)
	for _, raw := range regexRaw {
		arrayVars = append(arrayVars, raw[1])
	}

	for _, arrayName := range arrayVars { 
	  // Go from envArray table at given index to a string defining a bash array 
		bashString := arrayName + "=("
		for _, s := range envArrays[arrayName] {
			bashString += "'" + s + "' "
		}
		bashString += ")"
		env = append(env, bashString)
	}

	color.Magenta("Running parallel on %d nodes for %d iterations.", numNodes, numIters)
	for i := 0; i < numIters; i++ {

		// Initialize a channel with depth of number of nodes we're testing on simultaneously
		outputStrings := make(chan []string, numNodes)
		outputErr := make(chan bool, numNodes)
		for _, idx := range nodeIndices {
			// Command search and replace for index references into array (%i/%s) and 
			r1, _ := regexp.Compile("\\[%s\\]")
			r2, _ := regexp.Compile("\\[%i\\]")

			command := r1.ReplaceAllString(step.CMD, "[" + strconv.Itoa(idx-1) + "]")
			command = r2.ReplaceAllString(command, "[" + strconv.Itoa(i) + "]")

			// Hand this channel to the pod runner and let it fill the queue
			runInPodAsync(pods.Items[idx-1].Metadata.Name, command, env, step.Timeout, outputStrings, outputErr)
		}
		// Iterate through the queue to pull out results one-by-one
		// These may be out of order, but is there a better way to do this? Do we need them in order?
		for j := 0; j < numNodes; j++ {
			out := <-outputStrings
			err := <-outputErr
			if err {
				summary.Timeouts++
				continue // skip handling the output or other assertions since it timed out.
			}
			if len(step.WriteToFile) != 0 {
				errWrite := ioutil.WriteFile(step.WriteToFile, []byte(strings.Join(out, "\n")), 0664)
				if errWrite != nil {
					color.Red("Failed to write output file: %s", err)
				}
			}
			if len(step.Outputs) != 0 {
				for index, output := range step.Outputs {
					if index >= len(out) {
						color.Red("Not enough lines in output. Skipping")
						break
					}
					line := out[index]
					if output.SaveTo != "" {
						color.Magenta("### Saving output from line %d to variable %s: %s", output.Line, output.SaveTo, line)
						env = append(env, output.SaveTo+"=\""+line+"\"")
					} else if output.AppendTo != "" {
						color.Magenta("### Appending output from line %d to array variable %s: %s", output.Line, output.AppendTo, line)
						array, ok := envArrays[output.AppendTo]
						if !ok {
							envArrays[output.AppendTo] = make([]string, 0)
						}
						envArrays[output.AppendTo] = append(array, line)
					}
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
	}
	return env, envArrays
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

func selectNodes(step Step, config Config, subsetPartition map[int][]int) []int {
	var nodes []int
	switch {
	case step.Selection == nil:
		nodes = selectNodesFromOnStep(step)
	default: /* step.Selection != nil */
		nodes = selectNodesFromSelection(step, config, subsetPartition)
	}
	return nodes
}

func selectNodesFromSelection(step Step, config Config, subsetPartition map[int][]int) []int {
	var nodes []int 
	switch {
	case step.Selection.Range != nil && step.Selection.Subsets == nil:
		nodes = selectNodesRange(step, config, makeRange(1, config.Nodes))
	case step.Selection.Range !=nil && step.Selection.Subsets != nil:
		nodes = selectNodesRange(step, config, subsetPartition[step.Selection.Subsets[0]])
		for _, subset := range step.Selection.Subsets[1:] {
			nodes = append(nodes, selectNodesRange(step, config, subsetPartition[subset])...)
		}
	case step.Selection.Percent != nil && step.Selection.Subsets != nil:
		nodes = selectNodesPercent(step, config, makeRange(1, config.Nodes))
	case step.Selection.Percent != nil && step.Selection.Subsets != nil:
		nodes = selectNodesRange(step, config, subsetPartition[step.Selection.Subsets[0]])
		for _, subset := range step.Selection.Subsets[1:] {
			nodes = append(nodes, selectNodesPercent(step, config, subsetPartition[subset])...)
		}
	}
	return nodes
}

func selectNodesFromOnStep(step Step) []int {
	if step.EndNode == 0 {
		selected := make([]int, 1)
		selected[0] = step.OnNode
		return selected
	}
	return makeRange(step.OnNode, step.EndNode)
}

func selectNodesRange(step Step, config Config, nodes []int) []int {
	var selection []int
	switch step.Selection.Range.Order {
	case sequential:
		start := step.Selection.Range.Start - 1
		end := step.Selection.Range.End - 1
		selection = makeRange(nodes[start], nodes[end])
	case random:
		selection = shuffle(nodes)[0:step.Selection.Range.Number]
	}
	return selection
}

func selectNodesPercent(step Step, config Config, nodes []int) []int {
	var selection []int
	percent := step.Selection.Percent.Percent
	numNodes := int((float64(percent) / 100.0) * float64(config.Nodes))
	switch step.Selection.Percent.Order {
	case sequential:
		start := step.Selection.Percent.Start - 1 
		end := step.Selection.Percent.Start - 2 + numNodes
		selection = makeRange(nodes[start], nodes[end])
	case random:
		selection = shuffle(nodes)[0:numNodes]
	}
	return selection
}

func validateError(stepNum int, errorStr string) error {
	return errors.New(fmt.Sprintf(errorStr+" on test step %d", stepNum))
}

func validateSelections(steps []Step, subsetPartition map[int][]int, config Config) error {
	for idx, step := range steps {
		/* Exactly one of on_node or selection per step */
		if step.OnNode <= 0 && step.Selection == nil {
			return validateError(idx, "No selection method")
		}
		if step.OnNode > 0 && step.Selection != nil {
			return validateError(idx, "Two node selection methods")
		}
		if step.OnNode > 0 {
			continue
		}
		/* Selection specific verification */
		switch {
		/* If method is selection, exactly one selection format is used */
		case step.Selection.Range == nil && step.Selection.Percent == nil:
			return validateError(idx, "No selection method")
		case step.Selection.Range != nil && step.Selection.Percent != nil:
			return validateError(idx, "Two selection formats")
		case step.Selection.Subsets != nil:
			if subsetPartition == nil {
				return validateError(idx, "Subset specified without specifying partion in header")
			} 
			if len(step.Selection.Subsets) > len(subsetPartition) || max(step.Selection.Subsets) > len(subsetPartition) {
				return validateError(idx, "Subset specifies too many partitions")
			}
			if !allPositive(step.Selection.Subsets) {
				return validateError(idx, "Subset specifies invalid partition index")
			}

		case step.Selection.Range != nil:
			/* Parameters to validate for each subset */
			checkLengths := make([]int, 0)
			if step.Selection.Subsets == nil {
				checkLengths = append(checkLengths, config.Nodes)
			} else {
				for _, subset := range step.Selection.Subsets {
					checkLengths = append(checkLengths, len(subsetPartition[subset]))
				}
			}
			switch step.Selection.Range.Order {
			case sequential:
				for i := range checkLengths {
					if step.Selection.Range.Start <= 0 ||
						step.Selection.Range.End-step.Selection.Range.Start+1 > checkLengths[i]||
						step.Selection.Range.End > checkLengths[i] {
						return validateError(idx, "Invalid range")
					}
				}
			case random:
				for i := range checkLengths {
					if step.Selection.Range.Number > checkLengths[i] {
						return validateError(idx, "Invalid range")
					}
				}
			default:
				return validateError(idx, "Invalid order, must be SEQUENTIAL of RANDOM")
			}
		case step.Selection.Percent != nil:
			percent := step.Selection.Percent.Percent
			if percent > 100 || percent < 0 {
				return validateError(idx, "Invalid percent")
			}
			/* Parameters to validate for each subset */
			checkLengths := make([]int, 0)
			numNodes := make([]int, 0)
			if step.Selection.Subsets == nil {
				checkLengths = append(checkLengths, config.Nodes)
				numNodes = append(numNodes, int((float64(percent) / 100.0) * float64(config.Nodes)))
			} else {
				for _, subset := range step.Selection.Subsets {
					checkLengths = append(checkLengths, len(subsetPartition[subset]))
					numNodes = append(numNodes, int((float64(percent) / 100.0) * float64(len(subsetPartition[subset]))))
				}
			}
			switch step.Selection.Percent.Order {
			case sequential:				
				for i := range checkLengths {
					if step.Selection.Percent.Start-1+numNodes[i] > checkLengths[i] || step.Selection.Percent.Start < 1{
						return validateError(idx, "Invalid start position")
					}
				}
			case random: /* No checks needed */
			default:
				return validateError(idx, "Invalid order, must be SEQUENTIAL of RANDOM")
			}
		}
	}
	return nil
}

func partition(config Config) (map[int][]int, error) {
	if config.SubsetPartition == nil {
		return nil, nil
	}
	partitionMap := make(map[int][]int)
	var err error
	switch config.SubsetPartition.Order {
	case sequential:
		switch config.SubsetPartition.PartitionType {
		case even:
			err = seqEvenPartition(partitionMap, config.SubsetPartition.NumberPartitions, config.Nodes)
		case weighted:
			err = seqWeightedPartition(partitionMap, config.SubsetPartition.Percents, config.Nodes)
		default:
			err = errors.New("Partition has invalid partition weighting")
		}
	case random:
		switch config.SubsetPartition.PartitionType {
		case even:
			err = randEvenPartition(partitionMap, config.SubsetPartition.NumberPartitions, config.Nodes)
		case weighted:
			err = randWeightedPartition(partitionMap, config.SubsetPartition.Percents, config.Nodes)
		default:
			err = errors.New("Partition has invalid partition weighting ")
		}
	default:
		err = errors.New("Partition has invalid ordering")
	}
	if err != nil {
		partitionMap = nil
	}
	return partitionMap, err
}

func seqEvenPartition(partitionMap map[int][]int, numSubsets int, numNodes int) error {
	for i := 1; i <= numSubsets; i++ {
		startNode, endNode := getSubsetBounds(i, numSubsets, numNodes)
		partitionMap[i] = makeRange(startNode, endNode)
	}
	return nil
}

func randEvenPartition(partitionMap map[int][]int, numSubsets int, numNodes int) error {
	sample := onePerm(numNodes)
	for i := 1; i <= numSubsets; i++ {
		startNode, endNode := getSubsetBounds(i, numSubsets, numNodes)
		partitionMap[i] = sample[startNode-1 : endNode]
	}
	return nil
}

func weightedPartition(partitionMap map[int][]int, percents []int, numNodes int, random bool) error {
	/* Get all of the node nums for each partition, then spread
	   out leftovers from rounding among the earliest subsets */
	if len(percents) > numNodes {
		return errors.New("More partitions than number of nodes")
	}
	/* Calculate size of each partitions */
	partitionSize := make([]int, 0)
	var size, sum, percentSum, leftovers, acc int
	sum = 0
	percentSum = 0
	for _, percent := range percents {
		size = int((float64(percent) / 100.0) * float64(numNodes))
		percentSum += percent
		sum += size
		partitionSize = append(partitionSize, size)
	}

	if percentSum != 100 {
		return errors.New("Total subset percentages does not add to 100")
	}

	/* Calculate indices of nodes in partition, accounting for rounding error */
	leftovers = numNodes - sum
	acc = 0
	var sample []int
	if random {
		sample = onePerm(numNodes)
	} else { /* sequential */
		sample = makeRange(1, numNodes)
	}
	for i, size := range partitionSize {
		if leftovers > 0 {
			leftovers--
			size++
		}
		partitionMap[i+1] = sample[acc : acc+size]
		acc = acc + size
	}
	return nil
}

func seqWeightedPartition(partitionMap map[int][]int, percents []int, numNodes int) error {
	return weightedPartition(partitionMap, percents, numNodes, false)
}

func randWeightedPartition(partitionMap map[int][]int, percents []int, numNodes int) error {
	return weightedPartition(partitionMap, percents, numNodes, true)
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

func makeRange(min, max int) []int {
	a := make([]int, max-min+1)
	for i := range a {
		a[i] = min + i
	}
	return a
}

func onePerm(N int) []int {
	ret := rand.Perm(N)
	for i := 0; i < len(ret); i++ {
		ret[i]++
	}
	return ret
}

func shuffle(ints []int) []int{
	shuffled := make([]int, len(ints))
	idxs := rand.Perm(len(ints))
	for i, shufi := range idxs {
		shuffled[i] = ints[shufi]
	}
	return shuffled
}

/* Calculate max of an array of positive integers */
func max(ints []int) int {
	ret := 0
	for _, num := range ints {
		if num > ret {
			ret = num
		}
	}
	return ret
}

func allPositive(ints []int) bool {
	for _, num := range ints {
		if num <= 0 {
			return false
		}
	}
	return true
}
