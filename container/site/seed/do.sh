#!/bin/bash

# Set these env vars to change how the script behaves
GERRIT_ADMIN_USER="admin"
GERRIT_ADMIN_PASS="secret"
SEED_DIR="/home/gerrit/site/seed"

# Fail fast
set -x

# PUT the specified JSON ($1) to the specified endpoint ($2)
function PUT_api_call() {
    seedfile=$1
    api_path=$2

    if [[ -z "$seedfile" ]]; then
	echo "seed file not specified"
	exit 1
    fi

    if [[ -z "$api_path" ]]; then
	echo "API path not specified"
	exit 1
    fi

    curl -X PUT \
	 --digest --user "$GERRIT_ADMIN_USER:$GERRIT_ADMIN_PASS" \
	 --header "Content-Type: application/json" \
	 -d@$SEED_DIR/$seedfile \
	 http://localhost:8080/a/$api_path
}

# main driver function
function main() {
    echo "Adding default group"
    PUT_api_call "default_group.json" "groups/world"

    echo "Changing admin password"
    PUT_api_call "admin_password.json" "accounts/self/password.http"
}

#
# main
#
main
