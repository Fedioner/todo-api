#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

name=${1:-todo-lab}
export KUBECONFIG=${KUBECONFIG:-"$PWD/.kubeconfig"}
image="todo-api:dev"
operator="https://raw.githubusercontent.com/cloudnative-pg/cloudnative-pg/v1.30.2/releases/cnpg-1.30.2.yaml"

if ! kind get clusters | grep -Fxq "$name"; then
    kind create cluster --name "$name" --config deploy/kind.yaml \
        --image kindest/node:v1.36.1 --wait 180s
else
    kind export kubeconfig --name "$name" --kubeconfig "$KUBECONFIG"
fi

context="kind-$name"
kube() {
    kubectl --context "$context" "$@"
}

docker build -t "$image" .
kind load docker-image "$image" --name "$name"
kube apply --server-side -f "$operator"
kube rollout status -n cnpg-system deployment/cnpg-controller-manager --timeout=240s
kube wait --for=condition=Established crd/clusters.postgresql.cnpg.io --timeout=60s
kube apply -f deploy/namespace.yaml
kube -n todo create configmap todo-db-schema --from-file=db/schema.sql \
    --dry-run=client -o yaml | kube apply -f -
kube apply -f deploy/postgres.yaml
kube -n todo wait --for=condition=Ready cluster/todo-db --timeout=360s
# Preserve the running image on a rerun to avoid replacing a release with the dev tag.
if ! kube -n todo get deployment todo-api >/dev/null 2>&1; then
    kube apply -f deploy/api.yaml
fi
kube -n todo rollout status deployment/todo-api --timeout=160s
kube -n todo get pods -o wide
