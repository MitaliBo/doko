#!/bin/bash

docker build -t doko-demo-service demo-service

docker run --network host --rm -d --name doko-demo-service doko-demo-service
