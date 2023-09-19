#!/bin/bash

set -euxo pipefail

CONFIG_FILE_PATH='/usr/app/config_wechat.json'

echo "
{
  \"token\": \"${TOKEN}\",
  \"app_id\": \"${APP_ID}\",
  \"app_secret\": \"${APP_SECRET}\",
  \"host\":     \"${MYSQL_HOST}\",
  \"port\":     \"3306\",
  \"username\": \"${MYSQL_USER}\",
  \"password\": \"${MYSQL_PASSWORD}\"
}" > "${CONFIG_FILE_PATH}"

/usr/bin/wechat --config "${CONFIG_FILE_PATH}"