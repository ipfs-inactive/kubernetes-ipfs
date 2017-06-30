##Selection Specification

The DSL selection feature chooses nodes on which to run commands.
In addition to the on_node/end_node option each step can alternatively take
a selection object which allows tests to specify nodes in 2 major ways, ranges
and percentages.

# Ranges
Select nodes by range
      - specify order = {SEQUENTIAL, RANDOM}
      - Similar to on_node / end_node when specified sequentially
      - When specified randomly choose a number of random nodes
      **Example 1: run a command on nodes 1 through 10 inclusive **

``` selection:
        range:
          order: SEQUENTIAL
          start: 1
          end: 10
```

      **Example 2: run a command on 5 random nodes **

```selection:
        range:
          order: RANDOM
          number: 5
```
# Percents 
Select nodes by percent
      - specify order = {SEQUENTIAL, RANDOM}
      - When specifying sequential choose start node and percent
      - When specifying random just choose percent
      - If a command draws nodes at random it will not repeat node indexes
      e.g running at random on 100 percent will run the command on all nodes
      - If percentages do not correspond to whole numbers then the ceiling of 
      the percentage is used to specify the number of nodes to run


      **Example 3: run a command on a range of nodes equal to 30 percent
        of the total nodes and starting on the second node **

```selection:
        percent: 
          order: SEQUENTIAL
          start: 2 
          percent: 30
```

      **Example 4: run a command on 55 percent of randomly indexed nodes **

```selection:
        percent: 
          order: RANDOM
          percent: 55
```


# Subset specification and usage
        - Just as they can be specified with respect to the entire collection 
          of nodes as shown above, percents and ranges can be specified with 
          respect to a partition of the nodes.  
        - Partitions are specified in the config header of the 
          test file.  To specify the partition of nodes into subsets for a
          given test, we use the subset_partition structure. 
        - The first choice to make is whether to divide the nodes up evenly or 
          into specified chunks by percentage (EVEN vs WEIGHTED)  
        - The second choice is whether nodes should fill partitions 
          sequentially, or indexes should be selected at random
          (SEQUENTIAL vs RANDOM).
        **Example 5: Break the nodes into 3 subsets of different weights in 
          order.  The first 10% of nodes go to subset 1, the following 5% of 
          nodes go to subset 2, the final 85% of nodes go to subset 3 **

```config:
   ...
     subset_partion:
       partition_type: WEIGHTED
       order: SEQUENTIAL
       percents: [10, 5, 85]
   ...
```

        **Example 6: Break the nodes into 8 even partitions with randomly 
          chosen node indexes **


```config:
   ...
     subset_partion:
       partition_type: EVEN
       order: RANDOM
       number_partitions: 8
   ...

```
        - Of course you could pair EVEN with SEQUENTIAL and WEIGHTED with RANDOM
          too
        - Now to specify that a range or percent selection should be taken with 
          respect to a subset simply include the subset field in the top level of 
          the selection map and specify in this field a list of the desired headers

        **Example 7: Run a command on 10 random nodes of the 3rd partition and 10
          random nodes of the 4th partition **

```selection:
     subset: [3,4]
     range:
         order: RANDOM
         number: 10

```


        **Example 8: Run a command on 50 percent of the nodes on the 2nd partition
          starting from the 1st node of the 2nd partition **

```selection: 
     subset: [2]
     percent:
         order: SEQUENTIAL
         start: 1
         percent: 50
```
