#!/bin/sh

source ./env.prod.sh

aws ecr-public get-login-password --region us-east-1 | DOCKER_HOST=${KAIZER_HOST} docker login --username AWS --password-stdin ${KAIZER_DOCKER_REGISTRY}

DOCKER_HOST=${KAIZER_HOST} docker compose up -d