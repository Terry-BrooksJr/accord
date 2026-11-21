#!/bin/bash
set -euo pipefail

run_stripe_test_events() {
    local event
    local -a events=(
        payment_intent.amount_capturable_updated
        payment_intent.canceled
        payment_intent.created
        payment_intent.partially_funded
        payment_intent.payment_failed
        payment_intent.processing
        payment_intent.requires_action
        payment_intent.succeeded
    )
    
    local count=$1
    local i index event
    
    local failed=0
    
    for ((i = 0; i < count; i++)); do
        index=$((RANDOM % ${#events[@]}))
        event=${events[index]}
        
        printf '[%d/%d] Triggering %s\n' "$((i + 1))" "$count" "$event"
        if ! stripe trigger "$event"; then
            printf '[%d/%d] Failed: %s\n' "$((i + 1))" "$count" "$event" >&2
            failed=$((failed + 1))
        fi
    done
    
    printf 'Attempted %d triggers: %d succeeded, %d failed.\n' \
    "$count" "$((count - failed))" "$failed"
    
    if ((failed > 0)); then
        return 1
    fi
}

if [[ $# -ne 1 || ! ${1-} =~ ^(0|[1-9][0-9]*)$ ]]; then
    printf 'Usage: %s COUNT (a non-negative integer)\n' "$0" >&2
    exit 2
fi

run_stripe_test_events "$1"
