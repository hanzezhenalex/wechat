#!/bin/bash

set -exo pipefail

# find mysql container
if [ -z "${DOCKER_MYSQL_NAME}" ]; then
  DOCKER_MYSQL_NAME="mysql-for-wechat"
fi
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