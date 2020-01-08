#!/bin/bash
set -ev

go install ./cmd/registry
go install ./cmd/crane

registry &
PID=$!

crane pull debian:latest debianc.tar
crane push debianc.tar localhost:1338/debianc:latest
docker pull localhost:1338/debianc:latest
docker tag localhost:1338/debianc:latest localhost:1338/debiand:latest
docker push localhost:1338/debiand:latest
crane pull localhost:1338/debiand:latest debiand.tar

docker pull ubuntu:latest
docker tag ubuntu:latest localhost:1338/ubuntud:latest
docker push localhost:1338/ubuntud:latest
crane pull localhost:1338/ubuntud:latest ubuntu.tar
crane push ubuntu.tar localhost:1338/ubuntuc:foo
docker pull localhost:1338/ubuntuc:foo

kill $PID
