These tests are used for sanity checking the DSL implementation of node
selection.  Each test runs an operation on a given subset of nodes.
Right now the easiest verification process is manually inspecting
if each node has been changed in the desired way.  I am using
ipfs-cluster-service and ipfs deamon service kill, because it is
simple to identify if the operations have been completed per node
