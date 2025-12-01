# go-poppler

[![Go Reference](https://pkg.go.dev/badge/github.com/novvoo/go-poppler.svg)](https://pkg.go.dev/github.com/novvoo/go-poppler)
[![Go Report Card](https://goreportcard.com/badge/github.com/novvoo/go-poppler)](https://goreportcard.com/report/github.com/novvoo/go-poppler)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Go 语言实现的 PDF 处理库，提供与 Poppler 兼容的命令行工具。

## 🚀 快速开始

```bash
# 安装
go get github.com/novvoo/go-poppler

# 编译所有工具
go build ./...

# 提取文本
./pdftotext document.pdf output.txt

# 查看 PDF 信息
./pdfinfo document.pdf

# 提取图像
./pdfimages -png document.pdf images/
```

## ✨ 特性

- **纯 Go 实现**：无需依赖外部 C 库，无 CGO
- **完整的 PDF 解析**：支持 PDF 1.0 - 2.0 规范
- **多种流解码**：FlateDecode、LZWDecode、ASCII85Decode、ASCIIHexDecode、RunLengthDecode、DCTDecode、JBIG2Decode
- **加密支持**：RC4 和 AES 加密/解密
- **文本提取**：支持多种字符编码和 CMap
- **图像提取**：支持 JPEG、PNG、JBIG2 等格式
- **页面渲染**：渲染为 PPM、PNG、JPEG 格式
- **表单处理**：读取和填写 PDF 表单
- **附件管理**：添加和提取嵌入文件
- **数字签名**：验证 PDF 签名
- **跨平台**：支持 Windows、Linux、macOS、ARM

## 📦 安装

### 作为库使用

```bash
go get github.com/novvoo/go-poppler
```

### 编译命令行工具

```bash
# 克隆仓库
git clone https://github.com/novvoo/go-poppler.git
cd go-poppler

# 编译所有命令行工具（推荐：禁用 CGO 以获得纯静态二进制）
CGO_ENABLED=0 go build ./...

# Windows 下使用
set CGO_ENABLED=0
go build ./...

# 或单独编译某个工具
CGO_ENABLED=0 go build ./cmd/pdftotext
CGO_ENABLED=0 go build ./cmd/pdfinfo
```

> **注意**：本项目是纯 Go 实现，不依赖任何 C 库。建议在编译时设置 `CGO_ENABLED=0` 以确保生成完全静态链接的二进制文件，便于跨平台部署。

### 安装到 GOPATH

```bash
go install github.com/novvoo/go-poppler/cmd/...@latest
```

## 🧪 测试

```bash
# 运行所有测试
go test ./...

# 运行测试并显示详细输出
go test -v ./pkg/pdf/...

# 运行测试并生成覆盖率报告
go test -cover ./pkg/pdf/...

# 生成 HTML 覆盖率报告
go test -coverprofile=coverage.out ./pkg/pdf/...
go tool cover -html=coverage.out -o coverage.html
```

## 🛠️ 命令行工具

### pdftotext - PDF 转文本

从 PDF 文件中提取文本内容。

```bash
pdftotext [选项] <PDF文件> [输出文件]

选项:
  -f <int>      起始页码 (默认: 1)
  -l <int>      结束页码 (默认: 最后一页)
  -layout       保持原始布局
  -raw          按内容流顺序输出
  -nopgbrk      不输出分页符
  -enc <string> 输出编码 (默认: UTF-8)
  -opw <string> 所有者密码
  -upw <string> 用户密码
```

### pdfinfo - PDF 信息

显示 PDF 文件的元数据和结构信息。

```bash
pdfinfo [选项] <PDF文件>

选项:
  -f <int>      起始页码
  -l <int>      结束页码
  -box          显示页面边界框
  -meta         显示 XMP 元数据
  -js           显示 JavaScript
  -struct       显示文档结构
  -dests        显示命名目标
  -enc <string> 文本编码
  -opw <string> 所有者密码
  -upw <string> 用户密码
```

### pdffonts - 字体信息

列出 PDF 文件中使用的所有字体。

```bash
pdffonts [选项] <PDF文件>

选项:
  -f <int>      起始页码
  -l <int>      结束页码
  -subst        显示字体替换
  -opw <string> 所有者密码
  -upw <string> 用户密码
```

### pdfimages - 图像提取

从 PDF 文件中提取图像。

```bash
pdfimages [选项] <PDF文件> <输出前缀>

选项:
  -f <int>      起始页码
  -l <int>      结束页码
  -j            导出为 JPEG
  -png          导出为 PNG
  -all          导出所有图像
  -list         仅列出图像信息
  -opw <string> 所有者密码
  -upw <string> 用户密码
```

### pdftoppm - PDF 转 PPM

将 PDF 页面渲染为 PPM/PNG/JPEG 图像。

```bash
pdftoppm [选项] <PDF文件> <输出前缀>

选项:
  -f <int>      起始页码
  -l <int>      结束页码
  -r <int>      分辨率 DPI (默认: 150)
  -rx <int>     X 方向分辨率
  -ry <int>     Y 方向分辨率
  -scale-to <int>     缩放到指定大小
  -scale-to-x <int>   缩放到指定宽度
  -scale-to-y <int>   缩放到指定高度
  -png          输出 PNG 格式
  -jpeg         输出 JPEG 格式
  -jpegopt <string>   JPEG 选项 (quality=N)
  -gray         灰度输出
  -mono         单色输出
  -opw <string> 所有者密码
  -upw <string> 用户密码
```

### pdftocairo - PDF 转图像

使用 Cairo 风格渲染 PDF 为多种格式。

```bash
pdftocairo [选项] <PDF文件> [输出文件]

选项:
  -f <int>      起始页码
  -l <int>      结束页码
  -r <int>      分辨率 DPI (默认: 150)
  -scale-to <int>     缩放到指定大小
  -scale-to-x <int>   缩放到指定宽度
  -scale-to-y <int>   缩放到指定高度
  -png          输出 PNG 格式
  -jpeg         输出 JPEG 格式
  -svg          输出 SVG 格式
  -pdf          输出 PDF 格式
  -ps           输出 PostScript 格式
  -eps          输出 EPS 格式
  -gray         灰度输出
  -mono         单色输出
  -opw <string> 所有者密码
  -upw <string> 用户密码
```

### pdftops - PDF 转 PostScript

将 PDF 转换为 PostScript 格式。

```bash
pdftops [选项] <PDF文件> [PS文件]

选项:
  -f <int>      起始页码
  -l <int>      结束页码
  -level1       生成 Level 1 PostScript
  -level1sep    生成 Level 1 分色 PostScript
  -level2       生成 Level 2 PostScript
  -level2sep    生成 Level 2 分色 PostScript
  -level3       生成 Level 3 PostScript
  -level3sep    生成 Level 3 分色 PostScript
  -eps          生成 EPS
  -form         生成 PostScript form
  -opi          生成 OPI 注释
  -noembt1      不嵌入 Type 1 字体
  -noembtt      不嵌入 TrueType 字体
  -noembcidps   不嵌入 CID PostScript 字体
  -noembcidtt   不嵌入 CID TrueType 字体
  -passfonts    传递字体不转换
  -duplex       双面打印
  -opw <string> 所有者密码
  -upw <string> 用户密码
```

### pdftohtml - PDF 转 HTML

将 PDF 转换为 HTML 格式。

```bash
pdftohtml [选项] <PDF文件> [输出目录]

选项:
  -f <int>      起始页码
  -l <int>      结束页码
  -c            生成复杂 HTML (保持布局)
  -s            生成单个 HTML 文件
  -i            忽略图像
  -noframes     不使用框架
  -stdout       输出到标准输出
  -xml          输出 XML 格式
  -enc <string> 输出编码 (默认: UTF-8)
  -opw <string> 所有者密码
  -upw <string> 用户密码
```

### pdfseparate - 拆分 PDF

将 PDF 文件拆分为单独的页面。

```bash
pdfseparate [选项] <PDF文件> <输出模式>

输出模式使用 %d 作为页码占位符，例如: page-%d.pdf

选项:
  -f <int>      起始页码
  -l <int>      结束页码
  -opw <string> 所有者密码
  -upw <string> 用户密码
```

### pdfunite - 合并 PDF

将多个 PDF 文件合并为一个。

```bash
pdfunite [选项] <PDF文件1> <PDF文件2> ... <输出文件>

选项:
  -opw <string> 所有者密码
  -upw <string> 用户密码
```

### pdfattach - 添加附件

向 PDF 文件添加嵌入文件附件。

```bash
pdfattach [选项] <PDF文件> <附件文件> <输出文件>

选项:
  -name <string>  附件显示名称
  -desc <string>  附件描述
  -opw <string>   所有者密码
  -upw <string>   用户密码
```

### pdfdetach - 提取附件

从 PDF 文件中提取嵌入的附件。

```bash
pdfdetach [选项] <PDF文件>

选项:
  -list         列出所有附件
  -save <int>   保存指定编号的附件
  -saveall      保存所有附件
  -o <string>   输出文件名
  -opw <string> 所有者密码
  -upw <string> 用户密码
```

### pdfthumbnail - 生成缩略图

生成 PDF 页面的缩略图。

```bash
pdfthumbnail [选项] <PDF文件> <输出前缀>

选项:
  -f <int>      起始页码 (默认: 1)
  -l <int>      结束页码 (默认: 最后一页)
  -size <int>   缩略图最大尺寸 (默认: 128)
  -format <string> 输出格式 png/jpeg (默认: png)
  -quality <int>   JPEG 质量 1-100 (默认: 85)
  -opw <string> 所有者密码
  -upw <string> 用户密码
```

### pdfsig - 签名验证

验证 PDF 文件中的数字签名。

```bash
pdfsig [选项] <PDF文件>

选项:
  -nocert       不验证证书
  -dump         导出签名
  -opw <string> 所有者密码
  -upw <string> 用户密码
```

## 📚 库使用示例

### 打开 PDF 文件

```go
package main

import (
    "fmt"
    "github.com/novvoo/go-poppler/pkg/pdf"
)

func main() {
    doc, err := pdf.Open("document.pdf")
    if err != nil {
        panic(err)
    }
    defer doc.Close()

    fmt.Printf("页数: %d\n", doc.NumPages())
    fmt.Printf("版本: %s\n", doc.GetVersion())
}
```

### 提取文本

```go
package main

import (
    "fmt"
    "github.com/novvoo/go-poppler/pkg/pdf"
)

func main() {
    doc, err := pdf.Open("document.pdf")
    if err != nil {
        panic(err)
    }
    defer doc.Close()

    extractor := pdf.NewTextExtractor(doc)
    
    for i := 1; i <= doc.NumPages(); i++ {
        text, err := extractor.ExtractText(i)
        if err != nil {
            continue
        }
        fmt.Printf("=== 第 %d 页 ===\n%s\n", i, text)
    }
}
```

### 提取图像

```go
package main

import (
    "fmt"
    "github.com/novvoo/go-poppler/pkg/pdf"
)

func main() {
    doc, err := pdf.Open("document.pdf")
    if err != nil {
        panic(err)
    }
    defer doc.Close()

    extractor := pdf.NewImageExtractor(doc)
    
    for i := 1; i <= doc.NumPages(); i++ {
        images, err := extractor.ExtractImages(i)
        if err != nil {
            continue
        }
        for j, img := range images {
            filename := fmt.Sprintf("image-%d-%d.png", i, j)
            extractor.SaveImage(img, filename, "png")
        }
    }
}
```

### 渲染页面

```go
package main

import (
    "fmt"
    "github.com/novvoo/go-poppler/pkg/pdf"
)

func main() {
    doc, err := pdf.Open("document.pdf")
    if err != nil {
        panic(err)
    }
    defer doc.Close()

    renderer := pdf.NewRenderer(doc)
    renderer.SetResolution(300, 300) // 300 DPI
    
    for i := 1; i <= doc.NumPages(); i++ {
        img, err := renderer.RenderPage(i)
        if err != nil {
            continue
        }
        renderer.SaveImage(img, fmt.Sprintf("page-%d.png", i), "png")
    }
}
```

### 处理加密 PDF

```go
package main

import (
    "github.com/novvoo/go-poppler/pkg/pdf"
)

func main() {
    doc, err := pdf.Open("encrypted.pdf")
    if err != nil {
        panic(err)
    }
    defer doc.Close()

    // 检查是否加密
    info := doc.GetInfo()
    if info.Encrypted {
        // 使用密码解密
        err = doc.Decrypt("password")
        if err != nil {
            panic(err)
        }
    }
}
```

### 读取表单字段

```go
package main

import (
    "fmt"
    "github.com/novvoo/go-poppler/pkg/pdf"
)

func main() {
    doc, err := pdf.Open("form.pdf")
    if err != nil {
        panic(err)
    }
    defer doc.Close()

    form := pdf.NewFormExtractor(doc)
    fields, err := form.GetFields()
    if err != nil {
        panic(err)
    }

    for _, field := range fields {
        fmt.Printf("字段: %s = %s\n", field.Name, field.Value)
    }
}
```

## 📁 项目结构

```
go-poppler/
├── pkg/pdf/              # 核心 PDF 库
│   ├── document.go       # PDF 文档解析
│   ├── document_test.go  # 文档测试
│   ├── parser.go         # PDF 对象解析器
│   ├── lexer.go          # 词法分析器
│   ├── lexer_test.go     # 词法分析器测试
│   ├── objects.go        # PDF 对象类型
│   ├── objects_test.go   # 对象测试
│   ├── text.go           # 文本提取
│   ├── font.go           # 字体处理
│   ├── image.go          # 图像提取
│   ├── render.go         # 页面渲染
│   ├── writer.go         # PDF 写入
│   ├── crypto.go         # 加密/解密
│   ├── form.go           # 表单处理
│   ├── annotation.go     # 注释处理
│   ├── attachment.go     # 附件处理
│   ├── signature.go      # 数字签名
│   ├── html.go           # HTML 转换
│   ├── xfa.go            # XFA 表单
│   ├── vector.go         # 矢量图形
│   ├── advanced.go       # 高级功能
│   └── jbig2.go          # JBIG2 解码器
│
├── cmd/                  # 命令行工具
│   ├── pdftotext/        # PDF 转文本
│   ├── pdfinfo/          # PDF 信息
│   ├── pdffonts/         # 字体信息
│   ├── pdfimages/        # 图像提取
│   ├── pdftoppm/         # PDF 转 PPM
│   ├── pdftocairo/       # PDF 转图像
│   ├── pdftops/          # PDF 转 PostScript
│   ├── pdftohtml/        # PDF 转 HTML
│   ├── pdfseparate/      # 拆分 PDF
│   ├── pdfunite/         # 合并 PDF
│   ├── pdfattach/        # 添加附件
│   ├── pdfdetach/        # 提取附件
│   ├── pdfsig/           # 签名验证
│   └── pdfthumbnail/     # 生成缩略图
│
├── go.mod
├── go.sum
└── README.md
```

## ✅ 支持的 PDF 特性

| 特性 | 状态 | 说明 |
|------|------|------|
| PDF 1.0 - 2.0 | ✅ | 完整支持 |
| 交叉引用表 | ✅ | 完整支持 |
| 交叉引用流 | ✅ | 完整支持 |
| 对象流 | ✅ | 完整支持 |
| FlateDecode | ✅ | zlib 压缩 |
| LZWDecode | ✅ | LZW 压缩 |
| ASCII85Decode | ✅ | ASCII85 编码 |
| ASCIIHexDecode | ✅ | 十六进制编码 |
| RunLengthDecode | ✅ | 游程编码 |
| DCTDecode (JPEG) | ✅ | JPEG 图像 |
| JBIG2Decode | ✅ | JBIG2 图像 |
| CCITTFaxDecode | ⚠️ | 仅返回原始数据 |
| JPXDecode (JPEG2000) | ⚠️ | 仅返回原始数据 |
| RC4 加密 | ✅ | 40/128-bit |
| AES 加密 | ✅ | 128/256-bit |
| Type1 字体 | ✅ | 完整支持 |
| TrueType 字体 | ✅ | 完整支持 |
| CID 字体 | ✅ | 完整支持 |
| CMap | ✅ | 字符映射 |
| ToUnicode | ✅ | Unicode 映射 |
| AcroForm | ✅ | 交互式表单 |
| XFA 表单 | ⚠️ | 基础解析 |
| 数字签名 | ✅ | 基础验证 |
| 嵌入文件 | ✅ | 附件支持 |
| 注释 | ✅ | 完整支持 |
| 书签 | ✅ | 大纲支持 |
| 链接 | ✅ | 完整支持 |

## 🔄 对标 Poppler

本项目旨在提供与 [Poppler](https://poppler.freedesktop.org/) 兼容的纯 Go 实现。

### 命令行工具对照

| Poppler 工具 | go-poppler 工具 | 功能 | 兼容性 |
|-------------|-----------------|------|--------|
| pdftotext | ✅ pdftotext | 提取文本 | 完整 |
| pdfinfo | ✅ pdfinfo | 显示信息 | 完整 |
| pdffonts | ✅ pdffonts | 列出字体 | 完整 |
| pdfimages | ✅ pdfimages | 提取图像 | 完整 |
| pdftoppm | ✅ pdftoppm | 转 PPM/PNG/JPEG | 完整 |
| pdftocairo | ✅ pdftocairo | 多格式渲染 | 部分 |
| pdftops | ✅ pdftops | 转 PostScript | 基础 |
| pdftohtml | ✅ pdftohtml | 转 HTML | 完整 |
| pdfseparate | ✅ pdfseparate | 拆分页面 | 完整 |
| pdfunite | ✅ pdfunite | 合并文件 | 完整 |
| pdfattach | ✅ pdfattach | 添加附件 | 完整 |
| pdfdetach | ✅ pdfdetach | 提取附件 | 完整 |
| pdfsig | ✅ pdfsig | 签名验证 | 基础 |
| - | ✅ pdfthumbnail | 生成缩略图 | 扩展功能 |

### 核心功能对比

| 功能领域 | Poppler | go-poppler | 备注 |
|---------|---------|------------|------|
| **PDF 解析** | | | |
| PDF 1.0-2.0 规范 | ✅ | ✅ | 完整支持 |
| 交叉引用表/流 | ✅ | ✅ | 完整支持 |
| 对象流 | ✅ | ✅ | 完整支持 |
| **流解码器** | | | |
| FlateDecode (zlib) | ✅ | ✅ | 完整支持 |
| LZWDecode | ✅ | ✅ | 完整支持 |
| ASCII85/Hex | ✅ | ✅ | 完整支持 |
| DCTDecode (JPEG) | ✅ | ✅ | 完整支持 |
| JBIG2Decode | ✅ | ✅ | 完整支持 |
| CCITTFaxDecode | ✅ | ⚠️ | 仅返回原始数据，无解码 |
| JPXDecode (JPEG2000) | ✅ | ⚠️ | 仅返回原始数据，无解码 |
| **加密** | | | |
| RC4 40/128-bit | ✅ | ✅ | 完整支持 |
| AES 128/256-bit | ✅ | ✅ | 完整支持 |
| **字体** | | | |
| Type1/TrueType | ✅ | ✅ | 完整支持 |
| CID 字体 | ✅ | ✅ | 完整支持 |
| CMap/ToUnicode | ✅ | ✅ | 完整支持 |
| **渲染** | | | |
| 栅格 (PPM/PNG/JPEG) | ✅ | ✅ | 完整支持 |
| 矢量 (PS/SVG/PDF) | ✅ | ⚠️ | 基础生成，无Cairo后端 |
| **表单** | | | |
| AcroForm | ✅ | ✅ | 完整支持 |
| XFA 表单 | ❌ | ⚠️ | 仅解析XML结构 |
| **签名** | | | |
| 基础验证 | ✅ | ✅ | 完整支持 |
| OCSP/CRL | ✅ | ❌ | 不支持 |

### go-poppler 优势

| 特性 | 说明 |
|------|------|
| 🔧 **纯 Go 实现** | 无 CGO 依赖，无需安装 Poppler 库 |
| 🌍 **跨平台编译** | 单一二进制，支持 Windows/Linux/macOS/ARM |
| 🐳 **容器友好** | 镜像体积小，无外部依赖 |
| 📦 **易于集成** | 作为 Go 库直接导入使用 |
| 🔗 **静态链接** | 部署简单，无动态库依赖 |

### 已知限制

| 限制 | 影响 | 建议 |
|------|------|------|
| CCITTFaxDecode 基础支持 | 部分扫描 PDF 解码不完整 | 复杂扫描文档使用原版 Poppler |
| 无 Cairo 渲染后端 | 矢量输出质量受限 | 高质量 PS/SVG 使用原版 Poppler |
| 无 OCSP/CRL 验证 | 企业签名合规性不足 | 企业级签名验证使用原版 Poppler |
| 无 OCG 支持 | 多图层 PDF 无法管理 | CAD/工程图纸使用原版 Poppler |

### 适用场景

**✅ 推荐使用 go-poppler：**
- 需要纯 Go 实现，避免 CGO 复杂性
- 简单的文本/图像提取任务
- 跨平台部署（特别是 Windows）
- 容器化/Serverless 环境
- 嵌入式系统或资源受限环境
- 作为 Go 应用的库集成

**⚠️ 建议使用原版 Poppler：**
- 需要高质量矢量输出
- 处理复杂扫描文档
- 企业级签名验证
- 处理多图层 PDF

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

### 开发指南

```bash
# 克隆仓库
git clone https://github.com/novvoo/go-poppler.git
cd go-poppler

# 安装依赖
go mod download

# 运行测试
go test ./...

# 构建（推荐禁用 CGO）
CGO_ENABLED=0 go build ./...

# Windows 下构建
set CGO_ENABLED=0
go build ./...

# 交叉编译示例
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./...
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...
```

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 🙏 致谢

- [Poppler](https://poppler.freedesktop.org/) - 本项目的参考实现
- [PDF Reference](https://www.adobe.com/devnet/pdf/pdf_reference.html) - Adobe PDF 规范
