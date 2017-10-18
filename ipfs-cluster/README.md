Welcome to the kubernetes-ipfs cluster tests

To launch all tests execute ./runner.sh <N> <Y>
   N is the total number of ipfs-cluster nodes (kubernetes pods) running in the
     configuration
   Y is the total number of pins that tests add to cluster

To launch the ipfs-cluster deployment for a given node number and
kubernetes-ipfs config* but not start running all tests, execute
./runner.sh <N> <Y> manual
This is useful for running individual tests

* Note: the config (./tests/config.yml) can be changed manually if different
parameters are desired between multiple tests running on the same deployment

Cluster test initialization structure
  runner.sh is the entrypoint starting all initialization
    parses args
    updates deployment to launch N nodes
    launches init.sh
    launches config-writer.sh
    runs kubernetes-ipfs on all tests in ./tests folder

  init.sh initializes the kubernetes cluster to run ipfs-cluster
    deploys monitoring
    deploys ipfs-cluster
    waits for ipfs-cluster nodes to become initialized
    adds all ipfs-cluster nodes into 1 shared ipfs-cluster
    prints info about the cluster

  config-writer.sh writes N, Y and derived parameters to a kubernetes-ipfs config
  used for all cluster tests
    parses args
    writes config values to file
      (due to current limitations in parameter support in testing DSL each
       combination (e.g. 4Y + N) of parameters needs to be written to config)
    