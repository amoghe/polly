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
    --silent \
    --digest --user "$GERRIT_ADMIN_USER:$GERRIT_ADMIN_PASS" \
    --header "Content-Type: application/json" \
    -d@$SEED_DIR/$seedfile \
    http://localhost:8080/a/$api_path
}

# main driver function
function main() {
    echo "Adding team (leads) group"
    PUT_api_call "team-leads.group.json" "groups/team-leads"

    echo "Adding team (members) group"
    PUT_api_call "team-members.group.json" "groups/team-members"

    echo "Changing admin password"
    PUT_api_call "admin.password.json" "accounts/self/password.http"
}

#
# main
#
main
