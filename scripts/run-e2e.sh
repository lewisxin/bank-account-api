#!/bin/sh
set -eu

compose_command=$1
postman_cli=$2
if ! command -v "$postman_cli" >/dev/null; then
    printf 'Postman CLI not found: %s\n' "$postman_cli" >&2
    exit 1
fi

. ./scripts/test-project.sh
prepare_test_project e2e "$compose_command"
export DATABASE_URL=

make up COMPOSE="$test_compose" DATABASE_URL=
address=$($test_compose port bank-account-api 8080)

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
