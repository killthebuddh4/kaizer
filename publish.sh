#!/bin/sh

source ./env.prod.sh

docker build --platform linux/amd64 -t ${KAIZER_DOCKER_REGISTRY}/kaizer:latest .

aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin ${KAIZER_DOCKER_REGISTRY}

docker push ${KAIZER_DOCKER_REGISTRY}/kaizer:latest