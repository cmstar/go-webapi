# go-webapi 文档

go-webapi 是一个基于管线模型的 WebAPI 框架，内置 SlimAPI 协议实现，支持灵活的定制和扩展。

## 目录

- [Getting Started](getting-started.md) 。
- [框架设计与扩展](architecture.md) —— 介绍框架的管线模型、核心接口，以及如何定制和扩展框架。
- [SlimAPI](slim-api.md) —— 阐述 SlimAPI 通信协议，包括请求/响应格式、参数处理、错误处理、文件上传、流式响应和客户端调用。
  - [接收文件](upload-file.md) —— 详细说明如果通过 `multipart/form-data` 类型的请求传递文件和 JSON 数据。
- [SlimAuth](slim-auth.md) —— 添加了签名校验的 SlimAPI 协议扩展，包括签名算法、服务端集成和客户端调用。

## 依赖库

- [go-conv](https://github.com/cmstar/go-conv) —— 类型转换。
- [go-errx](https://github.com/cmstar/go-errx) —— 错误处理，提供 `BizError` 等类型。
- [go-logx](https://github.com/cmstar/go-logx) —— 日志抽象。
- [chi](https://github.com/go-chi/chi) —— URL 路由。
