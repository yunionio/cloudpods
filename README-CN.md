# Cloudpods

[![CircleCI](https://circleci.com/gh/yunionio/yunioncloud.svg?style=svg)](https://circleci.com/gh/yunionio/yunioncloud) 
[![Build Status](https://travis-ci.com/yunionio/yunioncloud.svg?branch=master)](https://travis-ci.org/yunionio/yunioncloud) 
[![Go Report Card](https://goreportcard.com/badge/github.com/yunionio/yunioncloud)](https://goreportcard.com/report/github.com/yunionio/yunioncloud) 

## Cloudpods是什么?

Cloudpods是一个开源的Golang实现的云原生融合多云/混合云的云平台。Cloudpods可以理解为pod(豆荚) of clouds，有多云管理的寓意。Cloupods可以管理多个云平台和云账号。并且，Cloudpods隐藏了这些平台和账号之间的云资源的数据模型和API的差异，对外暴露了一套统一的API。从而大大降低了访问多云的复杂度，提升了管理多云资源的效率。

## 功能

Cloudpods提供了如下的功能：

* 一个可以管理海量KVM虚拟机的轻量级私有云
* 一个能进行物理机全生命周期管理的裸机云
* 实现了VMware vSphere虚拟化集群的自助服务和自动化
* 管理多云资源的功能，可以管理大多数的主流云，包括私有云，例如OpenStack，以及公有云，例如AWS，Azure，GCP，阿里云，华为云和腾讯云等
* 提供一组统一的功能丰富的REST API访问以上的云资源
* 一套完整的多租户认证和访问控制体系
* 多云镜像管理自动将镜像转换为不同云平台需要的格式

## 安装

我们可以通过以下简单三步将Cloudpods安装在一台至少8GiB内存和100GB硬盘的Linux主机上：

### 准备SSH免密登录

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

### 安装ansible和git

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

### 安装cloudpods

请将脚本中的<host_ip>替换为该Linux主机的主IP地址。

```bash
# Git clone the ocboot installation tool locally
$ git clone https://github.com/yunionio/ocboot && cd ./ocboot && ./run.py <host_ip>
```

大概10-30分钟后，安装完成。访问https://<host_ip>登入Cloudpods的Web控制台。初始的账号和密码为：admin/admin@123

请参考文档 [快速开始](https://docs.yunion.io/zh/docs/quickstart/) 获得更详细的安装指导。


## 文档

- [文档中心](https://docs.yunion.io/zh)

- [Swagger API文档](https://docs.yunion.cn/api/)


## 规划

请访问[Cloudpods Roadmap](https://docs.yunion.io/zh/docs/roadmap/).


## 贡献

我们非常欢迎和感谢开发者向项目做贡献，流程细节请查看 [CONTRIBUTING.md](./CONTRIBUTING.md) 。


## License

Cloudpods 使用 Apache license 2.0. 详情请看 [LICENSE](./LICENSE) 。

