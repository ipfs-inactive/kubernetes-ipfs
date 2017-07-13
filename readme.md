Get Started
===========

kubernetes-ipfs works with both a full kubernetes deployment, or the
[minikube](https://kubernetes.io/docs/getting-started-guides/minikube/)
all-in-one system

With Minikube
-------------

`./reset-minikube.sh` to set minikube up in a clean state

`./init.sh` to create go-ipfs and grafana deployments on minikube

Running tests
-------------

`go run main.go tests/simple-add-and-cat.yml`

The go application returns `0` when expectations were met, `1` when they failed


Metrics Gathering: Prometheus/Grafana
=====================================

The following steps detail how to access the Grafana web UI's on your local
machine.

Before running the `init.sh` script, ensure that the `grafana-core` deployment
in the `prometheus-manifests.yml` has these variables set as shown:

```yml
- name: GF_AUTH_BASIC_ENABLED
  value: "false"
- name: GF_AUTH_ANONYMOUS_ENABLED
  value: "true"
- name: GF_AUTH_ANONYMOUS_ORG_ROLE
  value: Admin
```

This allows you to easily access the Grafana interface without any sort of
authentication.

Then, after running `init.sh`, you need to tell Kubernetes to forward
connections on one of your local ports to the port that's hosting the Grafana
interface. To do so, you can:

1.  Get the name of the pod hosting the grafana-core deployment. The command
    with an example output:

    ``` sh
    $ kubectl get pods --namespace=monitoring | grep grafana-core | awk '{print $1}'
    grafana-core-2701824778-j52cq
    ```

2.  Tell `kubectl` to forward all connections on a local port to port 3000 on
    the grafana-core pod found in the previous step. In this example, we simply
    use port 3000 for the local port as well:

    ``` sh
    kubectl port-forward --namespace=monitoring grafana-core-2701824778-j52cq 3000:3000
    ```

From there, you can access Grafana's web UI by navigating to `localhost:3000` in
your browser.


Example Reports
---------------

### Simple Add > Cat test with 2 nodes

go-ipfs:v0.4.0 - https://snapshot.raintank.io/dashboard/snapshot/Jk2Ek5pR28xF3vXAeO75thmmA6WBNbR0

go-ipfs:v0.4.4 - https://snapshot.raintank.io/dashboard/snapshot/JVmEIB1Ofac3jRBeOP01XQmdYWEMcxSa

go-ipfs:v0.4.5-pre1 - https://snapshot.raintank.io/dashboard/snapshot/JteUjtvn3hRjlGolcmcP2vn5Ud6gYwPW

### Simple Add > Pin test with 5 nodes

go-ipfs:v0.4.5-pre1 - https://snapshot.raintank.io/dashboard/snapshot/DfHSuzIo3TNDrMmJedqmSMvFniGLIWwH


Writing tests
=============

The tests are specified in a .yml file for each test.

Header
------

-   name: Name the test
-   nodes: How many nodes to run for the test. Kubernetes-ipfs will
    automatically scale the deployment to match the value here before starting
-   times: How many times to run the full test.
-   expected: define the number of expected outcomes. This value should be
    outcomes per test * times. Specify the expected successes, failures, and
    timeouts.

Steps
-----

Each step contains a few flags that specify how they will be run, and a `cmd` which is the command to run on the node

-   name: Name the step
-   on_node: On which node number should we run this test?
-   end_node: When specified, we will run this test in parallel from on_node
    to end_node inclusive. Useful for testing simultaneous group interactions.
-   selection: An alternate way to choose the nodes that run a command.  Allows
    for specifying ranges, percents and consistent subsets succinctly
-   for: An optional way to specify that a step be ran more than once.  Can
    specify an iteration bound or a for each style iteration over an input array
-   outputs: Specify a line number of output and what environment variable to
    save it to. It can be used for the following input section
-   inputs: Specify the environment variables to take in for this command.
-   cmd: Verbatim command to run on the node. Bash variables will be evaluated.
-   timeout: At this many seconds, the step will be cancelled and counted as
    "timeout".
-   assertions: At the moment, only `should_be_equal_to` Specify that a line
    number of stdout should be equal to a line you have used save_to on. On
    success, adds a success count, on fail, adds a failure count.

