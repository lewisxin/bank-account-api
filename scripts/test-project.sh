prepare_test_project() {
    test_suite=$1
    test_compose_command=$2
    test_project_dir=$(mktemp -d "${TMPDIR:-/tmp}/bank-account-api-${test_suite}.XXXXXXXX")
    test_project=$(basename "$test_project_dir" | tr '[:upper:].' '[:lower:]-')
    test_compose="$test_compose_command --env-file /dev/null -p $test_project -f docker-compose.yml -f docker-compose.test.yml --profile tools"

    trap cleanup_test_project EXIT
    trap 'exit 130' INT
    trap 'exit 143' TERM
}

cleanup_test_project() {
    result=$?
    trap - EXIT INT TERM
    $test_compose down --volumes || result=1
    rmdir "$test_project_dir" || result=1
    exit "$result"
}
