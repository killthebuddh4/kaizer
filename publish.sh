#!/bin/sh

docker build --platform linux/amd64 -t public.ecr.aws/t2b0u5z3/kaizer:latest .

aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin public.ecr.aws/t2b0u5z3

docker push public.ecr.aws/t2b0u5z3/kaizer:latest