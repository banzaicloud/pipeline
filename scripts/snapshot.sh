#!/bin/bash

set -euf

CLI_CONFIG_DIR="${HOME}/.circleci"
CLI_CONFIG="${CLI_CONFIG_DIR}/cli.yml"
TOKEN_FILE="${CLI_CONFIG_DIR}/token"
PROJECT_SLUG='gh/banzaicloud/pipeline'

function read_token_from_stdin()
{
    mkdir -p "${CLI_CONFIG_DIR}"
    read -p 'CircleCi API token (https://circleci.com/account/api): ' TOKEN_INPUT
    echo "${TOKEN_INPUT}" > "${TOKEN_FILE}"
    echo "${TOKEN_INPUT}"
}

function get_token()
{
    if [ ! -z "${CIRCLE_TOKEN}" ]; then
        echo "${CIRCLE_TOKEN}"
    elif [ -f "${TOKEN_FILE}" ]; then
        cat "${TOKEN_FILE}"
    elif [ -f "${CLI_CONFIG}" ]; then
        grep 'token: ' "${CLI_CONFIG}" | cut -d ' ' -f 2
    else
        read_token_from_stdin
    fi
}

function check_dirty()
{
    if [ -n "$(git diff --stat)" ]; then
        echo '[ERROR] git is currently in a dirty state'
        exit 1
    fi
}

function check_branch()
{
    local branch="$1"

    if [ "${branch}" = 'HEAD' ]; then
        echo '[ERROR] Cannot run on detached HEAD'
        exit 1
    fi
}


function main()
{
    check_dirty

    local token="$(get_token)"
    local gitref="$(git rev-parse --short HEAD)"
    local default_version="snapshot-${gitref}"
    local default_config_branch="$(git rev-parse --abbrev-ref HEAD)"

    read -p "CircleCI config branch [${default_config_branch}]: " config_branch
    config_branch="${config_branch:-${default_config_branch}}"
    check_branch "${config_branch}"

    read -p "Snapshot version [${default_version}]: " version
    version="${version:-${default_version}}"

    curl \
        -u "${token}:" \
        -X POST \
        --header "Content-Type: application/json" \
        -d "{
            \"branch\": \"${config_branch}\",
            \"parameters\": {
                \"snapshot\": true,
                \"gitref\": \"${gitref}\",
                \"snapshot_version\": \"${version}\"
            }
        }" "https://circleci.com/api/v2/project/${PROJECT_SLUG}/pipeline"
}

main "$@"
