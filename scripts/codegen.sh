#!/bin/bash

set -o errexit
set -o pipefail

ONECLOUD="yunion.io/x/onecloud"
PKG_ONECLOUD="$ONECLOUD/pkg"
PKG_APIS="$PKG_ONECLOUD/apis"
PKG_GENERATED="$PKG_ONECLOUD/generated"
PKG_SWAGGER="$PKG_GENERATED/swagger"

MODEL_API_PKG_MAP=(
    "$PKG_ONECLOUD/cloudcommon/db:$PKG_APIS"
    "$PKG_ONECLOUD/cloudprovider:$PKG_APIS/cloudprovider"
    "$PKG_ONECLOUD/compute/models:$PKG_APIS/compute"
    "$PKG_ONECLOUD/image/models:$PKG_APIS/image"
    "$PKG_ONECLOUD/keystone/models:$PKG_APIS/identity"
)

MODEL_SWAGGER_PKG_MAP=(
    "$PKG_ONECLOUD/compute/models:$PKG_SWAGGER/compute"
    "$PKG_ONECLOUD/image/models:$PKG_SWAGGER/image"
    "$PKG_ONECLOUD/keystone/models:$PKG_SWAGGER/identity"
)

CURDIR="$(dirname $(dirname $0))"
DOCS_DIR="$CURDIR/docs"
OUTPUT_DIR="$CURDIR/_output"
OUTPUT_SWAGGER_DIR="$OUTPUT_DIR/swagger"

generate_model_api() {
    for pkg_path in "${MODEL_API_PKG_MAP[@]}"; do
        model_pkg="${pkg_path%%:*}"
        api_pkg="${pkg_path##*:}"
        model-api-gen \
            --input-dirs $model_pkg \
            --output-package $api_pkg
    done
}

generate_swagger_spec() {
#    for pkg_path in "${MODEL_SWAGGER_PKG_MAP[@]}"; do
#        model_pkg="${pkg_path%%:*}"
#        swaager_pkg="${pkg_path##*:}"
#        swagger-gen \
#            --input-dirs $model_pkg \
#            --output-package $swaager_pkg
#    done
    swagger-gen \
        -i "$PKG_ONECLOUD/compute/models" \
        -p "$PKG_SWAGGER/compute"

    swagger-gen \
        -i "$PKG_ONECLOUD/image/models" \
        -p "$PKG_SWAGGER/image"

    swagger-gen \
        -i "$PKG_ONECLOUD/keystone/tokens" \
        -i "$PKG_ONECLOUD/keystone/models" \
        -p "$PKG_SWAGGER/identity"
}

generate_swagger_yaml() {
    mkdir -p "$OUTPUT_SWAGGER_DIR"
    for pkg_path in "${MODEL_SWAGGER_PKG_MAP[@]}"; do
        model_pkg="${pkg_path%%:*}"
        swaager_pkg="${pkg_path##*:}"
        work_dir=${swaager_pkg#"$ONECLOUD"}
        GO111MODULE=off swagger generate spec \
            --scan-models \
            --work-dir="$CURDIR/$work_dir" \
            -o "$OUTPUT_SWAGGER_DIR/swagger_$(basename $swaager_pkg).yaml"
    done
}

generate_swagger_serve() {
    input_files="$(find $OUTPUT_SWAGGER_DIR -name 'swagger_*.yaml' -type f | paste -sd,)"
    swagger-serve generate -i "$input_files" -o "$OUTPUT_SWAGGER_DIR" --serve
}

show_help() {
    cat <<EOF
usage: $(basename $0) <subcommand>
Subcommands:
    model_api: genereate model struct code
    swagger_spec: generate swagger spec code
    swagger_serve: generate swagger web site
EOF
}

subcmd=$1
case $subcmd in
    "" | "-h" | "--help")
        show_help
        ;;
    *)
        shift
        generate_${subcmd} $@
        if [ $? = 127 ]; then
            echo "Run --help for a list of known subcommands." >&2
            exit 1
        fi
        ;;
esac
