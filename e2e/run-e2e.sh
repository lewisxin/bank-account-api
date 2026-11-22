#!/bin/sh
set -eu

compose_command=$1
postman_cli=$2
if ! command -v "$postman_cli" >/dev/null; then
    printf 'Postman CLI not found: %s\n' "$postman_cli" >&2
    exit 1
fi

run_dir=$(mktemp -d "${TMPDIR:-/tmp}/bank-account-api-e2e.XXXXXXXX")
project=$(basename "$run_dir" | tr '[:upper:].' '[:lower:]-')
compose="$compose_command --env-file /dev/null -p $project -f docker-compose.yml -f e2e/docker-compose.e2e.yml"
export DATABASE_URL=

cleanup() {
    result=$?
    trap - EXIT INT TERM
    $compose down --volumes || result=1
    rmdir "$run_dir" || result=1
    exit "$result"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

make up COMPOSE="$compose" DATABASE_URL=
address=$($compose port bank-account-api 8080)

# The account route answers GET with 405 once the server starts listening.
status=$(curl --noproxy '*' --retry 5 --retry-connrefused --retry-delay 1 \
    --max-time 2 --silent --show-error --output /dev/null --write-out '%{http_code}' \
    "http://$address/account")
if [ "$status" != 405 ]; then
    printf 'Expected the API at %s to return 405 for GET /account, got %s\n' "$address" "$status" >&2
    exit 1
fi

"$postman_cli" collection run e2e/account_e2e.postman_collection.json \
    --env-var "baseUrl=http://$address" --no-report-events
