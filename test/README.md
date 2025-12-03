# go-poppler 测试模块

本目录包含 go-poppler 项目的测试文件和测试数据。

## 📁 文件说明

- `test.pdf` - 测试用PDF文件
- `test.txt` - 原始文本文件
- `output.txt` - 实际输出文本
- `test_verify.txt` - 用于验证的参考文本
- `pdf_basic_test.go` - 基础PDF功能单元测试
- `pdf_advanced_test.go` - 高级PDF功能测试
- `pdf_markdown_test.go` - Markdown转换功能测试
- `integration_test.go` - 集成测试和端到端测试

## 🧪 运行测试

### 运行所有测试

```bash
go test ./test/...
```

### 运行Markdown测试

```bash
# 运行所有Markdown测试
go test -v ./test -run TestMarkdown

# 运行特定Markdown测试
go test -v ./test -run TestMarkdownConversion
go test -v ./test -run TestMarkdownWithImages
go test -v ./test -run TestMarkdownHeadingDetection
```

### 运行详细测试

```bash
go test -v ./test/...
```

### 运行特定测试

```bash
# 运行单个测试
go test -v ./test -run TestPDFOpen

# 运行匹配模式的测试
go test -v ./test -run TestText
```

### 运行基准测试

```bash
go test -bench=. ./test
```

### 生成测试覆盖率报告

```bash
# 生成覆盖率数据
go test -coverprofile=coverage.out ./test

# 查看覆盖率
go tool cover -func=coverage.out

# 生成HTML报告
go tool cover -html=coverage.out -o coverage.html
```

### 跳过长时间运行的测试

```bash
go test -short ./test
```

## 📋 测试类别

### 基础功能测试 (pdf_basic_test.go)

- **TestPDFOpen** - 测试PDF文件打开
- **TestPDFInfo** - 测试PDF信息获取
- **TestTextExtraction** - 测试文本提取
- **TestTextExtractionAllPages** - 测试提取所有页面文本
- **TestTextExtractionToFile** - 测试文本提取到文件
- **TestImageExtraction** - 测试图像提取
- **TestPDFEncryption** - 测试加密PDF处理
- **TestPageRendering** - 测试页面渲染
- **TestPageInfo** - 测试页面信息获取
- **TestErrorHandling** - 测试错误处理

### Markdown转换测试 (pdf_markdown_test.go)

- **TestMarkdownConversion** - 测试PDF转Markdown基本功能
- **TestMarkdownWithImages** - 测试带图像的Markdown转换
- **TestMarkdownPageRange** - 测试页面范围转换
- **TestMarkdownHeadingDetection** - 测试标题检测功能
- **TestMarkdownCustomOptions** - 测试自定义选项
- **TestMarkdownFrontMatter** - 测试文档元数据前言
- **TestConvertToMarkdown** - 测试便捷函数
- **TestMarkdownAllPages** - 测试转换所有页面
- **TestMarkdownErrorHandling** - 测试Markdown错误处理

### 高级功能测试 (pdf_advanced_test.go)

- **TestAdvancedFeatures** - 高级功能测试
- **TestConcurrentAccess** - 并发访问测试
- **TestMemoryUsage** - 内存使用测试

### 集成测试 (integration_test.go)

- **TestFullWorkflow** - 完整工作流测试
- **TestMultiPageProcessing** - 多页处理测试
- **TestOutputComparison** - 输出对比测试
- **TestLargeFileHandling** - 大文件处理测试

### 性能测试

- **BenchmarkTextExtraction** - 文本提取性能
- **BenchmarkPageRendering** - 页面渲染性能
- **BenchmarkMarkdownConversion** - Markdown转换性能

## 🎯 测试覆盖的功能

### PDF 基础功能
- ✅ 打开和关闭PDF文件
- ✅ 获取PDF版本信息
- ✅ 获取页数
- ✅ 获取元数据（标题、作者等）

### 文本处理
- ✅ 单页文本提取
- ✅ 多页文本提取
- ✅ 文本输出到文件
- ✅ 不同编码支持

### Markdown转换
- ✅ PDF转Markdown基本转换
- ✅ 标题自动检测
- ✅ 列表格式化（有序/无序）
- ✅ 代码块检测
- ✅ 图像提取和引用
- ✅ 页面范围选择
- ✅ 自定义分隔符
- ✅ 文档元数据前言
- ✅ 自定义图像前缀

### 图像处理
- ✅ 图像提取
- ✅ 图像格式识别
- ✅ 图像保存

### 渲染功能
- ✅ 页面渲染为图像
- ✅ 分辨率设置
- ✅ 不同格式输出

### 高级功能
- ✅ 加密PDF处理
- ✅ 字体信息提取
- ✅ 并发访问
- ✅ 错误处理

## 📊 测试数据

测试使用的PDF文件应包含：
- 多页内容
- 文本内容（中英文）
- 图像
- 不同字体

## 🔧 添加新测试

创建新测试时，请遵循以下规范：

```go
func TestNewFeature(t *testing.T) {
    // 1. 准备测试数据
    testPDF := filepath.Join(".", "test.pdf")
    
    // 2. 执行测试操作
    doc, err := pdf.Open(testPDF)
    if err != nil {
        t.Skipf("跳过测试: %v", err)
        return
    }
    defer doc.Close()
    
    // 3. 验证结果
    if result != expected {
        t.Errorf("期望 %v, 实际 %v", expected, result)
    }
    
    // 4. 记录日志
    t.Logf("测试完成: %v", result)
}
```

## 🐛 调试测试

### 查看详细输出

```bash
go test -v ./test -run TestName
```

### 使用调试器

```bash
# 使用 delve
dlv test ./test -- -test.run TestName
```

### 查看测试日志

测试日志会在使用 `-v` 标志时显示，包括：
- 测试步骤
- 中间结果
- 错误信息

## 📝 注意事项

1. **测试文件** - 确保 `test.pdf` 存在且可读
2. **权限** - 测试可能需要文件读写权限
3. **并发** - 某些测试涉及并发操作，可能需要更多时间
4. **跳过测试** - 如果测试文件不存在，测试会自动跳过
5. **清理** - 测试会自动清理临时文件

## 🚀 持续集成

在CI/CD环境中运行测试：

```bash
# GitHub Actions 示例
go test -v -race -coverprofile=coverage.out ./test/...
go tool cover -html=coverage.out -o coverage.html
```

## 📈 性能基准

运行性能测试并生成报告：

```bash
# 运行基准测试
go test -bench=. -benchmem ./test > bench.txt

# 比较性能
go test -bench=. -benchmem ./test > new_bench.txt
benchcmp old_bench.txt new_bench.txt
```

## 🤝 贡献

添加新测试时，请确保：
1. 测试命名清晰
2. 包含必要的注释
3. 处理错误情况
4. 清理测试资源
5. 更新本README文档
