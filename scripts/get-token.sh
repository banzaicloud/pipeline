#!/bin/bash

# To get a Pipeline token in exchange for a Github access token, only for developer mode.

set -euo pipefail

curl -f -s http://localhost:9090/auth/github/callback\?access_token\=$GITHUB_TOKEN > /dev/null
curl -f -s -X POST http://localhost:9090/auth/tokens\?access_token\=$GITHUB_TOKEN | jq -r .token
