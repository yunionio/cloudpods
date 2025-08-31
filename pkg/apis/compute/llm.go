package compute

const (
	LLM_OLLAMA                  = "ollama"
	LLM_OLLAMA_EXEC_PATH        = "/bin/ollama"
	LLM_OLLAMA_PULL_ACTION      = "pull"
	LLM_OLLAMA_LIST_ACTION      = "list"
	LLM_OLLAMA_EXPORT_ENV_KEY   = "OLLAMA_HOST"
	LLM_OLLAMA_EXPORT_ENV_VALUE = "0.0.0.0:11434"
	LLM_OLLAMA_CACHE_HOST_DIR   = "/.llm_ollama_cache"
	LLM_OLLAMA_CACHE_MOUNT_PATH = "/usr/local/.llm_ollama_cache"
)

const (
	LLM_STATUS_CREATING_POD      = "creating_pod"
	LLM_STATUS_CREAT_POD_FAILED  = "creat_pod_failed"
	LLM_STATUS_PULLING_MODEL     = "pulling_model"
	LLM_STATUS_PULL_MODEL_FAILED = "pull_model_failed"
	LLM_STATUS_PULLED_MODEL      = "pulled_model"
)

const (
	LLM_CACHE_OLLAMA_MODEL = `#!/bin/sh
set -e
# 用法: ./cache_ollama_model.sh <模型名称:标签>
# 例如: ./cache_ollama_model.sh qwen2:0.5b

# --- 1. 参数和路径设置 ---
# 检查是否提供了模型名称参数
if [ -z "$1" ]; then
  echo "错误: 请提供一个模型名称作为参数 (例如: qwen2:0.5b)"
  exit 1
fi
MODEL_FULL_NAME=$1
# 定义基础路径
TARGET_PATH="/root/.ollama/models"
BLOBS_PATH="${TARGET_PATH}/blobs"
MANIFESTS_BASE_PATH="${TARGET_PATH}/manifests/registry.ollama.ai/library"
CACHE_PATH="%s"

# --- 2. 准备目标路径和文件 ---
# 从模型名称解析出 manifest 文件的相对路径 (例如: qwen2:0.5b -> qwen2/0.5b)
MODEL_PATH_PART=$(echo "$MODEL_FULL_NAME" | sed 's/:/\//')
MANIFEST_FILE="${MANIFESTS_BASE_PATH}/${MODEL_PATH_PART}"
# 检查 manifest 文件是否存在
if [ ! -f "$MANIFEST_FILE" ]; then
  echo "错误: manifest 文件未找到: ${MANIFEST_FILE}"
  exit 1
fi
# 创建本次缓存的目标目录
DEST_DIR="${CACHE_PATH}/${MODEL_FULL_NAME}"
echo "INFO: 创建缓存目录: ${DEST_DIR}"
mkdir -p "$DEST_DIR"

# --- 3. 解析并拷贝文件 ---
echo "INFO: 开始解析 manifest 文件: ${MANIFEST_FILE}"
# 解析出所有的 digest 值 (格式为 sha256-...)
# 使用 command substitution $(...) 将命令的输出结果赋值给变量
ALL_DIGESTS=$(cat "$MANIFEST_FILE" | grep -o '"digest":"[^"]*"' | sed 's/"digest":"//' | sed 's/"//' | sed 's/sha256:/sha256-/' )
if [ -z "$ALL_DIGESTS" ]; then
    echo "错误: 未能从 manifest 文件中解析出任何 digest。"
    exit 1
fi
# 首先拷贝 manifest 文件本身
echo "INFO: 拷贝 manifest 文件..."
cp "$MANIFEST_FILE" "$DEST_DIR/"
# 循环遍历所有 digest, 并拷贝对应的 blob 文件
echo "INFO: 开始拷贝模型 layers (blobs)..."
for digest in $ALL_DIGESTS; do
    BLOB_FILE="${BLOBS_PATH}/${digest}"

    if [ -f "$BLOB_FILE" ]; then
        echo "  -> 正在拷贝: ${digest}"
        cp "$BLOB_FILE" "$DEST_DIR/"
    else
        echo "  -> 错误: 未找到对应的 blob 文件: ${BLOB_FILE}"
		    exit 1
    fi
done

echo "INFO: 模型 '${MODEL_FULL_NAME}' 缓存完成！"
echo "INFO: 所有文件已保存至: ${DEST_DIR}"`

	LLM_RESTORE_OLLAMA_MODEL = `#!/bin/sh
set -e
# 脚本功能: 从缓存目录复原一个 Ollama 模型及其 layers 到 Ollama 的工作目录。
# 用法: ./restore_ollama_model.sh <模型名称:标签>
# 例如: ./restore_ollama_model.sh qwen2:0.5b

# --- 1. 参数和路径设置 ---
# 检查是否提供了模型名称参数
if [ -z "$1" ]; then
  echo "错误: 请提供一个模型名称作为参数 (例如: qwen2:0.5b)"
  exit 1
fi

MODEL_FULL_NAME=$1
# 定义基础路径
TARGET_PATH="/root/.ollama/models"
BLOBS_PATH="${TARGET_PATH}/blobs"
MANIFESTS_BASE_PATH="${TARGET_PATH}/manifests/registry.ollama.ai/library"
CACHE_PATH="%s"

# --- 2. 检查源缓存并确定文件路径 ---
# 源缓存目录
SOURCE_DIR="${CACHE_PATH}/${MODEL_FULL_NAME}"
# 检查源缓存目录是否存在
if [ ! -d "$SOURCE_DIR" ]; then
  echo "错误: 未找到该模型的缓存: ${SOURCE_DIR}"
  exit 1
fi
# 从模型名称解析出 manifest 文件的目标路径 (例如: qwen2:0.5b -> qwen2/0.5b)
MODEL_PATH_PART=$(echo "$MODEL_FULL_NAME" | sed 's/:/\//')
DEST_MANIFEST_FILE="${MANIFESTS_BASE_PATH}/${MODEL_PATH_PART}"
# 从模型名称中提取标签作为 manifest 的文件名 (例如: qwen2:0.5b -> 0.5b)
MANIFEST_FILENAME=$(echo "$MODEL_FULL_NAME" | cut -d':' -f2)
SOURCE_MANIFEST_FILE="${SOURCE_DIR}/${MANIFEST_FILENAME}"
# 再次确认源 manifest 文件存在
if [ ! -f "$SOURCE_MANIFEST_FILE" ]; then
  echo "错误: 在缓存目录中未找到 manifest 文件: ${SOURCE_MANIFEST_FILE}"
  exit 1
fi

# --- 3. 开始复原文件 ---
echo "INFO: 找到模型缓存，开始从 '${SOURCE_DIR}' 复原..."
# 首先，创建 manifest 的目标目录 (例如 /.../library/qwen2)
DEST_MANIFEST_DIR=$(dirname "$DEST_MANIFEST_FILE")
echo "INFO: 确保 manifest 目录存在: ${DEST_MANIFEST_DIR}"
mkdir -p "$DEST_MANIFEST_DIR"
# 复原 manifest 文件
echo "INFO: 复原 manifest 文件至: ${DEST_MANIFEST_FILE}"
cp "$SOURCE_MANIFEST_FILE" "$DEST_MANIFEST_FILE"
# 循环遍历源目录中的所有文件，复原 blobs
echo "INFO: 开始复原模型 layers (blobs) 至: ${BLOBS_PATH}"
for file in "$SOURCE_DIR"/*; do
    # 获取文件名
    filename=$(basename "$file")
    # 只要不是 manifest 文件，就把它当作 blob 文件处理
    if [ "$filename" != "$MANIFEST_FILENAME" ]; then
        echo "  -> 正在复原: ${filename}"
        cp "$file" "$BLOBS_PATH/"
    fi
done

echo "INFO: 模型 '${MODEL_FULL_NAME}' 复原完成！"
echo "INFO: 您现在应该可以使用 'ollama run ${MODEL_FULL_NAME}' 来运行该模型了。"
`
)
