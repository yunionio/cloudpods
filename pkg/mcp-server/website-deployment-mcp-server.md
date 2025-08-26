---
sidebar_position: 6
---

# MCP Server部署

### 源码克隆

这里假设克隆源码的目录为 **/root/cloudpods**

```sh
# 克隆后端源代码
$ git clone https://github.com/yunionio/cloudpods.git
# 这里需要将代码切到自己想要的分支，这里设为 release/3.10
$ cd cloudpods && git checkout release/3.10 && cd /root
# 创建配置文件放置目录
$ mkdir -p /etc/yunion
```

## MCP Server初始化

1) 首先配置MCP Server的配置文件

```sh
# 编译mcp-server
$ cd /root/cloudpods && make cmd/mcp-server

# 编写mcp-server服务的配置文件
$ mkdir -p /etc/yunion/mcp-server
# 编写配置文件，注意根据实际情况修改Cloudpods API的认证信息
$ cat<<EOF >/etc/yunion/mcp-server/config.conf
# ==================== 服务器基础配置 ====================
server.host = localhost       # 服务器监听地址（默认：localhost）
server.port = 12001            # 服务器监听端口

# ==================== MCP 服务配置 ====================
mcp.name = cloudpods-mcp-server    # MCP 服务名称（默认：cloudpods-mcp-server）
mcp.version = 1.0.0                # MCP 服务版本（默认：1.0.0）
mcp.description = the mcp server of the cloudpods server  # MCP 服务描述（默认）

# ==================== 外部服务配置 ====================
external.cloudpods.base_url = "https://<ip_or_domain_of_apigatway>/api/s/identity/v3"  # 认证服务入口（将ip部分换成实际的cloudpods api网关地址）

# ==================== 日志配置 ====================
log.level = info     # 日志输出级别（默认：info；可选：debug, warn, error, fatal, panic）
log.format = json    # 日志格式（默认：json；可选：text）
EOF
```

2) 启动MCP Server服务

```sh
# 启动mcp-server服务
# 默认会从以下路径查找配置文件: /etc/yunion/mcp-server/config.conf, ./config/config.conf, ./config.conf
$ /root/cloudpods/_output/bin/mcp-server --log-level debug

# 或者使用 --conf 参数指定配置文件路径
$ /root/cloudpods/_output/bin/mcp-server --log-level debug --conf /etc/yunion/mcp-server/config.conf
```

## 验证服务

MCP Server启动后，可以通过以下方式验证服务是否正常运行：

```sh
# 检查服务是否监听在指定端口
$ curl http://localhost:12001

# 如果配置了正确的Cloudpods认证信息，可以尝试调用工具
$ curl http://localhost:12001/tools
```