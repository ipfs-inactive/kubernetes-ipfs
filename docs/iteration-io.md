### Iteration and Array IO Specification

This DSL feature specifies that a command should be run for a given number of 
iterations.  These iterations can be bounded by an integer or correspond to the
length of an iterable environment variable.  Creation and access of array 
environment variables is also described in this document.


# append_to output
When present an internal array of main.go 
associated with the specified environment variable adds another entry 
at its last index.  The array is created if it has not yet been specified.

Ex:
```
 - steps:
    name: add random byte chunks to ipfs
    selection: 
      percent:
        order: SEQUENTIAL
        start: 1
        percent: 100
    cmd: head -c 10 /dev/urandom | base64 | ipfs add -q
    outputs:
    - line: 0
      append_to: HASH
```

# iteration or node specific array environment variable reference
Two possible indexes, %s always applies and corresponds to node index, %i 
applies if this step contains an iteration.

Ex:
```
...
cmd: ipfs-cluster-ctl pin add $HASH[%i]
...
```
This is used in more detail in the final section

# iteration
The for: key in the test yaml specifies 
1. How many iterations this step should be run for
2. Over what structure, internal array or numeric bound this step iterates

Ex1 -- simple for structure with fixed (or parameterized) iteration number:
```
- steps:
name: add random byte chunks to ipfs on one node
on_node: 1
for: 
 iter_structure: "BOUND"
 num: 100
cmd: head -c 10 /dev/urandom | base64 | ipfs add -q
outputs:
 - line: 0
   append_to: HASH
```

Ex2 -- for structure over length of variable array
```
- steps:
 ...
 name: pin hashes to ipfs-cluster
 on_node: 1
 for: 
  iter_structure: "HASH"
 cmd: ipfs-cluster-ctl pin add ${HASH[%i]}
```

Ex3 -- using default %s index to reference node this is being run on
```
- steps:
 name: Get cluster node ids
 selection:
   percent:
     order: SEQUENTIAL
     start: 1
     percent: 100
 cmd: ipfs-cluster-ctl id | jq -r '.id'
 outputs:
  - line: 0
    append_to: IDS
 ...
 name: Make sure ids haven't changed
 selection:
   percent:
     order: RANDOM
     percent: 100
 cmd: [[ ${IDS[%s]} == ipfs-cluster-ctl id | jq -r '.id' ]]; echo $?
 assertions:
   line: 0
   should_be_equal_to: "0"

```
