#!/bin/bash

docker build -t doko-demo-service demo-service

docker run --network host --rm -d --name doko-demo-service1 doko-demo-service
docker run -p 8081 --rm -d --name doko-demo-service2 doko-demo-service
docker run -p 8081 --rm -d --name doko-demo-service3 doko-demo-service
