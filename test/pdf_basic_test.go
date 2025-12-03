package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

// TestPDFOpen 测试PDF文件打开功能
func TestPDFOpen(t *testing.T) {
	testPDF := filepath.Join(".", "test.pdf")

	doc, err := pdf.Open(testPDF)
	if err != nil {
		t.Fatalf("无法打开PDF文件: %v", err)
	}
	defer doc.Close()

	if doc == nil {
		t.Fatal("文档对象为nil")
	}

	t.Logf("成功打开PDF文件: %s", testPDF)
}

// TestPDFInfo 测试PDF信息获取
func TestPDFInfo(t *testing.T) {
	testPDF := filepath.Join(".", "test.pdf")

	doc, err := pdf.Open(testPDF)
	if err != nil {
		t.Skipf("跳过测试，无法打开PDF: %v", err)
		return
	}
	defer doc.Close()

	numPages := doc.NumPages()
	if numPages <= 0 {
		t.Errorf("页数应该大于0，实际: %d", numPages)
	}

	version := doc.GetVersion()
	t.Logf("PDF版本: %s", version)
	t.Logf("页数: %d", numPages)

	info := doc.GetInfo()
	t.Logf("标题: %s", info.Title)
	t.Logf("作者: %s", info.Author)
	t.Logf("创建日期: %s", info.CreationDate)
}

// TestTextExtraction 测试文本提取功能
func TestTextExtraction(t *testing.T) {
	testPDF := filepath.Join(".", "test.pdf")

	doc, err := pdf.Open(testPDF)
	if err != nil {
		t.Skipf("跳过测试，无法打开PDF: %v", err)
		return
	}
	defer doc.Close()

	extractor := pdf.NewTextExtractor(doc)
	if extractor == nil {
		t.Fatal("文本提取器创建失败")
	}

	// 测试第一页文本提取
	if doc.NumPages() > 0 {
		text, err := extractor.ExtractPageText(1)
		if err != nil {
			t.Errorf("提取第1页文本失败: %v", err)
		} else {
			t.Logf("第1页文本长度: %d 字符", len(text))
			if len(text) > 100 {
				t.Logf("文本预览: %s...", text[:100])
			} else if len(text) > 0 {
				t.Logf("文本内容: %s", text)
			}
		}
	}
}

// TestTextExtractionAllPages 测试提取所有页面文本
func TestTextExtractionAllPages(t *testing.T) {
	testPDF := filepath.Join(".", "test.pdf")

	doc, err := pdf.Open(testPDF)
	if err != nil {
		t.Skipf("跳过测试: %v", err)
		return
	}
	defer doc.Close()

	extractor := pdf.NewTextExtractor(doc)
	text, err := extractor.ExtractText()
	if err != nil {
		t.Errorf("提取所有页面文本失败: %v", err)
	} else {
		t.Logf("提取的总文本长度: %d 字符", len(text))
	}
}

// TestTextExtractionToFile 测试文本提取到文件
func TestTextExtractionToFile(t *testing.T) {
	testPDF := filepath.Join(".", "test.pdf")
	outputFile := filepath.Join(".", "test_extracted.txt")

	// 清理旧文件
	os.Remove(outputFile)
	defer os.Remove(outputFile)

	doc, err := pdf.Open(testPDF)
	if err != nil {
		t.Skipf("跳过测试: %v", err)
		return
	}
	defer doc.Close()

	extractor := pdf.NewTextExtractor(doc)

	// 创建输出文件
	f, err := os.Create(outputFile)
	if err != nil {
		t.Fatalf("无法创建输出文件: %v", err)
	}
	defer f.Close()

	// 提取所有页面文本
	for i := 1; i <= doc.NumPages(); i++ {
		text, err := extractor.ExtractPageText(i)
		if err != nil {
			t.Logf("警告: 提取第%d页失败: %v", i, err)
			continue
		}
		f.WriteString(text)
		f.WriteString("\n\n")
	}

	// 验证文件已创建
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("输出文件未创建")
	} else {
		t.Logf("文本已提取到: %s", outputFile)
	}
}

// TestImageExtraction 测试图像提取功能
func TestImageExtraction(t *testing.T) {
	testPDF := filepath.Join(".", "test.pdf")

	doc, err := pdf.Open(testPDF)
	if err != nil {
		t.Skipf("跳过测试: %v", err)
		return
	}
	defer doc.Close()

	extractor := pdf.NewImageExtractor(doc)
	if extractor == nil {
		t.Fatal("图像提取器创建失败")
	}

	numPages := doc.NumPages()
	if numPages > 0 {
		images, err := extractor.ExtractImages(1, numPages)
		if err != nil {
			t.Logf("警告: 提取图像失败: %v", err)
		} else {
			t.Logf("总共提取 %d 张图像", len(images))
			for i, img := range images {
				if i < 3 { // 只显示前3张
					t.Logf("图像 %d: %dx%d, 类型: %s", i+1, img.Width, img.Height, img.Type)
				}
			}
		}
	}
}

// TestPDFEncryption 测试加密PDF处理
func TestPDFEncryption(t *testing.T) {
	testPDF := filepath.Join(".", "test.pdf")

	doc, err := pdf.Open(testPDF)
	if err != nil {
		t.Skipf("跳过测试: %v", err)
		return
	}
	defer doc.Close()

	info := doc.GetInfo()
	t.Logf("PDF加密状态: %v", info.Encrypted)
	if info.Encrypted {
		t.Log("此PDF已加密")
	} else {
		t.Log("此PDF未加密")
	}
}

// TestPageRendering 测试页面渲染功能
func TestPageRendering(t *testing.T) {
	testPDF := filepath.Join(".", "test.pdf")

	doc, err := pdf.Open(testPDF)
	if err != nil {
		t.Skipf("跳过测试: %v", err)
		return
	}
	defer doc.Close()

	options := pdf.RenderOptions{
		DPI:    150,
		Format: "png",
	}
	renderer := pdf.NewPageRenderer(doc, options)
	if renderer == nil {
		t.Fatal("渲染器创建失败")
	}

	// 测试渲染第一页
	if doc.NumPages() > 0 {
		rendered, err := renderer.RenderPage(1)
		if err != nil {
			t.Errorf("渲染第1页失败: %v", err)
		} else if rendered != nil {
			t.Logf("第1页渲染成功，尺寸: %dx%d", rendered.Width, rendered.Height)
		}
	}
}

// TestPageInfo 测试页面信息获取
func TestPageInfo(t *testing.T) {
	testPDF := filepath.Join(".", "test.pdf")

	doc, err := pdf.Open(testPDF)
	if err != nil {
		t.Skipf("跳过测试: %v", err)
		return
	}
	defer doc.Close()

	if doc.NumPages() > 0 {
		page, err := doc.GetPage(1)
		if err != nil {
			t.Errorf("获取第1页失败: %v", err)
		} else if page != nil {
			t.Logf("第1页信息:")
			t.Logf("  页码: %d", page.Number)
			t.Logf("  宽度: %.2f", page.Width())
			t.Logf("  高度: %.2f", page.Height())
			if page.Resources != nil {
				t.Log("  包含资源字典")
			}
		}
	}
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "不存在的文件",
			filename: "nonexistent.pdf",
			wantErr:  true,
		},
		{
			name:     "空文件名",
			filename: "",
			wantErr:  true,
		},
		{
			name:     "有效文件",
			filename: "test.pdf",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := pdf.Open(filepath.Join(".", tt.filename))
			if (err != nil) != tt.wantErr {
				t.Errorf("Open() error = %v, wantErr %v", err, tt.wantErr)
			}
			if doc != nil {
				doc.Close()
			}
		})
	}
}
