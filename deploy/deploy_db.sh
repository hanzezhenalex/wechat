#!/bin/bash

set -exo pipefail

PRODUCTION_MODE=$1
DOCKER_MYSQL_NAME='mysql-for-wechat'

if [ -z "${PRODUCTION_MODE}" ]; then
  docker run -d -p 3306:3306 --name "${DOCKER_MYSQL_NAME}" \
    -e MYSQL_ROOT_PASSWORD=sergey \
    -e MYSQL_DATABASE=wechat \
    -e MYSQL_USER=sergey \
    -e MYSQL_PASSWORD=sergey \
    mysql/mysql-server:latest
else
  echo "production mode"
  docker run -d --rm --name "${DOCKER_MYSQL_NAME}" \
    -e MYSQL_ROOT_PASSWORD=sergey \
    -e MYSQL_DATABASE=wechat \
    -e MYSQL_USER=sergey \
    -e MYSQL_PASSWORD=sergey \
    -v /home/mysql/data:/var/lib/mysql \
    --network "${DOCKER_NETWORK}" \
    mysql/mysql-server:latest
fi

sleep 5 # waiting for mysql

source ./deploy/db.sh

docker_mysql_id=$(docker ps | grep "${DOCKER_MYSQL_NAME}" | awk '{print $1}')
if [ -z "${docker_mysql_id}" ]; then
  echo "mysql is not working"
  exit 1
fi
echo "mysql container id: ${docker_mysql_id}"

# set user
if [ -z "${MYSQL_ROOT_USER}" ]; then
  MYSQL_ROOT_USER="root"
fi
if [ -z "${MYSQL_ROOT_PASSWORD}" ]; then
  MYSQL_ROOT_PASSWORD="sergey"
fi
if [ -z "${MYSQL_USER}" ]; then
  MYSQL_USER="sergey"
fi

docker_mysql_db="wechat"

# prepare db
echo "create database ${docker_mysql_db}"
docker exec "${docker_mysql_id}" mysql -u"${MYSQL_ROOT_USER}" -p"${MYSQL_ROOT_PASSWORD}" \
 -e "CREATE DATABASE IF NOT EXISTS ${docker_mysql_db};"

echo "grant full privilege for ${MYSQL_USER} on wechat"
docker exec "${docker_mysql_id}" mysql -u"${MYSQL_ROOT_USER}" -p"${MYSQL_ROOT_PASSWORD}" \
  -e "GRANT ALL PRIVILEGES ON ${docker_mysql_db}.* TO '${MYSQL_USER}'@'%';"