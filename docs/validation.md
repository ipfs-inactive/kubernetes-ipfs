# Validation & Assertions

## Inputs

Within the confines of a 'step', adding the inputs parameter allows the user to specify the same name within the `cmd` tag as a standard `bash` variable (`$VARNAME`), 
which creates the possibility to chain inputs from one step to the next.

The format is simply to specify a name.

```
- name: Cat added file
    on_node: 1
    inputs:
      - HASH
    cmd: ipfs cat $HASH
```

## Outputs

With in the confines of a 'step', adding the 'outputs' parameter allows the user to send the output from the command on a given line to a given name.
This can then be verified later to validate the output of a particular command.

The format is for a given `line`, save the output to the named variable `save_to`.

For documentation of the `append_to` feature, see `iteration-io.md`.

```
steps:
  - name: Add file
    on_node: 1
    cmd: head -c {{FILE_SIZE}} /dev/urandom | base64 > /tmp/file.txt && cat /tmp/file.txt && ipfs add -q /tmp/file.txt
    outputs: 
    - line: 0
      save_to: FILE
    - line: 1
      save_to: HASH
```

## Assertions

Once we're ready to validate the output of a particular command (often after chaining data from one node to another), we can make use of the assertions statement
to create a check which `kubernetes-ipfs` will validate.

Assertions as of the current version only have the parameter `should_be_equal_to`, which states that the given line should be equal to the provided output.

If it is not, `kubernetes-ipfs` adds 1 to the 'failures' count. If it is, `kubernetes-ipfs` adds 1 to the 'successes' count.

The format is as follows, using one of the 'inputs' passed to the step.

```
 name: Cat added file
    on_node: {{    ON_NODE}}
    inputs:
      - FILE
      - HASH
    cmd: ipfs cat $HASH
    assertions:
    - line: 0
      should_be_equal_to: FILE
```

## Expects

Once the successes and failures have been talied up across runs for a particular test, the observed value is compared against the `expected` block.
If the observed value matches, `kubernetes-ipfs` will report that expectations were met, and returns a 0 exit code. If they don't match, it will return
a non-zero exit code, and state that the expectations were not met.

Within the test definition:

```
 expected:
       successes: 10
       failures: 0
       timeouts: 0
```

In this case, if the test fails once, we will return a non-zero exit code and fail the test.
