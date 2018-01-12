# Timeouts

## Timeout for a step

When running a command using IPFS, the result of a failure to resolve is often to hang a long time rather than an outright failure as IPFS searches for a way to connect you to the data you requested.

Sometimes, we expect this to occur, and want to see that the test fails to resolve the data without waiting for its full timeout.

To this end, one can make use of the `timeout` feature, and the `timeouts` parameter under the expected section (see validation.md). If a timeout is met, the timeouts value increments, and we can check against that
to confirm our suspicions that the test has failed.

The format of the parameter is the delay after which we consider the command to have timed out, expressed in **whole seconds**.

```
 - name: Cat file on node 2
    on_node: 2
    inputs:
      - FILE
      - HASH
    cmd: ipfs cat $HASH
    timeout: 2 
```
