#!/bin/bash

set -euxo pipefail

make docker_compose

source ./deploy/db.sh