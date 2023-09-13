#!/bin/bash

set -exo pipefail

export DOCKER_MYSQL_NAME="mysql-for-wechat"
WECHAT_SERVER_NAME="wechat"
DOCKER_NETWORK="wechatService"

CONFIG_FILE_FOLDER="/home/ccloud/wechat"
CONFIG_FILE_NAME="config_wechat.json"
CONFIG_FILE_PATH="${CONFIG_FILE_FOLDER}/${CONFIG_FILE_NAME}"
mkdir -p "${CONFIG_FILE_FOLDER}"

# echo "clean up"
# docker container stop $(docker ps | grep "${DOCKER_MYSQL_NAME}" | awk '{print $1}')
# docker container stop $(docker ps | grep "${WECHAT_SERVER_NAME}" | awk '{print $1}')
# docker container rm $(docker ps -a | grep "${WECHAT_SERVER_NAME}" | awk '{print $1}')
# docker network rm $(docker network ls | grep "${DOCKER_NETWORK}" | awk '{print $1}')

  echo "
{
  \"token\": \"sdaregsghsd\",
  \"app_id\": \"wxa1e850de1191bd56\",
  \"app_secret\": \"3c87533a8b1902e37d08c5f60106bfe9\",
  \"host\":     \"${DOCKER_MYSQL_NAME}\",
  \"port\":     3306,
  \"username\": \"sergey\",
  \"password\": \"sergey\"
}" > "${CONFIG_FILE_PATH}"

if [ -z "${USE_DOCKER_COMPOSE}" ]; then
  echo "deploy by docker"

  docker network create "${DOCKER_NETWORK}"

  docker run -d --rm --name "${DOCKER_MYSQL_NAME}" \
    -e MYSQL_ROOT_PASSWORD=sergey \
    -e MYSQL_DATABASE=wechat \
    -e MYSQL_USER=sergey \
    -e MYSQL_PASSWORD=sergey \
    -v /home/mysql/data:/var/lib/mysql \
    --network "${DOCKER_NETWORK}" \
    mysql/mysql-server:latest

  sleep 5 # waiting for mysql

  source ./deploy/db.sh

  make docker_wechat_server

  docker run -d --name "${WECHAT_SERVER_NAME}" -p 8096:8096 \
    --network "${DOCKER_NETWORK}" \
    -v "${CONFIG_FILE_FOLDER}":/usr/app/wechat \
    alex/wechat_server:latest --config "/usr/app/wechat/${CONFIG_FILE_NAME}"

else
  echo "use docker_compose"
  make docker_compose
fi



