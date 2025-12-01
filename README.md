# go-poppler

Go 语言实现的 PDF 处理库，提供与 Poppler 兼容的命令行工具。

## 特性

- **纯 Go 实现**：无需依赖外部 C 库
- **完整的 PDF 解析**：支持 PDF 1.0 - 2.0 规范
- **多种流解码**：FlateDecode、LZWDecode、ASCII85Decode、ASCIIHexDecode、RunLengthDecode、DCTDecode、JBIG2Decode
- **加密支持**：RC4 和 AES 加密/解密
- **文本提取**：支持多种字符编码和 CMap
- **图像提取**：支持 JPEG、PNG、JBIG2 等格式
- **页面渲染**：渲染为 PPM、PNG、JPEG 格式
- **表单处理**：读取和填写 PDF 表单
- **附件管理**：添加和提取嵌入文件
- **数字签名**：验证 PDF 签名

## 安装

```bash
go get github.com/novvoo/go-poppler
```

## 编译

```bash
# 编译所有命令行工具
go build ./...

# 或单独编译某个工具
go build ./cmd/pdftotext
```

## 命令行工具

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

## 库使用示例

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

## 项目结构

```
go-poppler/
├── pkg/pdf/           # 核心 PDF 库
│   ├── document.go    # PDF 文档解析
│   ├── parser.go      # PDF 对象解析器
│   ├── lexer.go       # 词法分析器
│   ├── objects.go     # PDF 对象类型
│   ├── text.go        # 文本提取
│   ├── font.go        # 字体处理
│   ├── image.go       # 图像提取
│   ├── render.go      # 页面渲染
│   ├── writer.go      # PDF 写入
│   ├── crypto.go      # 加密/解密
│   ├── form.go        # 表单处理
│   ├── annotation.go  # 注释处理
│   ├── attachment.go  # 附件处理
│   ├── signature.go   # 数字签名
│   ├── html.go        # HTML 转换
│   └── jbig2.go       # JBIG2 解码器
│
├── cmd/               # 命令行工具
│   ├── pdftotext/     # PDF 转文本
│   ├── pdfinfo/       # PDF 信息
│   ├── pdffonts/      # 字体信息
│   ├── pdfimages/     # 图像提取
│   ├── pdftoppm/      # PDF 转 PPM
│   ├── pdftocairo/    # PDF 转图像
│   ├── pdftops/       # PDF 转 PostScript
│   ├── pdftohtml/     # PDF 转 HTML
│   ├── pdfseparate/   # 拆分 PDF
│   ├── pdfunite/      # 合并 PDF
│   ├── pdfattach/     # 添加附件
│   ├── pdfdetach/     # 提取附件
│   ├── pdfsig/        # 签名验证
│   └── pdfthumbnail/  # 生成缩略图
│
├── go.mod
├── go.sum
└── README.md
```

## 支持的 PDF 特性

| 特性 | 状态 |
|------|------|
| PDF 1.0 - 2.0 | ✅ |
| 交叉引用表 | ✅ |
| 交叉引用流 | ✅ |
| 对象流 | ✅ |
| FlateDecode | ✅ |
| LZWDecode | ✅ |
| ASCII85Decode | ✅ |
| ASCIIHexDecode | ✅ |
| RunLengthDecode | ✅ |
| DCTDecode (JPEG) | ✅ |
| JBIG2Decode | ✅ |
| CCITTFaxDecode | ⚠️ 基础支持 |
| JPXDecode (JPEG2000) | ⚠️ 基础支持 |
| RC4 加密 | ✅ |
| AES 加密 | ✅ |
| Type1 字体 | ✅ |
| TrueType 字体 | ✅ |
| CID 字体 | ✅ |
| CMap | ✅ |
| ToUnicode | ✅ |
| AcroForm | ✅ |
| XFA 表单 | ⚠️ 基础支持 |
| 数字签名 | ✅ |
| 嵌入文件 | ✅ |
| 注释 | ✅ |
| 书签 | ✅ |
| 链接 | ✅ |

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！
