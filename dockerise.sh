#!/bin/bash
VERSION=$(git tag --sort=v:refname|tail -n 1)
docker build -t "dolittle/edgeos-updatesite:$VERSION" -f Source/Dockerfile .
docker push "dolittle/edgeos-updatesite:$VERSION"