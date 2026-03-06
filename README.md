# VireFS

**VireFS** 是一个极简的 Go 文件系统抽象库。

它将**本地文件系统**和 **S3 兼容的对象存储**统一到同一套接口之下——你的业务代码只跟 `key` 打交道，不关心文件存在磁盘上还是云端。

---

## 定位

> **一句话**：用 key 管理文件，后端透明。

典型使用场景：你的项目既有本地文件（用户上传暂存、导出报表等），又有对象存储里的文件（图片、视频、备份等），文件的 key 存在你的数据库里，通过 VireFS 用同一套 API 操作它们。

```
你的业务代码 ──key──▶ VireFS ──▶ 本地磁盘 / S3 / MinIO / R2 / ...
```

## 架构

```mermaid
flowchart TB
    app["业务代码"]
    app -->|"key"| fsInterface["virefs.FS 接口"]

    subgraph backends [存储后端]
        localFS["LocalFS\n本地目录"]
        objectFS["ObjectFS\nS3 兼容对象存储"]
    end

    fsInterface --> localFS
    fsInterface --> objectFS

    localFS -->|"root + key"| disk["本地磁盘"]
    objectFS -->|"prefix + key"| s3["S3 / MinIO / R2"]

    subgraph optional [可选能力]
        presigner["Presigner\n预签名 URL"]
        copier["Copier\n高效复制"]
        accessFn["AccessFunc\nCDN / 自定义 URL"]
    end

    objectFS -.-> presigner
    objectFS -.-> copier
    localFS -.-> copier
    objectFS -.-> accessFn
```

## 核心概念

### Key

所有操作以 `key` 为寻址核心。key 是以 `/` 分隔的路径，例如 `photos/2026/cat.jpg`。

- 自动清理首尾 `/`、合并重复 `/`、解析 `.`
- 禁止 `..` 跳出，保证安全

### FS 接口

```go
type FS interface {
    Get(ctx, key)                        // 读取文件内容
    Put(ctx, key, reader, ...PutOption)  // 写入（支持 ContentType、Metadata）
    Delete(ctx, key)                     // 删除
    List(ctx, prefix)                    // 按前缀列举
    Stat(ctx, key)                       // 获取元信息
    Access(ctx, key)                     // 获取外部访问路径/URL
}
```

### 后端

| 后端 | 构造函数 | root 概念 |
|---|---|---|
| **LocalFS** | `NewLocalFS(rootDir, ...LocalOption)` | 指定的本地目录 |
| **ObjectFS** | `NewObjectFS(s3Client, bucket, ...ObjectOption)` | endpoint + bucket |

### 可选能力（类型断言）

| 接口 | 说明 | 实现者 |
|---|---|---|
| `Presigner` | 生成预签名上传/下载 URL | ObjectFS |
| `Copier` | 同后端高效复制 | LocalFS, ObjectFS |

### 错误模型

| 哨兵错误 | 含义 |
|---|---|
| `ErrNotFound` | key 不存在 |
| `ErrInvalidKey` | key 包含非法模式（如 `..`） |
| `ErrAlreadyExist` | 资源已存在（保留） |
| `ErrNotSupported` | 当前后端不支持此操作 |

所有后端错误都被包装为 `*OpError{Op, Key, Err}`，方便定位问题。

## 快速上手

### 安装

```bash
go get github.com/lin-snow/VireFS
```

### 本地文件系统

```go
package main

import (
    "context"
    "fmt"
    "io"
    "strings"

    virefs "github.com/lin-snow/VireFS"
)

func main() {
    ctx := context.Background()
    fs := virefs.NewLocalFS("/tmp/mydata", virefs.WithCreateRoot())

    // 写入
    _ = fs.Put(ctx, "hello.txt", strings.NewReader("world"))

    // 读取
    rc, _ := fs.Get(ctx, "hello.txt")
    defer rc.Close()
    data, _ := io.ReadAll(rc)
    fmt.Println(string(data)) // "world"

    // 获取本地路径
    info, _ := fs.Access(ctx, "hello.txt")
    fmt.Println(info.Path) // "/tmp/mydata/hello.txt"
}
```

### 对象存储（S3 / MinIO / R2）

```go
cfg, _ := config.LoadDefaultConfig(ctx)
client := s3.NewFromConfig(cfg, func(o *s3.Options) {
    o.BaseEndpoint = aws.String("https://s3.example.com")
    o.UsePathStyle = true
})

fs := virefs.NewObjectFS(client, "my-bucket", virefs.WithPrefix("uploads/"))

// 上传并指定 ContentType
_ = fs.Put(ctx, "photo.jpg", file,
    virefs.WithContentType("image/jpeg"),
    virefs.WithMetadata(map[string]string{"user": "alice"}),
)
```

## 功能详解

### Put 选项：ContentType 和 Metadata

ObjectFS 会将这些信息传递给 S3 PutObject；LocalFS 会忽略它们（本地文件系统无此概念）。

```go
_ = fs.Put(ctx, "report.pdf", file,
    virefs.WithContentType("application/pdf"),
    virefs.WithMetadata(map[string]string{"version": "2"}),
)
```

### Exists — 检查 key 是否存在

包级别便捷函数，内部调用 `Stat`，将 `ErrNotFound` 转为 `false`。

```go
ok, _ := virefs.Exists(ctx, fs, "maybe.txt")
```

### Copy — 文件复制

同后端复制走原生高效路径（S3 `CopyObject`、本地文件复制），跨后端自动退化为 `Get` + `Put`。

```go
// 同后端（S3 内部复制，无需下载再上传）
_ = virefs.Copy(ctx, objFS, "src.txt", objFS, "dst.txt")

// 跨后端（本地 → S3）
_ = virefs.Copy(ctx, localFS, "export.csv", objFS, "imports/export.csv",
    virefs.WithContentType("text/csv"),
)
```

### Access — 获取外部访问信息

核心 FS 接口方法，根据后端返回不同内容：

| 后端 | `AccessInfo.Path` | `AccessInfo.URL` |
|---|---|---|
| LocalFS | 绝对文件路径 | 空 |
| ObjectFS | 空 | 预签名/公开/CDN URL |

```go
// LocalFS
info, _ := localFS.Access(ctx, "doc.pdf")
fmt.Println(info.Path) // "/data/doc.pdf"

// ObjectFS（自动选择：AccessFunc > Presign > BaseURL）
info, _ = objFS.Access(ctx, "doc.pdf")
fmt.Println(info.URL)
```

自定义 CDN 域名：

```go
fs := virefs.NewObjectFS(client, "bucket",
    virefs.WithPrefix("assets/"),
    virefs.WithAccessFunc(func(key string) *virefs.AccessInfo {
        return &virefs.AccessInfo{URL: "https://cdn.example.com/" + key}
    }),
)
// Access("img/logo.png") → "https://cdn.example.com/assets/img/logo.png"
```

### 预签名 URL

通过 `Presigner` 可选接口，使用类型断言获取预签名能力：

```go
fs := virefs.NewObjectFS(client, "bucket",
    virefs.WithPresignClient(s3.NewPresignClient(client)),
)

if p, ok := fs.(virefs.Presigner); ok {
    get, _ := p.PresignGet(ctx, "secret.pdf", 15*time.Minute)
    put, _ := p.PresignPut(ctx, "upload.zip", 30*time.Minute)
    fmt.Println(get.URL, put.URL)
}
```

### 原子写入（LocalFS）

启用后 Put 先写临时文件，再原子 rename，防止并发写入数据损坏。

```go
fs := virefs.NewLocalFS("/data", virefs.WithAtomicWrite())
```

### Key 变换（KeyFunc）

在 `CleanKey` 之后、实际存储操作之前，对 key 进行自定义变换：

```go
fs := virefs.NewLocalFS("/data", virefs.WithLocalKeyFunc(func(key string) string {
    return time.Now().Format("2006/01/02") + "/" + key
}))
// Put("photo.jpg") → 实际写入 /data/2026/03/06/photo.jpg
```

### Schema — 声明式文件组织

用户数据库里只存简单的文件名（如 `cat.jpg`），但希望实际存储时按业务规则分目录。Schema 提供声明式的路由规则，按扩展名或自定义函数将文件归类到不同目录前缀。

```go
schema := virefs.NewSchema(
    virefs.RouteByExt("images/", ".jpg", ".jpeg", ".png", ".gif", ".webp"),
    virefs.RouteByExt("videos/", ".mp4", ".avi", ".mkv"),
    virefs.RouteByExt("docs/",   ".pdf", ".doc", ".docx"),
    virefs.DefaultRoute("other/"),
)

// 通过 WithLocalKeyFunc / WithObjectKeyFunc 接入
fs := virefs.NewLocalFS("/data", virefs.WithLocalKeyFunc(schema.Resolve))

fs.Put(ctx, "cat.jpg", r)       // → /data/images/cat.jpg
fs.Put(ctx, "report.pdf", r)    // → /data/docs/report.pdf
fs.Put(ctx, "readme.txt", r)    // → /data/other/readme.txt
```

对象存储同理：

```go
objFS := virefs.NewObjectFS(client, "bucket",
    virefs.WithPrefix("uploads/"),
    virefs.WithObjectKeyFunc(schema.Resolve),
)
// Put("cat.jpg") → S3 key: uploads/images/cat.jpg
```

路由规则按声明顺序匹配，第一个命中的生效。支持自定义匹配函数：

```go
virefs.RouteByFunc("archives/", func(key string) bool {
    return strings.HasSuffix(key, ".tar.gz") || strings.HasSuffix(key, ".zip")
})
```

### MountTable — 多后端路由（可选）

当需要通过单个 `FS` 接口操作多个后端时，使用 MountTable 按前缀路由：

```go
mt := virefs.NewMountTable()
mt.Mount("local", virefs.NewLocalFS("/data/files"))
mt.Mount("s3",    virefs.NewObjectFS(s3Client, "my-bucket"))

mt.Get(ctx, "local/reports/q1.csv")  // → LocalFS
mt.Put(ctx, "s3/images/logo.png", r) // → ObjectFS
```

## Key 处理流水线

```mermaid
flowchart LR
    raw["原始 key"] --> clean["CleanKey\n规范化 + 安全校验"]
    clean --> keyFunc["KeyFunc\n用户自定义变换"]
    keyFunc --> prefix["basePrefix\n拼接前缀"]
    prefix --> backend["存储后端操作"]
```

## 完整 Option 速查

### LocalFS

| Option | 说明 |
|---|---|
| `WithCreateRoot()` | root 目录不存在时自动创建 |
| `WithDirPerm(perm)` | 自动创建目录的权限（默认 0755） |
| `WithLocalKeyFunc(fn)` | key 变换函数 |
| `WithAtomicWrite()` | 启用原子写入 |

### ObjectFS

| Option | 说明 |
|---|---|
| `WithPrefix(p)` | 所有 key 添加前缀 |
| `WithObjectKeyFunc(fn)` | key 变换函数 |
| `WithPresignClient(pc)` | 启用预签名 URL |
| `WithBaseURL(url)` | Access 公开 URL 基地址 |
| `WithAccessExpires(d)` | Access 预签名默认过期时间 |
| `WithAccessFunc(fn)` | 自定义 Access URL 生成 |

### Put

| Option | 说明 |
|---|---|
| `WithContentType(ct)` | 设置 MIME 类型 |
| `WithMetadata(m)` | 设置自定义元数据 |

## License

See [LICENSE](LICENSE).
