#!/bin/bash

set -exo pipefail

export DOCKER_MYSQL_NAME="mysql-for-wechat"
export DOCKER_WECHAT_SERVER="wechat"
export DOCKER_NETWORK="wechatService"

export CONFIG_FILE_FOLDER="/home/ccloud/wechat"
export CONFIG_FILE_NAME="config_wechat.json"
export CONFIG_FILE_PATH="${CONFIG_FILE_FOLDER}/${CONFIG_FILE_NAME}"

mkdir -p "${CONFIG_FILE_FOLDER}"

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

  source ./deploy/deploy_db.sh true
  source ./deploy/deploy_wechat_server.sh true
else
  echo "use docker_compose"
  make docker_compose
fi



