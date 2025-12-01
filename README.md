# Go-Poppler

Go-Poppler 是 Poppler PDF 工具库的纯 Go 语言实现。无需外部 C 库依赖，可跨平台编译使用。

## 功能特性

- ✅ 完整的 PDF 解析器（支持 PDF 1.0-2.0）
- ✅ 文本提取（多编码支持）
- ✅ 图像提取和渲染
- ✅ 字体分析
- ✅ 表单字段提取
- ✅ 附件管理
- ✅ 数字签名验证
- ✅ 加密 PDF 支持（RC4/AES）
- ✅ 多种输出格式（HTML、PostScript、PNG、JPEG、PPM、TIFF、SVG）

## 安装

```bash
go get github.com/user/go-poppler
```

## 命令行工具

Go-Poppler 提供了与 Poppler 完全对应的 12 个命令行工具：

### pdftotext - PDF 转文本

```bash
go run cmd/pdftotext/main.go [选项] input.pdf [output.txt]

选项:
  -f int     起始页码 (默认 1)
  -l int     结束页码 (默认 0，表示最后一页)
  -layout    保持原始布局
  -raw       按内容流顺序输出
  -htmlmeta  输出 HTML 格式（带元数据）
  -bbox      输出带边界框的 HTML
  -enc       输出编码 (默认 UTF-8)
  -eol       行尾格式: unix, dos, mac
  -nopgbrk   不插入分页符
  -opw       所有者密码
  -upw       用户密码
```

### pdfinfo - PDF 信息

```bash
go run cmd/pdfinfo/main.go [选项] input.pdf

选项:
  -f int     起始页码
  -l int     结束页码
  -box       显示页面边界框
  -meta      显示 XMP 元数据
  -js        显示 JavaScript
  -rawdates  显示原始日期格式
  -enc       输出编码
  -opw       所有者密码
  -upw       用户密码
```

### pdffonts - 字体列表

```bash
go run cmd/pdffonts/main.go [选项] input.pdf

选项:
  -f int     起始页码
  -l int     结束页码
  -subst     显示字体替换
  -opw       所有者密码
  -upw       用户密码
```

### pdfimages - 图像提取

```bash
go run cmd/pdfimages/main.go [选项] input.pdf output-root

选项:
  -f int     起始页码
  -l int     结束页码
  -j         输出 JPEG 格式
  -png       输出 PNG 格式
  -tiff      输出 TIFF 格式
  -all       提取所有图像
  -list      仅列出图像信息
  -opw       所有者密码
  -upw       用户密码
```

### pdftoppm - PDF 转 PPM 图像

```bash
go run cmd/pdftoppm/main.go [选项] input.pdf output-root

选项:
  -f int     起始页码
  -l int     结束页码
  -r int     分辨率 DPI (默认 150)
  -rx int    X 方向分辨率
  -ry int    Y 方向分辨率
  -scale-to int  缩放到指定大小
  -x int     裁剪 X 坐标
  -y int     裁剪 Y 坐标
  -W int     裁剪宽度
  -H int     裁剪高度
  -mono      单色输出
  -gray      灰度输出
  -png       输出 PNG 格式
  -jpeg      输出 JPEG 格式
  -tiff      输出 TIFF 格式
  -opw       所有者密码
  -upw       用户密码
```

### pdftocairo - Cairo 渲染

```bash
go run cmd/pdftocairo/main.go [选项] input.pdf [output]

选项:
  -f int     起始页码
  -l int     结束页码
  -r int     分辨率 DPI
  -scale-to int  缩放到指定大小
  -x int     裁剪 X 坐标
  -y int     裁剪 Y 坐标
  -W int     裁剪宽度
  -H int     裁剪高度
  -png       输出 PNG 格式
  -jpeg      输出 JPEG 格式
  -tiff      输出 TIFF 格式
  -ps        输出 PostScript 格式
  -eps       输出 EPS 格式
  -pdf       输出 PDF 格式
  -svg       输出 SVG 格式
  -opw       所有者密码
  -upw       用户密码
```

### pdftops - PDF 转 PostScript

```bash
go run cmd/pdftops/main.go [选项] input.pdf [output.ps]

选项:
  -f int     起始页码
  -l int     结束页码
  -level1    生成 Level 1 PostScript
  -level1sep 生成 Level 1 分色 PostScript
  -level2    生成 Level 2 PostScript
  -level2sep 生成 Level 2 分色 PostScript
  -level3    生成 Level 3 PostScript
  -level3sep 生成 Level 3 分色 PostScript
  -eps       生成 EPS 格式
  -form      生成 PostScript 表单
  -opi       生成 OPI 注释
  -r int     分辨率
  -paper     纸张大小
  -nocrop    不裁剪到 CropBox
  -expand    扩展到纸张大小
  -noshrink  不缩小到纸张大小
  -nocenter  不居中
  -duplex    双面打印
  -opw       所有者密码
  -upw       用户密码
```

### pdftohtml - PDF 转 HTML

```bash
go run cmd/pdftohtml/main.go [选项] input.pdf [output.html]

选项:
  -f int     起始页码
  -l int     结束页码
  -c         生成复杂 HTML（保持布局）
  -s         生成单个 HTML 文件
  -i         忽略图像
  -noframes  不生成框架
  -stdout    输出到标准输出
  -xml       输出 XML 格式
  -enc       输出编码
  -opw       所有者密码
  -upw       用户密码
```

### pdfseparate - 分离页面

```bash
go run cmd/pdfseparate/main.go [选项] input.pdf output-%d.pdf

选项:
  -f int     起始页码
  -l int     结束页码
```

### pdfunite - 合并 PDF

```bash
go run cmd/pdfunite/main.go input1.pdf input2.pdf ... output.pdf
```

### pdfattach - 附件管理

```bash
go run cmd/pdfattach/main.go [选项] input.pdf

选项:
  -list      列出附件
  -save      保存附件
  -savefile  保存指定附件
  -saveall   保存所有附件
  -o         输出目录
  -opw       所有者密码
  -upw       用户密码
```

### pdfsig - 签名验证

```bash
go run cmd/pdfsig/main.go [选项] input.pdf

选项:
  -nocert    不验证证书
  -nofail    即使验证失败也返回 0
  -dump      导出签名
  -opw       所有者密码
  -upw       用户密码
```

## 作为库使用

```go
package main

import (
    "fmt"
    "github.com/user/go-poppler/pkg/pdf"
)

func main() {
    // 打开 PDF 文件
    doc, err := pdf.Open("input.pdf")
    if err != nil {
        panic(err)
    }
    defer doc.Close()

    // 获取文档信息
    info := doc.GetInfo()
    fmt.Printf("标题: %s\n", info.Title)
    fmt.Printf("作者: %s\n", info.Author)
    fmt.Printf("页数: %d\n", doc.NumPages())

    // 提取文本
    extractor := pdf.NewTextExtractor(doc)
    for i := 1; i <= doc.NumPages(); i++ {
        text, _ := extractor.ExtractPage(i)
        fmt.Printf("第 %d 页:\n%s\n", i, text)
    }

    // 提取图像
    images := doc.ExtractImages()
    for i, img := range images {
        fmt.Printf("图像 %d: %dx%d, %d bpp\n", i+1, img.Width, img.Height, img.BitsPerComponent)
    }

    // 获取表单字段
    fields := doc.GetFormFields()
    for _, field := range fields {
        fmt.Printf("字段: %s = %s\n", field.Name, field.Value)
    }
}
```

## 与 Poppler 功能对比

### 命令行工具 (12/12 ✅)

| Poppler 工具 | Go-Poppler | 状态 |
|-------------|------------|------|
| pdftotext | ✅ | 已实现 |
| pdfinfo | ✅ | 已实现 |
| pdffonts | ✅ | 已实现 |
| pdfimages | ✅ | 已实现 |
| pdftoppm | ✅ | 已实现 |
| pdftocairo | ✅ | 已实现 |
| pdftops | ✅ | 已实现 |
| pdftohtml | ✅ | 已实现 |
| pdfseparate | ✅ | 已实现 |
| pdfunite | ✅ | 已实现 |
| pdfattach | ✅ | 已实现 |
| pdfsig | ✅ | 已实现 |

### 加密支持

| 加密类型 | 状态 |
|---------|------|
| RC4 40-bit | ✅ |
| RC4 128-bit | ✅ |
| AES-128 | ✅ |
| AES-256 | ✅ |

### 压缩格式支持

| 格式 | 状态 |
|-----|------|
| FlateDecode | ✅ |
| LZWDecode | ✅ |
| ASCII85Decode | ✅ |
| ASCIIHexDecode | ✅ |
| RunLengthDecode | ✅ |
| DCTDecode (JPEG) | ✅ |
| CCITTFaxDecode | ✅ |
| JBIG2Decode | ⚠️ 基础支持 |
| JPXDecode (JPEG2000) | ⚠️ 基础支持 |

### 输出格式

| 格式 | 状态 |
|-----|------|
| 纯文本 | ✅ |
| HTML | ✅ |
| XML | ✅ |
| PostScript | ✅ |
| EPS | ✅ |
| PPM | ✅ |
| PNG | ✅ |
| JPEG | ✅ |
| TIFF | ✅ |
| SVG | ✅ |
| PDF | ✅ |

## 项目结构

```
go-poppler/
├── go.mod                    # Go 模块定义
├── README.md                 # 项目说明
├── pkg/pdf/                  # PDF 核心库
│   ├── objects.go           # PDF 对象类型
│   ├── lexer.go             # 词法分析器
│   ├── parser.go            # 语法解析器
│   ├── document.go          # 文档处理
│   ├── text.go              # 文本提取
│   ├── font.go              # 字体处理
│   ├── image.go             # 图像处理
│   ├── render.go            # 渲染引擎
│   ├── html.go              # HTML 转换
│   ├── writer.go            # PDF 写入
│   ├── attachment.go        # 附件处理
│   ├── signature.go         # 数字签名
│   ├── crypto.go            # 加密支持
│   └── form.go              # 表单支持
└── cmd/                      # 命令行工具
    ├── pdftotext/
    ├── pdfinfo/
    ├── pdffonts/
    ├── pdfimages/
    ├── pdftoppm/
    ├── pdftocairo/
    ├── pdftops/
    ├── pdftohtml/
    ├── pdfseparate/
    ├── pdfunite/
    ├── pdfattach/
    └── pdfsig/
```

## 许可证

MIT License
