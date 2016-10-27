#!/bin/bash

# Set these env vars to change how the script behaves
GERRIT_ADMIN_USER="admin"
GERRIT_ADMIN_PASS="secret"
SEED_DIR="/home/gerrit/site/seed"

# Fail fast
set -x

# PUT the specified JSON ($1) to the specified endpoint ($2)
function PUT_api_call() {
  bodyfile=$1
  api_path=$2

  if [[ -z "$bodyfile" ]]; then
	   echo "body file not specified"
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
    -d@$SEED_DIR/$bodyfile \
    http://localhost:8080/a/$api_path
}

# POST the specified JSON ($1) to the specified endpoint ($2)
function POST_api_call() {
  bodyfile=$1
  api_path=$2

  if [[ -z "$bodyfile" ]]; then
	   echo "body file not specified"
     exit 1
  fi

  if [[ -z "$api_path" ]]; then
	   echo "API path not specified"
     exit 1
  fi

  curl -X POST \
    --silent \
    --digest --user "$GERRIT_ADMIN_USER:$GERRIT_ADMIN_PASS" \
    --header "Content-Type: plain/text" \
    -d@$SEED_DIR/$bodyfile \
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

    echo "Adding ssh keys for admin"
    POST_api_call "/home/admin/.ssh/id_rsa.pub" "accounts/self/sshkeys"
}

#
# main
#
main
