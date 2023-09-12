#!/bin/bash

set -exo pipefail

DOCKER_MYSQL_NAME="mysql-for-wechat"
WECHAT_SERVER_NAME="wechat"

if [ -z "${USE_DOCKER_COMPOSE}" ]; then
  echo "deploy by docker"

  docker run -d --name "${DOCKER_MYSQL_NAME}" \
    -e MYSQL_ROOT_PASSWORD=sergey \
    -e MYSQL_DATABASE=wechat \
    -e MYSQL_USER=sergey \
    -e MYSQL_PASSWORD=sergey \
    -v /home/mysql/data:/var/lib/mysql \
    mysql/mysql-server:latest

  make docker_wechat_server

  echo "
{
  \"token\": \"sdaregsghsd\",
  \"app_id\": \"wxa1e850de1191bd56\",
  \"app_secret\": \"3c87533a8b1902e37d08c5f60106bfe9\",
  \"host\":     \"localhost\",
  \"port\":     3306,
  \"username\": \"sergey\",
  \"password\": \"sergey\"
}" > './config_wechat.json'

  docker run -d --name "${WECHAT_SERVER_NAME}" -p 8096:8096 \
    --network container:"${DOCKER_MYSQL_NAME}" \
    alex/wechat_server:latest -c './config_wechat.json'

else
  echo "use docker_compose"

  echo "
{
  \"token\": \"sdaregsghsd\",
  \"app_id\": \"wxa1e850de1191bd56\",
  \"app_secret\": \"3c87533a8b1902e37d08c5f60106bfe9\",
  \"host\":     \"mysql\",
  \"port\":     3306,
  \"username\": \"sergey\",
  \"password\": \"sergey\"
}" > '/home/wechat/config.json'

  make docker_compose
fi



