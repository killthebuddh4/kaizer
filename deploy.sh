#!/bin/sh

aws ecr-public get-login-password --region us-east-1 | DOCKER_HOST=ssh://kaizer docker login --username AWS --password-stdin public.ecr.aws/t2b0u5z3

DOCKER_HOST=ssh://kaizer docker compose up -d