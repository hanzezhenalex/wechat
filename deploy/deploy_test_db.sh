#!/bin/bash

set -euxo pipefail

export DOCKER_MYSQL_NAME='mysql-for-wechat-test'

docker run -d -p 3306:3306 --name "${DOCKER_MYSQL_NAME}" \
  -e MYSQL_ROOT_PASSWORD=sergey \
  -e MYSQL_DATABASE=wechat \
  -e MYSQL_USER=sergey \
  -e MYSQL_PASSWORD=sergey \
  mysql/mysql-server:latest

set +u

source ./deploy/db.sh