package llm

const OLLAMA_REGISTRY_YAML = `
ollama:
  qwen3-vl:
    description: Qwen3-vl is the most powerful vision-language model in the Qwen model family to date.
    tags:
      2b:
        model_size: 1.9GB
        context_length: 256K
        capabilities:
          - Text
          - Image
      4b:
        model_size: 3.3GB
        context_length: 256K
        capabilities:
          - Text
          - Image
      8b:
        model_size: 6.1GB
        context_length: 256K
        capabilities:
          - Text
          - Image
        is_latest: true
      30b:
        model_size: 20GB
        context_length: 256K
        capabilities:
          - Text
          - Image
      32b:
        model_size: 21GB
        context_length: 256K
        capabilities:
          - Text
          - Image
  qwen3:
    description: Qwen3 is the latest generation of large language models in Qwen series, offering a comprehensive suite of dense and mixture-of-experts (MoE) models. 
    tags:
      0.6b:
        model_size: 523MB
        context_length: 40K
        capabilities:
          - Text
      1.7b:
        model_size: 1.4GB
        context_length: 40K
        capabilities:
          - Text
      4b:
        model_size: 2.5GB
        context_length: 256K
        capabilities:
          - Text
      8b:
        model_size: 5.2GB
        context_length: 40K
        capabilities:
          - Text
        is_latest: true
      14b:
        model_size: 9.3GB
        context_length: 40K
        capabilities:
          - Text
      30b:
        model_size: 19GB
        context_length: 256K
        capabilities:
          - Text
      32b:
        model_size: 20GB
        context_length: 40K
        capabilities:
          - Text
  qwen2.5-coder:
    description: The latest series of Code-Specific Qwen models, with significant improvements in code generation, code reasoning, and code fixing.
    tags:
      latest:
        model_size: 4.7GB
        context_length: 32K
        capabilities:
          - Text
      0.5b:
        model_size: 398MB
        context_length: 32K
        capabilities:
          - Text
      1.5b:
        model_size: 986MB
        context_length: 32K
        capabilities:
          - Text
      3b:
        model_size: 1.9GB
        context_length: 32K
        capabilities:
          - Text
      7b:
        model_size: 4.7GB
        context_length: 32K
        capabilities:
          - Text
        is_latest: true
      14b:
        model_size: 9.0GB
        context_length: 32K
        capabilities:
          - Text
      32b:
        model_size: 20GB
        context_length: 32K
        capabilities:
          - Text
`
