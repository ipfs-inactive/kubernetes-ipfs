# Test config

Sometimes, we want to scale up or down our tests to compare against the size of a given architecture.

To this end, we can use config.yml to define certain constants we wish to make use of. For an example of how to use this in a real test, see `config-writer.sh`, which creates the config.yml file from the user's input to automatically scale the tests, both by scaling the number of nodes the tests run on as well as scaling the number of pins we make during a test.

In general, the config is available for substituting variables across tests.

To use the config, the user can do one of two things:

- A config file will be automatically found and used if named `config.yml` and in the same directory as the tests
- Specify the path to the config using the flag `--config`.

```
params:
  N: 5
```

Then, in the test definition:

```
config:
  nodes: {{N}}
```
