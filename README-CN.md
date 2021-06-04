# Cloudpods

[![CircleCI](https://circleci.com/gh/yunionio/cloudpods.svg?style=svg)](https://circleci.com/gh/yunionio/cloudpods)
[![Build Status](https://travis-ci.com/yunionio/cloudpods.svg?branch=master)](https://travis-ci.org/yunionio/cloudpods)
[![Go Report Card](https://goreportcard.com/badge/github.com/yunionio/cloudpods)](https://goreportcard.com/report/github.com/yunionio/cloudpods)

## Cloudpods是什么?

Cloudpods是一个开源的Golang实现的云原生融合多云/混合云的云平台。Cloudpods可以理解为pod(豆荚) of clouds。Cloupods可以管理多个云平台和云账号。并且，Cloudpods隐藏了这些平台和账号之间的云资源的数据模型和API的差异，对外暴露了一套统一的API。从而大大降低了访问多云的复杂度，提升了管理多云资源的效率。

## 功能

Cloudpods提供了如下的功能：

* 管理多云资源的功能，可以管理大多数的主流云，包括私有云，例如OpenStack，以及公有云，例如AWS，Azure，GCP，阿里云，华为云和腾讯云等
* 允许以统一的联邦身份访问各个云平台的原生控制台的SSO
* 一个可以管理海量KVM虚拟机的轻量级私有云
* 一个能进行物理机全生命周期管理的裸机云
* 实现了VMware vSphere虚拟化集群的自助服务和自动化
* 一套功能丰富、统一一致的RESTAPI和模型访问以上的云资源和功能
* 一套完整的多租户认证和访问控制体系
* 自动将镜像转换为不同云平台需要的格式的多云镜像服务

## 安装

我们可以通过以下简单三步将Cloudpods安装在一台至少8GiB内存和100GB硬盘的Linux主机上（假设该主机的IP为 *10.168.26.216*）：

### 1. 准备SSH免密登录

```bash
# Generate the local ssh keypair
# (SKIP this stekp if you already have ~/.ssh/id_rsa.pub locally)
$ ssh-keygen
# Copy the generated ~/.ssh/id_rsa.pub public key to the machine to be deployed
$ ssh-copy-id -i ~/.ssh/id_rsa.pub root@10.168.26.216
# Try to login to the machine to be deployed without password,
# should be able to get the hostname of the deployed machine
# without entering the login password
$ ssh root@10.168.26.216 "hostname"
```

### 2. 安装ansible和git

#### CentOS
```bash
# Install ansible and git locally
$ yum install -y epel-release ansible git
```
#### Debian 10
```bash
# Install ansible locally
$ apt install -y ansible git
```

### 3. 安装Cloudpods

通过以下命令开始安装Cloudpods：

```bash
# Git clone the ocboot installation tool locally
$ git clone https://github.com/yunionio/ocboot && cd ./ocboot && ./run.py 10.168.26.216
```

大概10-30分钟后，安装完成。访问 https://10.168.26.216 登入Cloudpods的Web控制台。初始的账号为 *admin* ，密码为 *admin@123*

请参考文档 [快速开始](https://docs.yunion.io/zh/docs/quickstart/) 获得更详细的安装指导。


## 文档

- [Cloudpods文档](https://docs.yunion.io/zh)

- [Swagger API文档](https://docs.yunion.cn/api/)


## 开发计划

请访问[Cloudpods Roadmap](https://docs.yunion.io/zh/docs/roadmap/).


## 贡献

我们非常欢迎和感谢开发者向项目做贡献，流程细节请查看 [CONTRIBUTING](https://docs.yunion.io/zh/docs/contribute/contrib/) 。


## License

Apache license 2.0，详情请看 [LICENSE](./LICENSE)。

