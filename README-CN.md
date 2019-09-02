# Yunion OneCloud (云联融合云）

[![CircleCI](https://circleci.com/gh/yunionio/onecloud.svg?style=svg)](https://circleci.com/gh/yunionio/onecloud) 
[![Build Status](https://travis-ci.com/yunionio/onecloud.svg?branch=master)](https://travis-ci.org/yunionio/onecloud) 
[![Go Report Card](https://goreportcard.com/badge/github.com/yunionio/onecloud)](https://goreportcard.com/report/github.com/yunionio/onecloud) 

## 什么是云联融合云?

云联融合云（Yunion OneCloud）是一个开源的多云平台。融合云构建在用户分布于多云基础设施之上，通过技术手段将分布于多云的异构IT资源统一管理，将多底层多云的差异向用户屏蔽，并通过网络和调度实现资源的融合与打通，在多云之上进行抽象，向用户呈现统一的使用界面和API接口，让用户就像使用一个云平台一样使用分布于多云的资源，实现一个统一的“云上之云”的云平台。

鉴于企业IT基础设施的两个趋势：1. 企业使用多云基础设施，并且公有云是最主要的基础设施提供者；2. 企业应用将逐步云原生化，企业IT基础设施需要为Kubernetes优化。融合云为此架构而设计。一方面抽象管理底层的多云基础设施，另一方面为多云Kubernetes提供运行环境。融合云为企业未来的的IT基础设施架构而设计。

云联融合云具备以下功能特性:

- 多云资源统一管理

  统一API、镜像、调度、账号体系、监控和计费等操作，能够全面管理 On premises、 私有云、公有云资源。

- 内置私有云

  OneCloud内置完备的私有云实现，提供对用户本地IDC的虚拟机、物理机和负载均衡等资源管理。

- 为多云Kubernetes提供运行环境

  OneCloud自身为运行在Kubernetes的云原生应用，并且能在多云环境部署运行Kubernetes集群。

欢迎安装部署云联融合云，期待大家的反馈！


## 部署

请参考文档 [安装部署](https://docs.yunion.io/docs/setup/) 搭建 OneCloud 集群。


## 文档

- [文档中心](https://docs.yunion.io/)

- [Swagger API文档](https://docs.yunion.cn/api/)


## 架构

![architecture](./docs/architecture.png)


## 贡献

我们非常欢迎和感谢开发者向项目做贡献，流程细节请查看 [CONTRIBUTING.md](./CONTRIBUTING.md) 。


## License

OneCloud 使用 Apache license 2.0. 详情请看 [LICENSE](./LICENSE) 。
