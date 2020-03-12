#!/bin/bash

# See: https://github.com/kubernetes/kubernetes/issues/79384#issuecomment-521493597

set -eo pipefail

export GO111MODULE=on

VERSION=${1#"v"}

if [ -z "$VERSION" ]; then
    cat <<EOF
Must specify version!

Usage: $0 v1.15.0
EOF
    exit 1
fi

MODS=($(
    curl -sS https://raw.githubusercontent.com/kubernetes/kubernetes/v${VERSION}/go.mod |
    sed -n 's|.*k8s.io/\(.*\) => ./staging/src/k8s.io/.*|k8s.io/\1|p'
))

for MOD in "${MODS[@]}"; do
    V=$(
        go mod download -json "${MOD}@kubernetes-${VERSION}" |
        sed -n 's|.*"Version": "\(.*\)".*|\1|p'
    )
    go mod edit "-replace=${MOD}=${MOD}@${V}"
done

go get "k8s.io/kubernetes@v${VERSION}"
