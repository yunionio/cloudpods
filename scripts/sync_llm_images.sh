#!/bin/bash
# 用法:
#   ./sync-images.sh <targetRegistry>
# 示例:
#   ./sync-images.sh crpi-nf3abu98o8qf9y2x.cn-beijing.personal.cr.aliyuncs.com/eikoh

set -euo pipefail

if [ $# -ne 1 ]; then
  echo "用法: $0 <targetRegistry>"
  echo "例如: $0 crpi-nf3abu98o8qf9y2x.cn-beijing.personal.cr.aliyuncs.com/eikoh"
  exit 1
fi

TARGET_REGISTRY="$1"
SOURCE_REGISTRY="${SOURCE_REGISTRY:-docker.io}"

# ----------------------------
# 要同步的镜像列表
# ----------------------------
IMAGES=(
  # "nginx:latest"
  # "redis:6-alpine"
  # "postgres:15-alpine"
  # "langgenius/dify-api:1.7.2"
  # "langgenius/dify-sandbox:0.2.12"
  # "langgenius/dify-plugin-daemon:0.2.0-local"
  # "langgenius/dify-web:1.7.2"
  # "ubuntu/squid:latest"
  # "semitechnologies/weaviate:1.19.0"
  # "ollama/ollama:0.15.1"
  # "vllm/vllm-openai:v0.15.1"
  # "yanwk/comfyui-boot:cu128-slim"
  # "node:22-bookworm"
  # "coollabsio/openclaw:latest"
  # "coollabsio/openclaw-browser:latest"
  # ghcr.io/coollabsio/openclaw-base:latest
  # lscr.io/linuxserver/webtop:ubuntu-xfce
  registry.cn-beijing.aliyuncs.com/zexi/openclaw:ubu-20260309.4
)

for image in "${IMAGES[@]}"; do
  # 拆分 name 和 tag
  if [[ "$image" == *":"* ]]; then
    name="${image%%:*}"  # 冒号前
    tag="${image##*:}"   # 冒号后
  else
    name="$image"
    tag="latest"
  fi

  short_name="${name##*/}"  # 目标镜像只取最后一级名字

  # 如果 name 已经包含 registry（例如 ghcr.io/xxx 或 localhost:5000/xxx），就不要再前缀 docker.io
  first_component="${name%%/*}"
  if [[ "$name" == */* ]] && { [[ "$first_component" == *.* ]] || [[ "$first_component" == *:* ]] || [[ "$first_component" == "localhost" ]]; }; then
    SRC="docker://${name}:${tag}"
  else
    SRC="docker://${SOURCE_REGISTRY}/${name}:${tag}"
  fi
  DST="docker://${TARGET_REGISTRY}/${short_name}:${tag}"

  echo
  echo "Sync dify image"
  echo "  Source: ${SRC}"
  echo "  Target: ${DST}"
  echo

  skopeo copy --override-os linux --multi-arch all "${SRC}" "${DST}"

  echo "Completed: ${short_name}:${tag}"
done

echo "All images sync completed"
