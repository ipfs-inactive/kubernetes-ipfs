These tests are used for sanity checking the DSL implementation of node
selection.  Each test runs an operation on a given subset of nodes.
Right now the easiest verification process is manually inspecting
if each node has been changed in the desired way.  I am using
writes to files to quickly inspect with ls.  As each test uses the
same mechanism for running commands given an index array, these
tests are mostly useful for comparing the expected set of running
nodes, usually specified in the test name, to the printout of
running indices in each step
