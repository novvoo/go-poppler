# go-poppler

Go 语言实现的 PDF 处理库，对标 poppler-utils 工具集。

## 功能特性

- **pdftotext** - 从 PDF 提取文本
- **pdfinfo** - 显示 PDF 文档信息
- **pdfimages** - 从 PDF 提取图像
- **pdfseparate** - 将 PDF 分离为单页文件
- **pdfunite** - 合并多个 PDF 文件

## 安装

```bash
go install github.com/novvoo/go-poppler/cmd/pdftotext@latest
go install github.com/novvoo/go-poppler/cmd/pdfinfo@latest
go install github.com/novvoo/go-poppler/cmd/pdfimages@latest
go install github.com/novvoo/go-poppler/cmd/pdfseparate@latest
go install github.com/novvoo/go-poppler/cmd/pdfunite@latest
```

## 使用方法

### pdftotext - 文本提取

```bash
pdftotext [options] <PDF-file> [<text-file>]

选项:
  -f int        起始页码 (默认 1)
  -l int        结束页码 (默认 0，表示最后一页)
  -layout       保持原始布局
  -raw          保持内容流顺序
  -nopgbrk      不插入分页符
  -enc string   输出编码 (默认 "UTF-8")
  -eol string   行尾格式: unix, dos, mac (默认 "unix")
  -opw string   所有者密码
  -upw string   用户密码
  -h, -help     显示帮助
  -v            显示版本
```

### pdfinfo - 文档信息

```bash
pdfinfo [options] <PDF-file>

选项:
  -f int        起始页码 (默认 1)
  -l int        结束页码 (默认 0)
  -box          显示页面边界框
  -meta         显示元数据
  -js           显示 JavaScript
  -struct       显示结构信息
  -rawdates     显示原始日期格式
  -enc string   文本编码 (默认 "UTF-8")
  -opw string   所有者密码
  -upw string   用户密码
  -h, -help     显示帮助
  -v            显示版本
```

### pdfimages - 图像提取

```bash
pdfimages [options] <PDF-file> <image-root>

选项:
  -f int        起始页码 (默认 1)
  -l int        结束页码 (默认 0)
  -png          输出 PNG 格式
  -j            输出 JPEG 格式
  -all          输出所有格式
  -list         仅列出图像信息
  -opw string   所有者密码
  -upw string   用户密码
  -h, -help     显示帮助
  -v            显示版本
```

### pdfseparate - 页面分离

```bash
pdfseparate [options] <PDF-file> <PDF-page-pattern>

PDF-page-pattern 应包含 %d 作为页码占位符

选项:
  -f int        起始页码 (默认 1)
  -l int        结束页码 (默认 0)
  -opw string   所有者密码
  -upw string   用户密码
  -h, -help     显示帮助
  -v            显示版本

示例:
  pdfseparate input.pdf page-%d.pdf
```

### pdfunite - 文件合并

```bash
pdfunite [options] <PDF-file-1> ... <PDF-file-n> <output-PDF>

选项:
  -h, -help     显示帮助
  -v            显示版本

示例:
  pdfunite page1.pdf page2.pdf page3.pdf output.pdf
```

## 作为库使用

```go
package main

import (
    "fmt"
    "github.com/novvoo/go-poppler/pkg/pdf"
)

func main() {
    // 打开 PDF
    doc, err := pdf.Open("document.pdf")
    if err != nil {
        panic(err)
    }
    defer doc.Close()

    // 获取文档信息
    fmt.Printf("页数: %d\n", doc.NumPages())
    fmt.Printf("标题: %s\n", doc.Info.Title)
    fmt.Printf("作者: %s\n", doc.Info.Author)

    // 提取文本
    extractor := pdf.NewTextExtractor(doc)
    text, _ := extractor.ExtractPage(1)
    fmt.Println(text)

    // 提取图像
    imgExtractor := pdf.NewImageExtractor(doc)
    images, _ := imgExtractor.ExtractPage(1)
    for i, img := range images {
        img.Save(fmt.Sprintf("image-%d.%s", i, img.Format))
    }

    // 分离页面
    pdf.ExtractPage(doc, 1, "page1.pdf")

    // 合并文件
    pdf.MergeFiles([]string{"a.pdf", "b.pdf"}, "merged.pdf")
}
```

## 支持的 PDF 特性

- PDF 1.0 - 1.7 版本
- 文本提取（支持多种编码）
- 图像提取（JPEG、PNG、PPM）
- 页面分离与合并
- 压缩流解码（FlateDecode、DCTDecode、ASCII85Decode、ASCIIHexDecode）
- 交叉引用表和流
- 文档信息字典

## 限制

- 不支持加密 PDF（密码保护）
- 不支持 PDF 2.0 特性
- 不支持 JBIG2、JPEG2000 压缩
- 不支持表单和注释提取

## 许可证

MIT License
