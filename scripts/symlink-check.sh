#!/usr/bin/env bash

# symlink
if [[ -n "${FILES}" ]]; then
  echo "✖ make clean_vendor needs to be run"
  exit 1
fi


