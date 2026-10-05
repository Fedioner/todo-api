#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

: "${KUBE_CONTEXT:?Set KUBE_CONTEXT to the target Kubernetes context}"
image=${1:?Pass an image reference pinned by digest}
version=${2:?Pass the version baked into the image}
if [[ ! $image =~ @sha256:[0-9a-f]{64}$ ]]; then
    echo "Expected an image pinned by sha256 digest" >&2
    exit 2
fi

kube() {
    kubectl --context "$KUBE_CONTEXT" -n todo "$@"
}

smoke() (
    log=$(mktemp)
    kubectl --context "$KUBE_CONTEXT" -n todo port-forward --address 127.0.0.1 svc/todo-api :80 >"$log" 2>&1 &
    forward_pid=$!
    trap 'kill "$forward_pid" 2>/dev/null || true; wait "$forward_pid" 2>/dev/null || true; rm -f "$log"' EXIT

    for ((attempt=0; attempt<30; attempt++)); do
        port=$(sed -n 's/^Forwarding from 127\.0\.0\.1:\([0-9]*\) .*/\1/p' "$log")
        if [[ -n $port ]]; then
            python3 scripts/smoke.py "http://127.0.0.1:$port" --read-only "$@"
            return
        fi
        if ! kill -0 "$forward_pid" 2>/dev/null; then
            break
        fi
        sleep 1
    done
    cat "$log" >&2
    return 1
)

deployment=deployment/todo-api
previous=$(kube get "$deployment" -o "jsonpath={.spec.template.spec.containers[?(@.name=='api')].image}")
if [[ -z $previous ]]; then
    echo "Current API image is missing" >&2
    exit 1
fi

echo "Deploying $image (previous: $previous)"
if kube set image "$deployment" "api=$image" &&
   kube rollout status "$deployment" --timeout=150s &&
   smoke --version "$version"; then
    echo "Deployment and API check passed."
    exit 0
fi

echo "Release failed. Restoring $previous" >&2
if kube set image "$deployment" "api=$previous" &&
   kube rollout status "$deployment" --timeout=150s &&
   smoke; then
    echo "Previous release restored." >&2
else
    echo "Rollback failed. Check deployment events and container logs." >&2
fi
exit 1
