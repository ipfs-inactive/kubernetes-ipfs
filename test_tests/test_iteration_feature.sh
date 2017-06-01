#!/bin/bash

go install ..
test $? -eq 0 || exit 1
echo "Running iteration tests"
for t in iteration_feature/*.yml; do
    kubernetes-ipfs $t
    if [ $? -ne 0 ]; then
        exit 1
    fi
done
