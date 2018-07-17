#!/bin/bash

set -o pipefail

go list ./... | xargs -n1 go test -v -parallel 1 2>&1 | tee test.txt
