#!/bin/bash
set -euo pipefail

id=$(sudo VBoxControl --nologo guestproperty get label | grep Value | cut -d' ' -f2)
owner=$(sudo VBoxControl --nologo guestproperty get owner | grep Value | cut -d' ' -f2)
repo=$(sudo VBoxControl --nologo guestproperty get repo | grep Value | cut -d' ' -f2)
api_token=$(sudo VBoxControl --nologo guestproperty get token | grep Value | cut -d' ' -f2)
token=$(curl -u ${owner}:${token} -X POST -H "Accept: application/vnd.github.v3+json" \
  https://api.github.com/repos/${owner}/${repo}/actions/runners/registration-token | jq -r .token)
if [[ "${id}" != "" ]] && [[ "${token}" != "" ]]; then
  cd ~/actions-runner
  ./config.sh --url https://github.com/${owner}/${repo} --token ${token} --name ${id} --labels ${id} --unattended
  ./run.sh
fi