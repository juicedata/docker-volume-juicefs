#!/bin/bash

make
make enable
source test/verify.sh
if [ "${TRAVIS_PULL_REQUEST}" == "false" ] && [ "${TRAVIS_BRANCH}" == "master" ]
then
    docker login --username ${DOCKER_USERNAME} --password ${DOCKER_PASSWORD}
    make push
fi
