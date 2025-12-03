# PDF 预览工具

将 PDF 页面渲染为图像，支持文字和图像混合渲染。

## ✨ 功能特性

- ✅ **文字渲染** - 在图像上渲染 PDF 文字内容
- ✅ **图像渲染** - 渲染 PDF 中的图像
- ✅ **混合渲染** - 正确处理文字和图像的混合显示
- ✅ **多种格式** - 支持 PNG、SVG 等格式
- ✅ **可调分辨率** - 自定义 DPI 设置
- ✅ **批量处理** - 一次渲染多个页面

## 🚀 使用方法

### 基本用法

```bash
# 使用默认文件 test/test.pdf
go run cmd/pdfpreview/main.go

# 指定 PDF 文件
go run cmd/pdfpreview/main.go -pdf document.pdf

# 或者直接传递文件路径
go run cmd/pdfpreview/main.go document.pdf
```

### 高级选项

```bash
# 预览前 5 页，使用 300 DPI
go run cmd/pdfpreview/main.go -pdf document.pdf -pages 5 -dpi 300

# 指定输出目录
go run cmd/pdfpreview/main.go -pdf file.pdf -output ./my_preview

# 不生成 SVG 和缩略图
go run cmd/pdfpreview/main.go -pdf file.pdf -no-svg -no-thumb

# 预览所有页面
go run cmd/pdfpreview/main.go -pdf file.pdf -pages 0

# 只生成高分辨率图像
go run cmd/pdfpreview/main.go -pdf file.pdf -dpi 300 -no-thumb -no-text
```

### 查看帮助

```bash
go run cmd/pdfpreview/main.go -h
```

## 📋 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-pdf` | - | PDF 文件路径 |
| `-output` | `文件名_preview` | 输出目录 |
| `-pages` | `3` | 预览的最大页数（0 表示全部） |
| `-dpi` | `150` | 渲染分辨率 DPI |
| `-thumb-dpi` | `72` | 缩略图分辨率 DPI |
| `-no-svg` | `false` | 不生成 SVG 预览 |
| `-no-text` | `false` | 不显示文本预览 |
| `-no-thumb` | `false` | 不生成缩略图 |

## 📁 输出文件

程序会在输出目录生成以下文件：

- `page_1.png`, `page_2.png`, ... - 高分辨率页面图像（包含文字和图像）
- `thumb_1.png`, `thumb_2.png`, ... - 缩略图（包含文字和图像）
- `page_1.svg` - SVG 格式（可在浏览器中查看）

## 🎨 渲染说明

### 文字渲染

- 使用 PDF 内容流解析提取文字位置
- 保持文字的原始位置和布局
- 使用基础字体渲染文字（7x13 像素）
- 文字颜色为黑色

### 图像渲染

- 提取 PDF 中的 XObject 图像
- 支持 FlateDecode、DCTDecode、JPXDecode 等压缩格式
- 自动缩放图像以匹配目标分辨率

### 混合渲染

1. 首先渲染图像（作为背景）
2. 然后在图像上渲染文字（作为前景）
3. 保持正确的 Z-order（层次顺序）

## 🔧 技术细节

### 坐标转换

PDF 使用笛卡尔坐标系（原点在左下角，Y 轴向上），而图像使用屏幕坐标系（原点在左上角，Y 轴向下）。程序会自动处理坐标转换：

```
图像 Y = 页面高度 - PDF Y
```

### 分辨率计算

```
缩放比例 = DPI / 72.0
图像宽度 = PDF 宽度 × 缩放比例
图像高度 = PDF 高度 × 缩放比例
```

### 文字定位

- 从 PDF 内容流中提取文字项和位置
- 按照 Y 坐标（从上到下）和 X 坐标（从左到右）排序
- 使用文本矩阵计算实际位置

## 📊 示例输出

```
正在打开 PDF 文件: test/test.pdf

=== PDF 文档信息 ===
版本: 1.4
页数: 14
创建日期: 2023-10-08 10:01:10

=== 渲染页面预览（包含文字）===
正在渲染第 1/14 页...
  ✓ 已保存: test/test_preview/page_1.png (1240x1754)
正在渲染第 2/14 页...
  ✓ 已保存: test/test_preview/page_2.png (1240x1754)
正在渲染第 3/14 页...
  ✓ 已保存: test/test_preview/page_3.png (1240x1754)

=== 生成缩略图（包含文字）===
正在生成第 1/14 页缩略图...
  ✓ 已保存: test/test_preview/thumb_1.png (595x842)
...

=== 完成 ===
预览文件已保存到: test/test_preview
```

## ⚠️ 已知限制

1. **字体** - 目前使用基础字体，不支持 PDF 中的自定义字体
2. **字体大小** - 文字大小固定，不根据 PDF 字体大小调整
3. **颜色** - 文字颜色固定为黑色
4. **特殊效果** - 不支持文字旋转、倾斜等变换
5. **复杂布局** - 对于复杂的多栏布局可能不够精确

## 🔮 未来改进

- [ ] 支持 PDF 字体加载和渲染
- [ ] 支持文字颜色和样式
- [ ] 支持文字变换（旋转、缩放）
- [ ] 改进文字定位精度
- [ ] 支持更多图像格式
- [ ] 添加进度条显示
- [ ] 支持并行渲染

## 💡 提示

- 对于高质量预览，使用 `-dpi 300` 或更高
- 对于快速预览，使用 `-dpi 72` 或 `-dpi 96`
- 缩略图默认使用 72 DPI，适合快速浏览
- SVG 格式适合在浏览器中查看，支持缩放

## 🐛 问题排查

### 文字显示不正确

- 检查 PDF 是否使用了特殊字体
- 尝试提高 DPI 以获得更好的文字清晰度
- 查看 SVG 输出以对比文字内容

### 图像显示不正确

- 检查 PDF 图像的压缩格式
- 某些特殊格式可能不支持
- 查看控制台错误信息

### 性能问题

- 减少预览页数 `-pages 1`
- 降低 DPI `-dpi 72`
- 禁用不需要的输出 `-no-thumb -no-svg`
