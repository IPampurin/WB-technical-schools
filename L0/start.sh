#!/bin/bash

docker-compose -f ./kafka/compose.yml up -d
docker-compose -f ./producer/compose.yml up -d
docker-compose -f ./service/compose.yml up -d
docker-compose -f ./consumer/compose.yml up -d
docker-compose -f ./redisCache/compose.yml up -d