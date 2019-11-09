#!/bin/bash

docker build -t 830-base --file Dockerfile.base .
if docker run --rm -it 830-base /bin/sh -c "if [ -d '/usr/share/bug/systemd' ]; then echo 'it is a dir'; exit 0; else exit 1; fi"; then
  go run cmd/tartest/main.go
else
  echo 'fail'
  exit 1
fi
