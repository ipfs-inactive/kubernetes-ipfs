# Test config

Sometimes, we want to scale up or down our tests to compare against the size of a given architecture.

To this end, we can make use of the config.yml to define certain constants we wish to make use of. For an example of how to use this in a real test, see `config-writer.sh`, which creates the config.yml file from the user's input to automatically scale the tests.

```
params:
  N: 5
```

Then, in the test definition:

```
config:
  nodes: {{N}}
```
