# gcode

从 `.proto` 文件生成纯 Go struct 的代码生成工具。不依赖 `protoc`，不引入 protobuf 官方运行时，生成的代码是普通 Go struct，可直接用于 GORM、JSON 序列化和 gin HTTP 服务。

[English](README.md)

**为什么做 gcode？** 官方 `protoc-gen-go` 生成的 struct 包含运行时字段（`protoimpl.MessageState`、`sizeCache`、`unknownFields`），不适合直接作为 DAO 层数据结构，且强制依赖 `google.golang.org/protobuf` 运行时。gcode 生成的代码是纯 Go struct，零运行时依赖，可直接用于 GORM 操作、JSON 序列化和 gin HTTP binding，同时保持与官方 protobuf wire format 的完全兼容。

---

## 特性

- **不依赖 protoc** — 基于 `protocompile` 纯 Go 解析 proto 文件， `go install` 即用
- **wire format 兼容** — 生成的 `MarshalBinary`/`UnmarshalBinary` 与官方 protobuf 二进制格式完全兼容
- **JSON tag 内置** — 默认生成 `json:"camelCase"` tag，支持 `omitempty`/`ignore` 注解
- **GORM 支持** — 通过 `(gcode.message).gorm` 注解生成 gorm struct tag 和 `TableName()` 方法
- **validate 内置** — 复用 `buf/validate` 注解语法，生成 `Validate() error` 方法，无需额外工具
- **派生 message 自动生成** — 通过注解声明 update/create 派生 message，自动继承 validate 规则
- **gin HTTP adapter** — 生成 handler 工厂函数，与 service interface 解耦，路由完全由用户控制
- **TypeScript 生成** — 通过 `gcode gen-ts` 从 proto 文件生成 TypeScript interface、enum 和验证元数据
- **注释透传** — proto leading comment 全部透传到生成代码（struct、field、enum、service、handler）

---

## 安装

```bash
go install github.com/pinealctx/gcode/cmd/gcode@latest
```

验证安装：

```bash
gcode -h
```

---

## 项目搭建

创建 Go 模块和 proto 目录：

```bash
mkdir myapp && cd myapp
go mod init myapp
mkdir proto dao
```

---

## 快速开始

**1. 编写 proto 文件**

```proto
// proto/user.proto
syntax = "proto3";
package myapp;

// go_package 决定生成代码的 Go 包名：
//   "import/path;pkg" — 分号后的 pkg 是 Go 包名，
//   import/path 用于 gcode 的包解析，
//   -out 参数控制文件写入磁盘的位置。
option go_package = "myapp/dao;dao";

message User {
  string name = 1;
  int32  age  = 2;
}
```

**2. 生成代码**

```bash
gcode -in proto/ -out dao/
```

**3. 安装运行时依赖**

生成的代码会导入 gcode 模块的公开包：

```bash
go get github.com/pinealctx/gcode/runtime
```

> 基础 struct 生成只需 `runtime`。使用 `buf/validate` 注解需额外安装 `go get github.com/pinealctx/gcode/validateruntime`；使用 `service` + gin handler 需额外安装 `go get github.com/gin-gonic/gin` 和 `go get github.com/pinealctx/gcode/httpruntime`。
>
> **注意**：在执行 `go get` 之前运行 `go build` 会报缺少模块的错误，这是正常的——先执行 `go get`，再构建。

**4. 使用生成的 struct**

```go
package main

import (
    "fmt"
    "log"
    "myapp/dao"
)

func main() {
    u := &dao.User{Name: "Alice", Age: 30}

    // 序列化为 protobuf wire format
    wire, err := u.MarshalBinary()
    if err != nil {
        log.Fatal(err)
    }

    // 反序列化
    var u2 dao.User
    if err := u2.UnmarshalBinary(wire); err != nil {
        log.Fatal(err)
    }
    fmt.Println(u2.Name, u2.Age)
}
```

> **派生 message**：使用 `gcode.update_message` 或 `gcode.create_message` 注解时，需先运行 `gcode gen-proto -in proto/` 生成中间 proto 文件，再运行 `gcode -in proto/ -out dao/`。详见 [开箱即用指南 — 第二步](docs/getting-started.zh.md#第二步生成派生-proto)。

**5. 生成 TypeScript 类型（可选）**

如果需要前端类型定义：

```bash
gcode gen-ts -in proto/ -out ts/
```

生成 `.pb.ts` 文件，包含 TypeScript interface、enum 和验证元数据。详见 [开箱即用指南 — TypeScript 代码生成](docs/getting-started.zh.md#typescript-代码生成)。

---

## 文档

| 文档                                       | 说明                                            |
| ------------------------------------------ | ----------------------------------------------- |
| [开箱即用指南](docs/getting-started.zh.md) | 完整 8 步示例（proto 到 HTTP 服务）、注解速查表 |
| [架构概览](docs/architecture.zh.md)        | 整体流水线、各层职责、目录结构                  |
| [注解参考](docs/annotations.zh.md)         | 所有注解的详细说明和示例                        |
| [设计决策](docs/design-decisions.zh.md)    | 关键架构决策（ADR 风格，D1-D15）                |

---

## 已知限制

| 限制                                    | 说明                                                      |
| --------------------------------------- | --------------------------------------------------------- |
| 不支持 streaming rpc                    | 遇到 `stream` 关键字报错退出                              |
| 不支持 path param                       | HTTP handler 统一使用 `c.ShouldBindJSON`，不支持 URL 路径参数 |
| 不支持 `map`、`oneof`、well-known types | 使用这些特性的 proto 文件可能生成错误代码                 |
| 不支持跨 package 引用                   | 同 package 跨文件已支持，跨 package 未充分测试            |
| 仅支持 proto3                           | 不支持 proto2 语法                                        |

---

## 贡献

开发环境搭建、代码风格、PR 流程见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 更新日志

版本发布历史见 [CHANGELOG.md](CHANGELOG.md)。

---

## License

[MIT](LICENSE)
