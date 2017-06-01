package main

import (
  "testing"
  "os"
  "path/filepath"
  "fmt"
  "errors"
  "github.com/fatih/color"
)
/* First key: filename, second key: step num 
   data is an array of expected indices, or [-1]*(expected number)*/
var expected map[string]map[int][]int 

func visitBad(path string, f os.FileInfo, err error) error {
  var test Test
  var subsetPartition map[int][]int
  if err != nil {
    return err
  }
  if f.IsDir() {
    return nil /* skip dir, not a test file */
  }
  err = readTestFile(path, &test)
  if err != nil {
    return err
  }
  /* This should always error */
  err = validate(&test, &subsetPartition)
  if err == nil {
    return errors.New(fmt.Sprintf("Bad input file %s incorrectly validates", path))
  }
  return nil
}

func TestBadSelectionFiles(t *testing.T) {
  testDir := "test_tests/selection_framework/failtests"
  err := filepath.Walk(testDir, visitBad)
  if err != nil {
    t.Error(err.Error())
  }
}

func visit(path string, f os.FileInfo, err error) error {
  var test Test
  var subsetPartition map[int][]int
  if err != nil {
    return err
  }
  if f.IsDir() {
    return nil /* skip dir, not a test file */
  }
  err = readTestFile(path, &test)
  if err != nil {
    return err
  }
  /* This should validate correctly */
  err = validate(&test, &subsetPartition)
  if err != nil {
    return err
  }

  /* Run through steps and check that indices are as expected */
  color.Cyan("!!! Running test file %s", path)
  for i, step := range test.Steps {
    expectedIndices := expected[path][i]
    actualIndices := selectNodes(step, test.Config, subsetPartition)
    color.Blue("### Running step %s on nodes %v", step.Name, actualIndices)
    color.Cyan("### Expecting nodes %v", expectedIndices)
    if len(actualIndices) == 0 {
      return errors.New(fmt.Sprintf("Test %s step %d not running on any nodes", path, i))
    }
    if expectedIndices[0] < 0 { /* Indicates random selection and value doesn't matter */
      if len(actualIndices) != len(expectedIndices) {
        return errors.New(fmt.Sprintf("Test %s step %d running on the wrong number of nodes", path, i))
      }
    } else { /* Equality should be exact */
      if !slicesEqual(actualIndices, expectedIndices) {
        return errors.New(fmt.Sprintf("Test %s step %d running on the wrong nodes", path, i))
      }
    }
  }
  return nil
}

func TestValidSelectionFiles(t *testing.T) {
  testDir := "test_tests/selection_framework/succeedtests"
  /* Set up the map of expected indices for comparison */
  expected = make(map[string]map[int][]int)

  expected[testDir + "/old_range.yml"] = make(map[int][]int)
  expected[testDir + "/old_range.yml"][0] = []int{2,3,4}

  expected[testDir +"/percent.yml"] = make(map[int][]int)
  expected[testDir +"/percent.yml"][0] = []int{1,2}
  expected[testDir +"/percent.yml"][1] = []int{3,4}
  expected[testDir +"/percent.yml"][2] = []int{-1,-1,-1}
  expected[testDir +"/percent.yml"][3] = []int{-1,-1}

  expected[testDir +"/rand_even_subset.yml"] = make(map[int][]int)
  expected[testDir +"/rand_even_subset.yml"][0] = []int{-1,-1,-1}
  expected[testDir +"/rand_even_subset.yml"][1] = []int{-1,-1,-1}

  expected[testDir +"/rand_weighted_subset.yml"] = make(map[int][]int)
  expected[testDir +"/rand_weighted_subset.yml"][0] = []int{-1,-1}
  expected[testDir +"/rand_weighted_subset.yml"][1] = []int{-1,-1}
  expected[testDir +"/rand_weighted_subset.yml"][2] = []int{-1}

  expected[testDir +"/range.yml"] = make(map[int][]int)
  expected[testDir +"/range.yml"][0] = []int{2,3,4}
  expected[testDir +"/range.yml"][1] = []int{-1,-1,-1}
  expected[testDir +"/range.yml"][2] = []int{-1,-1,-1,-1}

  expected[testDir +"/seq_even_subset.yml"] = make(map[int][]int)
  expected[testDir +"/seq_even_subset.yml"][0] = []int{1,3,5}

  expected[testDir +"/seq_weighted_subset.yml"] = make(map[int][]int)
  expected[testDir +"/seq_weighted_subset.yml"][0] = []int{1,2}
  expected[testDir +"/seq_weighted_subset.yml"][1] = []int{1,2,3,4}
  expected[testDir +"/seq_weighted_subset.yml"][2] = []int{5}

  expected[testDir +"/seq_weighted_subset_trunc.yml"] = make(map[int][]int)
  expected[testDir +"/seq_weighted_subset_trunc.yml"][0] = []int{1,2,3}
  expected[testDir +"/seq_weighted_subset_trunc.yml"][1] = []int{1,2,3,4}
  expected[testDir +"/seq_weighted_subset_trunc.yml"][2] = []int{5}

  expected[testDir +"/range_subset.yml"] = make(map[int][]int)
  expected[testDir +"/range_subset.yml"][0] = []int{-1,-1,-1}
  expected[testDir +"/range_subset.yml"][1] = []int{5}
  expected[testDir +"/range_subset.yml"][2] = []int{2,4}
  expected[testDir +"/range_subset.yml"][3] = []int{1,2,3,4}
  expected[testDir +"/range_subset.yml"][4] = []int{-1,-1,-1,-1}
  expected[testDir +"/range_subset.yml"][5] = []int{1,3,5}

  expected[testDir +"/percent_subset.yml"] = make(map[int][]int)
  expected[testDir +"/percent_subset.yml"][0] = []int{-1,-1,-1,-1,-1}
  expected[testDir +"/percent_subset.yml"][1] = []int{3,4}
  expected[testDir +"/percent_subset.yml"][2] = []int{-1,-1}
  expected[testDir +"/percent_subset.yml"][3] = []int{-1}

  err := filepath.Walk(testDir, visit)
  if err != nil {
    t.Error(err.Error())
  }
}


func slicesEqual(s1, s2 []int) bool {
  if len(s1) != len(s2) {
    return false
  }
  for j := 0; j < len(s1); j++ {
    if s1[j] != s2[j] {
      return false
    }
  }
  return true
}


