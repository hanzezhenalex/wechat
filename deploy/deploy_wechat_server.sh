set -exo pipefail

DOCKER_WECHAT_SERVER="wechat-server"
PRODUCTION_MODE=$1

if [ -z "${CONFIG_FILE_FOLDER}" ]; then
  CONFIG_FILE_FOLDER="/home/ccloud/wechat"
fi
if [ -z "${CONFIG_FILE_NAME}" ]; then
  CONFIG_FILE_NAME="config_wechat.json"
fi
if [ -z "${DOCKER_NETWORK}" ]; then
  DOCKER_NETWORK="wechatService"
fi

make docker_wechat_server

if [ -z "${PRODUCTION_MODE}" ]; then
  # debug mode
  docker run -d --rm --name "${DOCKER_WECHAT_SERVER}" -p 8096:8096 alex/wechat_server:latest
else
  docker run -d --name "${DOCKER_WECHAT_SERVER}" -p 8096:8096 \
      --network "${DOCKER_NETWORK}" \
      -v "${CONFIG_FILE_FOLDER}":/usr/app/wechat \
      alex/wechat_server:latest --config "/usr/app/wechat/${CONFIG_FILE_NAME}"
fi