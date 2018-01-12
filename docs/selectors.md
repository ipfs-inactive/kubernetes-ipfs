# Selectors

## Label Selectors in Kubernetes for test run selection

Sometimes, we want to run a particular test only against certain subsets of pods in the Kubernetes cluster (as we may be running multiple simultaneous tests together).

For this purpose, one can make use of the `selector` attribute under a test definition. This follows the kubernetes format of `key=value`, where some of the most common are the `run` key, which indicates a grouping of pods that are part of a particular rungroup.

In this way, one can shape their tests against only go-ipfs or js-ipfs implementations, only ipfs-cluster deployments, only low-bandwidth pods, etc.

```
config:
  selector: run=go-ipfs-stress
```
