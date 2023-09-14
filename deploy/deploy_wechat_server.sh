set -exo pipefail

DOCKER_WECHAT_SERVER="wechat"
PRODUCTION_MODE=$1

echo "clean up wechat_server"

containers=$(docker ps | grep "${DOCKER_WECHAT_SERVER}" | awk '{print $1}')

if [ -z "${containers}" ]; then
  echo "no running containers "
else
  docker container stop $(docker ps | grep "${DOCKER_WECHAT_SERVER}" | awk '{print $1}')
fi

containers=$(docker ps -a | grep "${DOCKER_WECHAT_SERVER}" | awk '{print $1}')

if [ -z "${containers}" ]; then
  echo "no stopped containers"
else
  docker container rm $(docker ps -a | grep "${DOCKER_WECHAT_SERVER}" | awk '{print $1}')
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