set -euxo pipefail

MYSQL_ROOT_USER='root'

# prepare db
echo "create database ${MYSQL_DATABASE}"
mysql -h "${MYSQL_HOST}" -u"${MYSQL_ROOT_USER}" -p"${MYSQL_ROOT_PASSWORD}" \
 -e "CREATE DATABASE IF NOT EXISTS ${MYSQL_DATABASE};"

echo "grant full privilege for ${MYSQL_USER} on wechat"
mysql -h "${MYSQL_HOST}" -u"${MYSQL_ROOT_USER}" -p"${MYSQL_ROOT_PASSWORD}" \
  -e "GRANT ALL PRIVILEGES ON ${MYSQL_DATABASE}.* TO '${MYSQL_USER}'@'%';"