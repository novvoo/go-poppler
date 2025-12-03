package test

import (
"path/filepath"
"testing"
"github.com/novvoo/go-poppler/pkg/pdf"
)

func BenchmarkPDFOpen(b *testing.B) {
testPDF := filepath.Join(".", "test.pdf")
b.ResetTimer()
for i := 0; i < b.N; i++ {
doc, err := pdf.Open(testPDF)
if err != nil {
b.Skipf("无法打开PDF: %v", err)
return
}
doc.Close()
}
}
