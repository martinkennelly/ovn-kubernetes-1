#!/bin/bash

while true; do
 cd contrib/
 ./kind.sh --enable-interconnect
 cd -
 cd test/e2e
 go test -test.v -race . -ginkgo.v -ginkgo.failFast --num-nodes=2
 cd -
done
