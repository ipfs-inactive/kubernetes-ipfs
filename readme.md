# Get Started

kubernetes-ipfs works with both a full kubernetes deployment, or the [minikube](https://kubernetes.io/docs/getting-started-guides/minikube/) all-in-one system

## With Minikube

`./reset-minikube.sh` to set minikube up in a clean state

`./init.sh` to create go-ipfs and grafana deployments on minikube

## Running tests

`go run main.go tests/simple-add-and-cat.yml` 

The go application returns `0` when expectations were met, `1` when they failed

# Example Reports

## Simple Add > Cat test with 2 nodes

go-ipfs:v0.4.0 - https://snapshot.raintank.io/dashboard/snapshot/Jk2Ek5pR28xF3vXAeO75thmmA6WBNbR0

go-ipfs:v0.4.4 - https://snapshot.raintank.io/dashboard/snapshot/JVmEIB1Ofac3jRBeOP01XQmdYWEMcxSa

go-ipfs:v0.4.5-pre1 - https://snapshot.raintank.io/dashboard/snapshot/JteUjtvn3hRjlGolcmcP2vn5Ud6gYwPW

## Simple Add > Pin test with 5 nodes

go-ipfs:v0.4.5-pre1 - https://snapshot.raintank.io/dashboard/snapshot/DfHSuzIo3TNDrMmJedqmSMvFniGLIWwH

# Writing tests

The tests are specified in a .yml file for each test.

## Header
* name: Name the test
* nodes: How many nodes to run for the test. Kubernetes-ipfs will automatically scale the deployment to match the value here before starting
* times: How many times to run the full test.
* expected: define the number of expected outcomes. This value should be outcomes per test * times. Specify the expected successes, failures, and timeouts.

## Steps
Each step contains a few flags that specify how they will be run, and a `cmd` which is the command to run on the node
* name: Name the step
* on_node: On which node number should we run this test?
* end_node: When specified, we will run this test in parallel from on_node to end_node inclusive. Useful for testing simultaneous group interactions.
* outputs: Specify a line number of output and what environment variable to save it to. It can be used for the following input section
* inputs: Specify the environment variables to take in for this command.
* cmd: Verbatim command to run on the node. Bash variables will be evaluated.
* timeout: At this many seconds, the step will be cancelled and counted as "timeout".
* assertions: At the moment, only `should_be_equal_to` Specify that a line number of stdout should be equal to a line you have used save_to on. On success, adds a success count, on fail, adds a failure count.

