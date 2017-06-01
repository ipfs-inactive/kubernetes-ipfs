#!/bin/bash

go install ..
test $? -eq 0 || exit 1
echo "Testing that command runs on correct nodes as specified by selection"
for t in selection_framework/succeedtests/*.yml; do
    kubernetes-ipfs $t
    if [ $? -ne 0 ]; then
        exit 1
    fi
done
echo "Now testing for failures on invalid yml selection"
for t in selection_framework/failtests/*.yml; do
    kubernetes-ipfs $t
    if [ $? -eq 0 ]; then
        echo "Unexpected pass of bad input format"
        exit 1
    fi
done
echo "Successfully failed all bad inputs"
