#!/bin/bash

###
docker container stop $(docker ps -a | grep wechat-server | awk '{print $1}')
docker container rm $(docker ps -a | grep wechat-server | awk '{print $1}')

docker container stop $(docker ps -a | grep mysql | awk '{print $1}')

docker network rm wechatService