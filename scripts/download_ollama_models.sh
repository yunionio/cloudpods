#!/bin/bash

# 脚本: Ollama 模型安全下载器
# 描述: 此脚本根据输入的模型名称（例如 "qwen3:8b"）安全地下载 Ollama 模型文件。
#       它使用临时文件进行下载，并在成功后重命名，以防止文件损坏。
#
# 用法: ./download_model.sh <模型名称:标签>
# 示例: ./download_model.sh qwen3:8b

# --- 配置 ---

# Ollama 注册表的基础 URL
LLM_OLLAMA_LIBRARY_BASE_URL="https://registry.ollama.ai/v2/library"
# 模型在主机上保存的基础路径
LLM_OLLAMA_HOST_PATH="/opt/ollama-models"
# 主机上的清单目录
LLM_OLLAMA_HOST_MANIFESTS_DIR="/manifests"
# 主机上的 blob 目录
LLM_OLLAMA_HOST_BLOBS_DIR="/blobs"

# --- 脚本 ---

# 检查是否提供了模型名称作为参数
if [ -z "$1" ]; then
  echo "错误：未提供模型名称。"
  echo "用法: $0 <模型名称:标签>"
  exit 1
fi

# 从输入参数中解析模型名称和标签
MODEL_FULL_NAME=$1
MODEL_NAME=$(echo "$MODEL_FULL_NAME" | cut -d':' -f1)
MODEL_TAG=$(echo "$MODEL_FULL_NAME" | cut -d':' -f2)

# 检查模型名称和标签是否已成功解析
if [ "$MODEL_NAME" == "$MODEL_TAG" ] || [ -z "$MODEL_TAG" ]; then
  echo "错误：模型名称格式无效。应为 '名称:标签' (例如 'qwen3:8b')。"
  exit 1
fi

echo "开始安全下载模型: $MODEL_FULL_NAME"
echo "----------------------------------------"

# --- 1. 下载清单文件 ---

# 创建清单文件要保存的目录 (如果不存在)
MANIFEST_DIR="$LLM_OLLAMA_HOST_PATH$LLM_OLLAMA_HOST_MANIFESTS_DIR"
mkdir -p "$MANIFEST_DIR"

# 构造清单文件的 URL、最终路径和临时路径
MANIFEST_SUFFIX_URL="$MODEL_NAME/manifests/$MODEL_TAG"
MANIFEST_URL="$LLM_OLLAMA_LIBRARY_BASE_URL/$MANIFEST_SUFFIX_URL"
MANIFEST_FILE_PATH="$MANIFEST_DIR/$MODEL_NAME-$MODEL_TAG"
MANIFEST_FILE_PATH_TMP="${MANIFEST_FILE_PATH}.tmp"

echo "步骤 1: 正在从 $MANIFEST_URL 下载清单..."

# 检查最终文件是否已存在，如果存在则跳过
if [ -f "$MANIFEST_FILE_PATH" ]; then
    echo "清单文件已存在，跳过下载。"
else
    # 使用 wget 下载到临时文件
    wget --quiet --show-progress -O "$MANIFEST_FILE_PATH_TMP" "$MANIFEST_URL"
    # 检查 wget 的退出状态
    if [ $? -eq 0 ]; then
        # 如果成功，重命名临时文件
        mv "$MANIFEST_FILE_PATH_TMP" "$MANIFEST_FILE_PATH"
        echo "清单已成功下载到: $MANIFEST_FILE_PATH"
    else
        # 如果失败，打印错误并删除临时文件
        echo "错误：下载清单失败。请检查模型名称是否正确以及网络连接。"
        rm -f "$MANIFEST_FILE_PATH_TMP"
        exit 1
    fi
fi

echo "----------------------------------------"

# --- 2. 从清单中提取 Blob 的摘要 (digest) ---

echo "步骤 2: 正在从清单文件中解析 blob 摘要..."
# 使用 grep 和 sed 通过正则表达式提取所有 blob 的 digest
BLOBS=$(grep -o '"digest":"sha256:[^"]*' "$MANIFEST_FILE_PATH" | sed 's/"digest":"//')

if [ -z "$BLOBS" ]; then
  echo "警告：在清单文件中未找到任何 blob 摘要。"
  exit 0
fi

echo "已找到以下 blobs:"
echo "$BLOBS"
echo "----------------------------------------"

# --- 3. 下载所有 Blob 文件 ---

echo "步骤 3: 正在下载所有 blob 文件..."
# 创建 blob 文件要保存的目录 (如果不存在)
BLOBS_DIR="$LLM_OLLAMA_HOST_PATH$LLM_OLLAMA_HOST_BLOBS_DIR"
mkdir -p "$BLOBS_DIR"

# 逐行遍历所有提取出的 blob 摘要
for BLOB in $BLOBS; do
  # 将 blob 摘要中的 "sha256:" 替换为 "sha256-" 以用作文件名
  BLOB_FILENAME=$(echo "$BLOB" | sed 's/sha256:/sha256-/')
  BLOB_FILE_PATH="$BLOBS_DIR/$BLOB_FILENAME"
  BLOB_FILE_PATH_TMP="${BLOB_FILE_PATH}.tmp"

  # 如果最终文件已经存在，则跳过下载
  if [ -f "$BLOB_FILE_PATH" ]; then
    echo "  文件已存在, 跳过下载: $BLOB_FILENAME"
    continue
  fi

  # 构造 blob 的下载 URL
  BLOB_URL="$LLM_OLLAMA_LIBRARY_BASE_URL/$MODEL_NAME/blobs/$BLOB"

  echo "  正在下载 $BLOB..."
  # 使用 wget 下载 blob 到临时文件
  wget --quiet --show-progress -O "$BLOB_FILE_PATH_TMP" "$BLOB_URL"
  
  # 检查 wget 的退出状态
  if [ $? -eq 0 ]; then
    # 如果成功，重命名临时文件
    mv "$BLOB_FILE_PATH_TMP" "$BLOB_FILE_PATH"
    echo "  已成功保存到: $BLOB_FILE_PATH"
  else
    # 如果失败，打印错误并删除临时文件
    echo "  错误：下载 blob $BLOB 失败。已从 $BLOB_URL 尝试下载。"
    rm -f "$BLOB_FILE_PATH_TMP"
    # 如果希望在任何一个 blob 下载失败时立即中止整个脚本，请取消下一行的注释
    # exit 1 
  fi
done

echo "----------------------------------------"
echo "所有任务已完成。"
echo "模型 '$MODEL_FULL_NAME' 已成功下载到 '$LLM_OLLAMA_HOST_PATH'。"