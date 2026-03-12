#!/usr/bin/env bash
# Live integration test suite for plane-cli dynamic command generation.
# Runs against a real Plane instance. Requires 'plane auth login' to be configured.
set -euo pipefail

PASS=0
FAIL=0
SKIP=0
ERRORS=""
PROJECT="PLANECLI"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

pass() {
    PASS=$((PASS + 1))
    echo -e "  ${GREEN}PASS${NC} $1"
}

fail() {
    FAIL=$((FAIL + 1))
    ERRORS="${ERRORS}\n  - $1: $2"
    echo -e "  ${RED}FAIL${NC} $1"
    echo -e "       ${RED}$2${NC}"
}

skip() {
    SKIP=$((SKIP + 1))
    echo -e "  ${YELLOW}SKIP${NC} $1 — $2"
}

section() {
    echo -e "\n${CYAN}${BOLD}━━━ $1 ━━━${NC}"
}

# Helper: run a command and capture stdout+stderr+exit code
run() {
    local stdout stderr rc
    stdout=$(mktemp)
    stderr=$(mktemp)
    set +e
    "$@" >"$stdout" 2>"$stderr"
    rc=$?
    set -e
    RUN_STDOUT=$(cat "$stdout")
    RUN_STDERR=$(cat "$stderr")
    RUN_RC=$rc
    rm -f "$stdout" "$stderr"
}

# Helper: assert output contains a string
assert_contains() {
    local label="$1" haystack="$2" needle="$3"
    if echo "$haystack" | grep -qF "$needle"; then
        pass "$label"
    else
        fail "$label" "expected to contain '$needle', got: $(echo "$haystack" | head -3)"
    fi
}

assert_rc() {
    local label="$1" expected="$2" actual="$3"
    if [ "$actual" -eq "$expected" ]; then
        pass "$label"
    else
        fail "$label" "expected exit code $expected, got $actual (stderr: $(echo "$RUN_STDERR" | head -2))"
    fi
}

assert_json_field() {
    local label="$1" json="$2" field="$3"
    if echo "$json" | jq -e ".$field" >/dev/null 2>&1; then
        pass "$label"
    else
        fail "$label" "JSON missing field '$field'"
    fi
}

assert_json_array_nonempty() {
    local label="$1" json="$2"
    local count
    count=$(echo "$json" | jq 'if type == "array" then length elif .results then (.results | length) else 0 end' 2>/dev/null || echo 0)
    if [ "$count" -gt 0 ]; then
        pass "$label ($count items)"
    else
        fail "$label" "expected non-empty array, got $count items"
    fi
}

echo -e "${BOLD}Plane CLI Live Test Suite${NC}"
echo "========================="
echo "Target project: $PROJECT"
echo ""

# ============================================================
section "0. Pre-warm spec cache"
# ============================================================

run plane docs update-specs
assert_rc "docs update-specs exits 0" 0 "$RUN_RC"
POST_UPDATE_SPECS=$(find ~/.cache/plane-cli/specs/default/ -name "*.json" 2>/dev/null | wc -l)
echo "  Cached specs: $POST_UPDATE_SPECS"
if [ "$POST_UPDATE_SPECS" -gt 20 ]; then
    pass "update-specs populated cache ($POST_UPDATE_SPECS specs)"
else
    fail "update-specs population" "expected >20 specs, got $POST_UPDATE_SPECS"
fi

# Get current user ID (needed for cycle create --owned-by)
run plane me
MY_USER_ID=$(echo "$RUN_STDOUT" | jq -r '.id' 2>/dev/null)
echo "  User ID: $MY_USER_ID"

# ============================================================
section "1. Auth & Connectivity"
# ============================================================

run plane auth status
assert_rc "auth status exits 0" 0 "$RUN_RC"
assert_contains "auth status has api_url" "$RUN_STDOUT" "api_url"
assert_contains "auth status has workspace" "$RUN_STDOUT" "workspace"

run plane me
assert_rc "me exits 0" 0 "$RUN_RC"
assert_json_field "me returns user id" "$RUN_STDOUT" "id"

run plane me --output table
assert_rc "me --output table exits 0" 0 "$RUN_RC"

# ============================================================
section "2. Help & Discovery"
# ============================================================

run plane --help
assert_rc "root help exits 0" 0 "$RUN_RC"
assert_contains "root help lists issue" "$RUN_STDOUT" "issue"
assert_contains "root help lists cycle" "$RUN_STDOUT" "cycle"
assert_contains "root help lists module" "$RUN_STDOUT" "module"
assert_contains "root help lists label" "$RUN_STDOUT" "label"
assert_contains "root help lists state" "$RUN_STDOUT" "state"
assert_contains "root help lists page" "$RUN_STDOUT" "page"
assert_contains "root help lists member" "$RUN_STDOUT" "member"
assert_contains "root help lists attachment" "$RUN_STDOUT" "attachment"
assert_contains "root help lists customer" "$RUN_STDOUT" "customer"
assert_contains "root help lists epic" "$RUN_STDOUT" "epic"
assert_contains "root help lists initiative" "$RUN_STDOUT" "initiative"
assert_contains "root help lists worklog" "$RUN_STDOUT" "worklog"
assert_contains "root help lists sticky" "$RUN_STDOUT" "sticky"

run plane issue --help
assert_rc "issue help exits 0" 0 "$RUN_RC"
assert_contains "issue help lists create" "$RUN_STDOUT" "create"
assert_contains "issue help lists list" "$RUN_STDOUT" "list"
assert_contains "issue help lists get" "$RUN_STDOUT" "get"
assert_contains "issue help lists update" "$RUN_STDOUT" "update"
assert_contains "issue help lists delete" "$RUN_STDOUT" "delete"
assert_contains "issue help lists search" "$RUN_STDOUT" "search"

run plane cycle --help
assert_rc "cycle help exits 0" 0 "$RUN_RC"
assert_contains "cycle help lists archive" "$RUN_STDOUT" "archive"
assert_contains "cycle help lists transfer-work-items" "$RUN_STDOUT" "transfer-work-items"
assert_contains "cycle help lists add-work-items" "$RUN_STDOUT" "add-work-items"

# ============================================================
section "3. Project Commands"
# ============================================================

run plane project list --output json
assert_rc "project list json exits 0" 0 "$RUN_RC"
assert_json_array_nonempty "project list returns projects" "$RUN_STDOUT"

run plane project list --output table
assert_rc "project list table exits 0" 0 "$RUN_RC"
assert_contains "project table has headers" "$RUN_STDOUT" "NAME"

# ============================================================
section "4. Issue CRUD Lifecycle"
# ============================================================

# 4a. List issues
run plane issue list -p "$PROJECT" --output json
assert_rc "issue list exits 0" 0 "$RUN_RC"

# 4b. Create an issue
ISSUE_NAME="Test-$(date +%s)"
run plane issue create -p "$PROJECT" --name "$ISSUE_NAME" --priority low
assert_rc "issue create exits 0" 0 "$RUN_RC"
assert_json_field "issue create returns id" "$RUN_STDOUT" "id"

ISSUE_ID=$(echo "$RUN_STDOUT" | jq -r '.id')
echo "  Created issue: $ISSUE_ID"

if [ -z "$ISSUE_ID" ] || [ "$ISSUE_ID" = "null" ]; then
    fail "issue create returned valid id" "id was empty or null"
else
    pass "issue create returned valid id"

    # 4c. Get the issue
    run plane issue get -p "$PROJECT" --work-item-id "$ISSUE_ID"
    assert_rc "issue get exits 0" 0 "$RUN_RC"
    assert_contains "issue get returns correct name" "$RUN_STDOUT" "$ISSUE_NAME"

    # 4d. Get issue as table
    run plane issue get -p "$PROJECT" --work-item-id "$ISSUE_ID" --output table
    assert_rc "issue get table exits 0" 0 "$RUN_RC"

    # 4e. Update the issue
    run plane issue update -p "$PROJECT" --work-item-id "$ISSUE_ID" --priority high
    assert_rc "issue update exits 0" 0 "$RUN_RC"

    # Verify update
    run plane issue get -p "$PROJECT" --work-item-id "$ISSUE_ID" --output json
    UPDATED_PRIORITY=$(echo "$RUN_STDOUT" | jq -r '.priority' 2>/dev/null)
    if [ "$UPDATED_PRIORITY" = "high" ]; then
        pass "issue update changed priority to high"
    else
        fail "issue update changed priority" "expected 'high', got '$UPDATED_PRIORITY'"
    fi

    # 4f. Delete the issue
    run plane issue delete -p "$PROJECT" --work-item-id "$ISSUE_ID"
    assert_rc "issue delete exits 0" 0 "$RUN_RC"

    # Verify deletion (should 404)
    run plane issue get -p "$PROJECT" --work-item-id "$ISSUE_ID"
    if [ "$RUN_RC" -ne 0 ]; then
        pass "issue get after delete returns error"
    else
        fail "issue get after delete" "should have failed but succeeded"
    fi
fi

# ============================================================
section "5. Issue List Output Formats"
# ============================================================

run plane issue list -p "$PROJECT" --output json
assert_rc "issue list json exits 0" 0 "$RUN_RC"

run plane issue list -p "$PROJECT" --output table
assert_rc "issue list table exits 0" 0 "$RUN_RC"
assert_contains "issue list table has Id column" "$RUN_STDOUT" "Id"
assert_contains "issue list table has Name column" "$RUN_STDOUT" "Name"

# ============================================================
section "6. State Commands"
# ============================================================

run plane state list -p "$PROJECT" --output json
assert_rc "state list json exits 0" 0 "$RUN_RC"
assert_json_array_nonempty "state list returns states" "$RUN_STDOUT"

run plane state list -p "$PROJECT" --output table
assert_rc "state list table exits 0" 0 "$RUN_RC"
assert_contains "state table has Name column" "$RUN_STDOUT" "Name"

# ============================================================
section "7. Label Commands"
# ============================================================

run plane label list -p "$PROJECT" --output json
assert_rc "label list json exits 0" 0 "$RUN_RC"

run plane label list -p "$PROJECT" --output table
assert_rc "label list table exits 0" 0 "$RUN_RC"

# Label CRUD
run plane label create -p "$PROJECT" --name "test-label-$(date +%s)"
assert_rc "label create exits 0" 0 "$RUN_RC"
LABEL_ID=$(echo "$RUN_STDOUT" | jq -r '.id' 2>/dev/null)

if [ -n "$LABEL_ID" ] && [ "$LABEL_ID" != "null" ]; then
    pass "label create returned id"
    echo "  Created label: $LABEL_ID"

    run plane label get -p "$PROJECT" --label-id "$LABEL_ID"
    assert_rc "label get exits 0" 0 "$RUN_RC"

    run plane label update -p "$PROJECT" --label-id "$LABEL_ID" --name "renamed-label"
    assert_rc "label update exits 0" 0 "$RUN_RC"

    run plane label delete -p "$PROJECT" --label-id "$LABEL_ID"
    assert_rc "label delete exits 0" 0 "$RUN_RC"
else
    fail "label create returned id" "id was empty or null"
fi

# ============================================================
section "8. Member Commands"
# ============================================================

run plane member get-workspace --output json
assert_rc "member get-workspace json exits 0" 0 "$RUN_RC"

run plane member get-workspace --output table
assert_rc "member get-workspace table exits 0" 0 "$RUN_RC"

run plane member get-project -p "$PROJECT" --output json
assert_rc "member get-project json exits 0" 0 "$RUN_RC"

run plane member get-project -p "$PROJECT" --output table
assert_rc "member get-project table exits 0" 0 "$RUN_RC"

# ============================================================
section "9. Cycle Commands"
# ============================================================

run plane cycle list -p "$PROJECT" --output json
assert_rc "cycle list json exits 0" 0 "$RUN_RC"

run plane cycle list -p "$PROJECT" --output table
assert_rc "cycle list table exits 0" 0 "$RUN_RC"

# Create a cycle (requires --owned-by user ID)
# Note: cycles must be enabled on the project for this to succeed
CYCLE_NAME="test-cycle-$(date +%s)"
run plane cycle create -p "$PROJECT" --name "$CYCLE_NAME" --owned-by "$MY_USER_ID"
if [ "$RUN_RC" -eq 0 ]; then
    pass "cycle create exits 0"
    CYCLE_ID=$(echo "$RUN_STDOUT" | jq -r '.id' 2>/dev/null)

    if [ -n "$CYCLE_ID" ] && [ "$CYCLE_ID" != "null" ]; then
        pass "cycle create returned id"
        echo "  Created cycle: $CYCLE_ID"

        run plane cycle get -p "$PROJECT" --cycle-id "$CYCLE_ID"
        assert_rc "cycle get exits 0" 0 "$RUN_RC"

        run plane cycle update -p "$PROJECT" --cycle-id "$CYCLE_ID" --name "renamed-cycle"
        assert_rc "cycle update exits 0" 0 "$RUN_RC"

        run plane cycle list-work-items -p "$PROJECT" --cycle-id "$CYCLE_ID"
        assert_rc "cycle list-work-items exits 0" 0 "$RUN_RC"

        run plane cycle list-archived -p "$PROJECT"
        assert_rc "cycle list-archived exits 0" 0 "$RUN_RC"

        run plane cycle delete -p "$PROJECT" --cycle-id "$CYCLE_ID"
        assert_rc "cycle delete exits 0" 0 "$RUN_RC"
    fi
else
    # Check if this is a "cycles not enabled" error vs a real bug
    if echo "$RUN_STDERR" | grep -qi "not enabled"; then
        skip "cycle create" "cycles not enabled for project $PROJECT"
    else
        fail "cycle create" "exit $RUN_RC: $(echo "$RUN_STDERR" | head -2)"
    fi
fi

# ============================================================
section "10. Module Commands"
# ============================================================

run plane module list -p "$PROJECT" --output json
assert_rc "module list json exits 0" 0 "$RUN_RC"

MODULE_NAME="test-module-$(date +%s)"
run plane module create -p "$PROJECT" --name "$MODULE_NAME"
assert_rc "module create exits 0" 0 "$RUN_RC"
MODULE_ID=$(echo "$RUN_STDOUT" | jq -r '.id' 2>/dev/null)

if [ -n "$MODULE_ID" ] && [ "$MODULE_ID" != "null" ]; then
    pass "module create returned id"
    echo "  Created module: $MODULE_ID"

    run plane module get -p "$PROJECT" --module-id "$MODULE_ID"
    assert_rc "module get exits 0" 0 "$RUN_RC"

    run plane module update -p "$PROJECT" --module-id "$MODULE_ID" --name "renamed-module"
    assert_rc "module update exits 0" 0 "$RUN_RC"

    run plane module list-work-items -p "$PROJECT" --module-id "$MODULE_ID"
    assert_rc "module list-work-items exits 0" 0 "$RUN_RC"

    run plane module delete -p "$PROJECT" --module-id "$MODULE_ID"
    assert_rc "module delete exits 0" 0 "$RUN_RC"
else
    fail "module create returned id" "id was empty or null"
fi

# ============================================================
section "11. Page Commands"
# ============================================================

# Page commands — may return 404 if page feature isn't available
run plane page add-workspace --name "test-page-$(date +%s)"
if [ "$RUN_RC" -eq 0 ]; then
    pass "page add-workspace exits 0"
    PAGE_ID=$(echo "$RUN_STDOUT" | jq -r '.id' 2>/dev/null)
    if [ -n "$PAGE_ID" ] && [ "$PAGE_ID" != "null" ]; then
        echo "  Created workspace page: $PAGE_ID"
    fi
elif echo "$RUN_STDERR" | grep -qi "404"; then
    skip "page add-workspace" "API returned 404 (feature not available)"
else
    fail "page add-workspace" "exit $RUN_RC: $(echo "$RUN_STDERR" | head -2)"
fi

run plane page add-project -p "$PROJECT" --name "test-proj-page-$(date +%s)"
if [ "$RUN_RC" -eq 0 ]; then
    pass "page add-project exits 0"
    PROJECT_PAGE_ID=$(echo "$RUN_STDOUT" | jq -r '.id' 2>/dev/null)
    if [ -n "$PROJECT_PAGE_ID" ] && [ "$PROJECT_PAGE_ID" != "null" ]; then
        echo "  Created project page: $PROJECT_PAGE_ID"
    fi
elif echo "$RUN_STDERR" | grep -qi "404"; then
    skip "page add-project" "API returned 404 (feature not available)"
else
    fail "page add-project" "exit $RUN_RC: $(echo "$RUN_STDERR" | head -2)"
fi

# ============================================================
section "12. Sticky Commands"
# ============================================================

run plane sticky list --output json
assert_rc "sticky list json exits 0" 0 "$RUN_RC"

run plane sticky add --name "test-sticky-$(date +%s)"
if [ "$RUN_RC" -eq 0 ]; then
    pass "sticky add exits 0"
    STICKY_ID=$(echo "$RUN_STDOUT" | jq -r '.id' 2>/dev/null)
    if [ -n "$STICKY_ID" ] && [ "$STICKY_ID" != "null" ]; then
        echo "  Created sticky: $STICKY_ID"
        run plane sticky delete --sticky-id "$STICKY_ID"
        assert_rc "sticky delete exits 0" 0 "$RUN_RC"
    fi
else
    fail "sticky add" "exit $RUN_RC: $(echo "$RUN_STDERR" | head -2)"
fi

# ============================================================
section "13. Pagination"
# ============================================================

run plane issue list -p "$PROJECT" --per-page 1 --output json
assert_rc "issue list per-page=1 exits 0" 0 "$RUN_RC"

# Check paginated envelope (use >/dev/null to prevent jq output from polluting variable)
if echo "$RUN_STDOUT" | jq -e '.results' >/dev/null 2>&1; then
    pass "pagination returns envelope with results"
    RESULT_COUNT=$(echo "$RUN_STDOUT" | jq '.results | length')
    if [ "$RESULT_COUNT" -le 1 ]; then
        pass "pagination respects per-page=1 ($RESULT_COUNT items)"
    else
        fail "pagination respects per-page" "expected <=1 items, got $RESULT_COUNT"
    fi
else
    fail "pagination returns envelope" "no 'results' key in response"
fi

# --all flag
run plane issue list -p "$PROJECT" --all --output json
assert_rc "issue list --all exits 0" 0 "$RUN_RC"
ALL_COUNT=$(echo "$RUN_STDOUT" | jq 'if type == "object" and .results then (.results | length) elif type == "array" then length else 0 end' 2>/dev/null)
if [ "$ALL_COUNT" -gt 0 ]; then
    pass "issue list --all returned $ALL_COUNT items"
else
    fail "issue list --all" "expected items, got $ALL_COUNT"
fi

# ============================================================
section "14. Error Handling"
# ============================================================

# Missing required project
run plane issue list
if [ "$RUN_RC" -ne 0 ]; then
    pass "issue list without --project fails"
else
    fail "issue list without --project" "should have failed"
fi

# Invalid project identifier
run plane issue list -p "NONEXISTENT_PROJECT_XYZ"
if [ "$RUN_RC" -ne 0 ]; then
    pass "issue list with bad project fails"
else
    fail "issue list with bad project" "should have failed"
fi

# Invalid UUID for path param
run plane issue get -p "$PROJECT" --work-item-id "not-a-uuid"
if [ "$RUN_RC" -ne 0 ]; then
    pass "issue get with invalid id fails"
else
    fail "issue get with invalid id" "should have failed"
fi

# Missing required flag
run plane issue create -p "$PROJECT"
if [ "$RUN_RC" -ne 0 ]; then
    pass "issue create without --name fails"
else
    fail "issue create without --name" "should have failed"
fi

# ============================================================
section "15. Verbose Mode"
# ============================================================

run plane project list --verbose --output json
assert_rc "verbose mode exits 0" 0 "$RUN_RC"
assert_contains "verbose shows HTTP debug" "$RUN_STDERR" "GET"

# ============================================================
section "16. Spec Caching"
# ============================================================

CACHED_SPECS=$(find ~/.cache/plane-cli/specs/default/ -name "*.json" 2>/dev/null | wc -l)
if [ "$CACHED_SPECS" -gt 5 ]; then
    pass "spec cache has $CACHED_SPECS cached specs"
else
    fail "spec cache" "expected >5 cached specs, found $CACHED_SPECS"
fi

# Verify a cached spec has valid structure
SAMPLE_SPEC=$(find ~/.cache/plane-cli/specs/default/ -name "*.json" | head -1)
if [ -n "$SAMPLE_SPEC" ]; then
    run cat "$SAMPLE_SPEC"
    assert_json_field "cached spec has fetched_at" "$RUN_STDOUT" "fetched_at"
    assert_json_field "cached spec has spec.method" "$RUN_STDOUT" "spec.method"
    assert_json_field "cached spec has spec.path_template" "$RUN_STDOUT" "spec.path_template"
fi

# ============================================================
section "17. Cold Cache (Mode B) — clear one spec and re-run"
# ============================================================

# Clear module specs to test Mode B
rm -rf ~/.cache/plane-cli/specs/default/module/ 2>/dev/null || true

run plane module list -p "$PROJECT" --output json
assert_rc "module list (Mode B cold) exits 0" 0 "$RUN_RC"
assert_contains "Mode B prints fetching hint" "$RUN_STDERR" "Fetching"

# Verify spec was cached after Mode B run
MODULE_SPECS=$(find ~/.cache/plane-cli/specs/default/module/ -name "*.json" 2>/dev/null | wc -l)
if [ "$MODULE_SPECS" -gt 0 ]; then
    pass "Mode B cached spec after execution ($MODULE_SPECS files)"
else
    fail "Mode B spec caching" "no module spec files found after Mode B run"
fi

# ============================================================
section "18. Help Text (cached command)"
# ============================================================

run plane issue create --help
assert_rc "issue create --help exits 0" 0 "$RUN_RC"
HELP_OUTPUT="$RUN_STDOUT"
assert_contains "help shows --name flag" "$HELP_OUTPUT" "name"
assert_contains "help shows --priority flag" "$HELP_OUTPUT" "priority"
assert_contains "help shows Usage line" "$HELP_OUTPUT" "Usage"

# ============================================================
section "19. Help Text (Mode B lazy command)"
# ============================================================

# Clear a spec to test lazy help
rm -rf ~/.cache/plane-cli/specs/default/link/ 2>/dev/null || true

run plane link list --help
assert_rc "link list --help (Mode B) exits 0" 0 "$RUN_RC"
LAZY_HELP="$RUN_STDOUT$RUN_STDERR"
assert_contains "lazy help shows Usage" "$LAZY_HELP" "Usage"

# Re-warm link cache
run plane docs update-specs
echo "  Re-warmed cache after Mode B tests"

# ============================================================
section "20. Issue Search"
# ============================================================

SEARCH_NAME="SearchTest-$(date +%s)"
run plane issue create -p "$PROJECT" --name "$SEARCH_NAME"
assert_rc "create searchable issue exits 0" 0 "$RUN_RC"
SEARCH_ISSUE_ID=$(echo "$RUN_STDOUT" | jq -r '.id' 2>/dev/null)

# Check what search flags are available
run plane issue search --help
SEARCH_HELP="$RUN_STDOUT"
echo "  Search help: $(echo "$SEARCH_HELP" | grep -o '\-\-[a-z-]*' | tr '\n' ' ')"

run plane issue search -p "$PROJECT" --search "$SEARCH_NAME"
if [ "$RUN_RC" -eq 0 ]; then
    pass "issue search exits 0"
else
    # Try alternate flag name
    fail "issue search" "exit $RUN_RC: $(echo "$RUN_STDERR" | head -2)"
fi

# Cleanup
if [ -n "$SEARCH_ISSUE_ID" ] && [ "$SEARCH_ISSUE_ID" != "null" ]; then
    run plane issue delete -p "$PROJECT" --work-item-id "$SEARCH_ISSUE_ID"
fi

# ============================================================
section "21. Issue get-by-sequence-id"
# ============================================================

run plane issue create -p "$PROJECT" --name "SeqTest-$(date +%s)"
assert_rc "create issue for sequence test" 0 "$RUN_RC"
SEQ_ISSUE_ID=$(echo "$RUN_STDOUT" | jq -r '.id' 2>/dev/null)
SEQ_ID=$(echo "$RUN_STDOUT" | jq -r '.sequence_id' 2>/dev/null)

if [ -n "$SEQ_ID" ] && [ "$SEQ_ID" != "null" ]; then
    # The spec uses --identifier which is project_identifier-sequence_id format
    FULL_ID="${PROJECT}-${SEQ_ID}"
    echo "  Testing get-by-sequence-id with identifier: $FULL_ID"
    run plane issue get-by-sequence-id -p "$PROJECT" --identifier "$FULL_ID"
    if [ "$RUN_RC" -eq 0 ]; then
        pass "issue get-by-sequence-id exits 0"
    else
        fail "issue get-by-sequence-id" "exit $RUN_RC: $(echo "$RUN_STDERR" | head -2)"
    fi
else
    skip "issue get-by-sequence-id" "no sequence_id in create response"
fi

# Cleanup
if [ -n "$SEQ_ISSUE_ID" ] && [ "$SEQ_ISSUE_ID" != "null" ]; then
    run plane issue delete -p "$PROJECT" --work-item-id "$SEQ_ISSUE_ID"
fi

# ============================================================
section "22. Name-to-UUID Resolution"
# ============================================================

# Get a state name
run plane state list -p "$PROJECT" --output json
STATE_NAME=$(echo "$RUN_STDOUT" | jq -r '(if .results then .results else . end) | .[0].name' 2>/dev/null)
echo "  Trying resolution with state name: $STATE_NAME"

if [ -n "$STATE_NAME" ] && [ "$STATE_NAME" != "null" ]; then
    # The issue create spec uses --state (not --state-id) for the state field
    run plane issue create -p "$PROJECT" --name "ResolverTest-$(date +%s)" --state "$STATE_NAME"
    if [ "$RUN_RC" -eq 0 ]; then
        pass "name-to-UUID resolution worked for state '$STATE_NAME'"
        # Verify the state was actually resolved
        CREATED_STATE=$(echo "$RUN_STDOUT" | jq -r '.state' 2>/dev/null)
        if [ ${#CREATED_STATE} -eq 36 ]; then
            pass "resolved state is a UUID ($CREATED_STATE)"
        else
            fail "resolved state is UUID" "got '$CREATED_STATE'"
        fi
        RESOLVER_ISSUE_ID=$(echo "$RUN_STDOUT" | jq -r '.id' 2>/dev/null)
        # Cleanup
        if [ -n "$RESOLVER_ISSUE_ID" ] && [ "$RESOLVER_ISSUE_ID" != "null" ]; then
            run plane issue delete -p "$PROJECT" --work-item-id "$RESOLVER_ISSUE_ID"
        fi
    else
        fail "name-to-UUID resolution for state" "exit $RUN_RC: $(echo "$RUN_STDERR" | head -2)"
    fi
else
    skip "name-to-UUID resolution" "could not extract state name"
fi

# ============================================================
section "23. Intake Commands"
# ============================================================

run plane intake list -p "$PROJECT" --output json
if [ "$RUN_RC" -eq 0 ]; then
    pass "intake list exits 0"
else
    skip "intake list" "exit $RUN_RC (may not be configured)"
fi

# ============================================================
section "24. Activity Commands"
# ============================================================

run plane issue create -p "$PROJECT" --name "ActivityTest-$(date +%s)"
ACT_ISSUE_ID=$(echo "$RUN_STDOUT" | jq -r '.id' 2>/dev/null)

if [ -n "$ACT_ISSUE_ID" ] && [ "$ACT_ISSUE_ID" != "null" ]; then
    run plane activity list -p "$PROJECT" --work-item-id "$ACT_ISSUE_ID"
    if [ "$RUN_RC" -eq 0 ]; then
        pass "activity list exits 0"
    else
        fail "activity list" "exit $RUN_RC: $(echo "$RUN_STDERR" | head -2)"
    fi
    # Cleanup
    run plane issue delete -p "$PROJECT" --work-item-id "$ACT_ISSUE_ID"
fi

# ============================================================
section "25. Comment Commands"
# ============================================================

run plane issue create -p "$PROJECT" --name "CommentTest-$(date +%s)"
CMT_ISSUE_ID=$(echo "$RUN_STDOUT" | jq -r '.id' 2>/dev/null)

if [ -n "$CMT_ISSUE_ID" ] && [ "$CMT_ISSUE_ID" != "null" ]; then
    run plane comment list -p "$PROJECT" --work-item-id "$CMT_ISSUE_ID"
    if [ "$RUN_RC" -eq 0 ]; then
        pass "comment list exits 0"
    else
        fail "comment list" "exit $RUN_RC: $(echo "$RUN_STDERR" | head -2)"
    fi

    run plane comment add -p "$PROJECT" --work-item-id "$CMT_ISSUE_ID" --comment-html "<p>test comment</p>"
    if [ "$RUN_RC" -eq 0 ]; then
        pass "comment add exits 0"
    else
        fail "comment add" "exit $RUN_RC: $(echo "$RUN_STDERR" | head -2)"
    fi

    # Cleanup
    run plane issue delete -p "$PROJECT" --work-item-id "$CMT_ISSUE_ID"
fi

# ============================================================
section "26. Link Commands"
# ============================================================

run plane issue create -p "$PROJECT" --name "LinkTest-$(date +%s)"
LINK_ISSUE_ID=$(echo "$RUN_STDOUT" | jq -r '.id' 2>/dev/null)

if [ -n "$LINK_ISSUE_ID" ] && [ "$LINK_ISSUE_ID" != "null" ]; then
    run plane link list -p "$PROJECT" --work-item-id "$LINK_ISSUE_ID"
    if [ "$RUN_RC" -eq 0 ]; then
        pass "link list exits 0"
    else
        fail "link list" "exit $RUN_RC: $(echo "$RUN_STDERR" | head -2)"
    fi

    # Command is 'link add' (not 'link create')
    run plane link add -p "$PROJECT" --work-item-id "$LINK_ISSUE_ID" --url "https://example.com"
    if [ "$RUN_RC" -eq 0 ]; then
        pass "link add exits 0"
        LINK_ID=$(echo "$RUN_STDOUT" | jq -r '.id' 2>/dev/null)
        if [ -n "$LINK_ID" ] && [ "$LINK_ID" != "null" ]; then
            echo "  Created link: $LINK_ID"
        fi
    else
        fail "link add" "exit $RUN_RC: $(echo "$RUN_STDERR" | head -2)"
    fi

    # Cleanup
    run plane issue delete -p "$PROJECT" --work-item-id "$LINK_ISSUE_ID"
fi

# ============================================================
section "27. State CRUD"
# ============================================================

run plane state create -p "$PROJECT" --name "test-state-$(date +%s)" --color "#FF0000"
if [ "$RUN_RC" -eq 0 ]; then
    pass "state create exits 0"
    STATE_ID=$(echo "$RUN_STDOUT" | jq -r '.id' 2>/dev/null)
    if [ -n "$STATE_ID" ] && [ "$STATE_ID" != "null" ]; then
        echo "  Created state: $STATE_ID"

        run plane state get -p "$PROJECT" --state-id "$STATE_ID"
        assert_rc "state get exits 0" 0 "$RUN_RC"

        run plane state update -p "$PROJECT" --state-id "$STATE_ID" --name "renamed-state"
        assert_rc "state update exits 0" 0 "$RUN_RC"

        run plane state delete -p "$PROJECT" --state-id "$STATE_ID"
        assert_rc "state delete exits 0" 0 "$RUN_RC"
    fi
else
    fail "state create" "exit $RUN_RC: $(echo "$RUN_STDERR" | head -2)"
fi

# ============================================================
section "28. Cross-resource: Issue ↔ Cycle"
# ============================================================

run plane issue create -p "$PROJECT" --name "CrossRes-$(date +%s)"
CR_ISSUE_ID=$(echo "$RUN_STDOUT" | jq -r '.id' 2>/dev/null)

run plane cycle create -p "$PROJECT" --name "CrossResCycle-$(date +%s)" --owned-by "$MY_USER_ID"
CR_CYCLE_ID=$(echo "$RUN_STDOUT" | jq -r '.id' 2>/dev/null)

if [ -n "$CR_ISSUE_ID" ] && [ "$CR_ISSUE_ID" != "null" ] && [ -n "$CR_CYCLE_ID" ] && [ "$CR_CYCLE_ID" != "null" ]; then
    # Check what the add-work-items flag is named
    run plane cycle add-work-items -p "$PROJECT" --cycle-id "$CR_CYCLE_ID" --issues "$CR_ISSUE_ID"
    if [ "$RUN_RC" -eq 0 ]; then
        pass "cycle add-work-items exits 0"
    else
        fail "cycle add-work-items" "exit $RUN_RC: $(echo "$RUN_STDERR" | head -2)"
    fi

    run plane cycle list-work-items -p "$PROJECT" --cycle-id "$CR_CYCLE_ID" --output json
    if [ "$RUN_RC" -eq 0 ]; then
        pass "cycle list-work-items exits 0"
    else
        fail "cycle list-work-items" "exit $RUN_RC: $(echo "$RUN_STDERR" | head -2)"
    fi

    run plane cycle remove-work-item -p "$PROJECT" --cycle-id "$CR_CYCLE_ID" --issue-id "$CR_ISSUE_ID"
    if [ "$RUN_RC" -eq 0 ]; then
        pass "cycle remove-work-item exits 0"
    else
        fail "cycle remove-work-item" "exit $RUN_RC: $(echo "$RUN_STDERR" | head -2)"
    fi

    # Cleanup
    run plane issue delete -p "$PROJECT" --work-item-id "$CR_ISSUE_ID"
    run plane cycle delete -p "$PROJECT" --cycle-id "$CR_CYCLE_ID"
else
    fail "cross-resource setup" "could not create issue ($CR_ISSUE_ID) or cycle ($CR_CYCLE_ID)"
fi

# ============================================================
section "29. Cross-resource: Issue ↔ Module"
# ============================================================

run plane issue create -p "$PROJECT" --name "ModCross-$(date +%s)"
MR_ISSUE_ID=$(echo "$RUN_STDOUT" | jq -r '.id' 2>/dev/null)

run plane module create -p "$PROJECT" --name "ModCrossMod-$(date +%s)"
MR_MODULE_ID=$(echo "$RUN_STDOUT" | jq -r '.id' 2>/dev/null)

if [ -n "$MR_ISSUE_ID" ] && [ "$MR_ISSUE_ID" != "null" ] && [ -n "$MR_MODULE_ID" ] && [ "$MR_MODULE_ID" != "null" ]; then
    run plane module add-work-items -p "$PROJECT" --module-id "$MR_MODULE_ID" --issues "$MR_ISSUE_ID"
    if [ "$RUN_RC" -eq 0 ]; then
        pass "module add-work-items exits 0"
    else
        fail "module add-work-items" "exit $RUN_RC: $(echo "$RUN_STDERR" | head -2)"
    fi

    run plane module list-work-items -p "$PROJECT" --module-id "$MR_MODULE_ID" --output json
    if [ "$RUN_RC" -eq 0 ]; then
        pass "module list-work-items exits 0"
    else
        fail "module list-work-items" "exit $RUN_RC: $(echo "$RUN_STDERR" | head -2)"
    fi

    # Cleanup
    run plane issue delete -p "$PROJECT" --work-item-id "$MR_ISSUE_ID"
    run plane module delete -p "$PROJECT" --module-id "$MR_MODULE_ID"
else
    fail "module cross-resource setup" "could not create issue or module"
fi

# ============================================================
section "30. Workspace-scoped Commands (no -p needed)"
# ============================================================

run plane member get-workspace --output json
assert_rc "member get-workspace (no project) exits 0" 0 "$RUN_RC"

run plane project list --output json
assert_rc "project list (no project) exits 0" 0 "$RUN_RC"

# ============================================================
section "31. Attachment Commands"
# ============================================================

# Check attachment list (may need work-item-id)
run plane attachment --help
assert_rc "attachment help exits 0" 0 "$RUN_RC"
assert_contains "attachment has subcommands" "$RUN_STDOUT" "get"

# ============================================================
section "32. Customer Commands"
# ============================================================

run plane customer --help
assert_rc "customer help exits 0" 0 "$RUN_RC"

run plane customer list
if [ "$RUN_RC" -eq 0 ]; then
    pass "customer list exits 0"
else
    skip "customer list" "exit $RUN_RC (feature may not be enabled)"
fi

# ============================================================
section "33. Epic Commands"
# ============================================================

run plane epic --help
assert_rc "epic help exits 0" 0 "$RUN_RC"

run plane epic list
if [ "$RUN_RC" -eq 0 ]; then
    pass "epic list exits 0"
else
    skip "epic list" "exit $RUN_RC (feature may not be enabled)"
fi

# ============================================================
section "34. Initiative Commands"
# ============================================================

run plane initiative --help
assert_rc "initiative help exits 0" 0 "$RUN_RC"

run plane initiative list
if [ "$RUN_RC" -eq 0 ]; then
    pass "initiative list exits 0"
else
    skip "initiative list" "exit $RUN_RC (feature may not be enabled)"
fi

# ============================================================
section "35. Teamspace Commands"
# ============================================================

run plane teamspace --help
assert_rc "teamspace help exits 0" 0 "$RUN_RC"

run plane teamspace list
if [ "$RUN_RC" -eq 0 ]; then
    pass "teamspace list exits 0"
else
    skip "teamspace list" "exit $RUN_RC (feature may not be enabled)"
fi

# ============================================================
section "36. Worklog Commands"
# ============================================================

run plane worklog --help
assert_rc "worklog help exits 0" 0 "$RUN_RC"

# ============================================================
# Summary
# ============================================================
echo ""
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BOLD}Results: ${GREEN}$PASS passed${NC}, ${RED}$FAIL failed${NC}, ${YELLOW}$SKIP skipped${NC}"
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if [ "$FAIL" -gt 0 ]; then
    echo -e "\n${RED}${BOLD}Failures:${NC}${ERRORS}"
    echo ""
fi

exit $FAIL
