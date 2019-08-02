#!/bin/bash

docker build -t docons-demo-service demo-service

docker run --network host --rm -d --name docons-demo-service docons-demo-service
