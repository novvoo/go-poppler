package pdf

import (
	"testing"
)

func TestMultiColumnDetector(t *testing.T) {
	// Create test data with two columns
	items := []textItem{
		// Column 1
		{text: "Column", x: 50, y: 700},
		{text: "One", x: 50, y: 680},
		{text: "Text", x: 50, y: 660},
		// Column 2
		{text: "Column", x: 300, y: 700},
		{text: "Two", x: 300, y: 680},
		{text: "Text", x: 300, y: 660},
	}

	detector := NewMultiColumnDetector(600, 800)
	layout := detector.DetectColumns(items)

	if len(layout.Columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(layout.Columns))
	}

	if len(layout.Gaps) != 1 {
		t.Errorf("Expected 1 gap, got %d", len(layout.Gaps))
	}
}

func TestAdvancedTextLayout(t *testing.T) {
	items := []textItem{
		{text: "First", x: 50, y: 700},
		{text: "line", x: 100, y: 700},
		{text: "Second", x: 50, y: 680},
		{text: "line", x: 110, y: 680},
	}

	layout := NewAdvancedTextLayout(600, 800, items)
	err := layout.Coalesce()
	if err != nil {
		t.Errorf("Coalesce failed: %v", err)
	}

	text := layout.BuildText()
	if text == "" {
		t.Error("Expected non-empty text")
	}
}

func TestTextLayoutPreservation(t *testing.T) {
	items := []textItem{
		{text: "Left", x: 50, y: 700},
		{text: "Right", x: 400, y: 700},
		{text: "Bottom", x: 50, y: 600},
	}

	layout := NewTextLayout(600, 800, items)
	text := layout.BuildLayoutText()

	if text == "" {
		t.Error("Expected non-empty text")
	}

	// Check that spacing is preserved
	if len(text) < 20 {
		t.Error("Expected text with preserved spacing")
	}
}

func TestColumnDetection(t *testing.T) {
	// Test single column
	singleCol := []textItem{
		{text: "Line1", x: 50, y: 700},
		{text: "Line2", x: 50, y: 680},
	}

	detector := NewMultiColumnDetector(600, 800)
	layout := detector.DetectColumns(singleCol)

	if len(layout.Columns) != 1 {
		t.Errorf("Expected 1 column for single column text, got %d", len(layout.Columns))
	}
}

func TestAdvancedTextRenderer(t *testing.T) {
	renderer := NewAdvancedTextRenderer(150)

	// Test configuration
	renderer.SetKerning(true)
	renderer.SetSubpixelPositioning(true)
	renderer.SetAntiAliasing(true)

	// Just verify it doesn't crash
	if renderer.dpi != 150 {
		t.Errorf("Expected DPI 150, got %f", renderer.dpi)
	}
}

func TestTextBlockGrouping(t *testing.T) {
	items := []textItem{
		// Block 1
		{text: "Para1Line1", x: 50, y: 700},
		{text: "Para1Line2", x: 50, y: 680},
		// Block 2 (different X position)
		{text: "Para2Line1", x: 100, y: 650},
		{text: "Para2Line2", x: 100, y: 630},
	}

	layout := NewAdvancedTextLayout(600, 800, items)
	err := layout.Coalesce()
	if err != nil {
		t.Fatalf("Coalesce failed: %v", err)
	}

	blocks := layout.GetBlocks()
	if len(blocks) < 1 {
		t.Error("Expected at least 1 block")
	}
}

func TestTextFlowGrouping(t *testing.T) {
	items := []textItem{
		{text: "Flow1", x: 50, y: 700},
		{text: "Flow1", x: 50, y: 680},
		{text: "Flow2", x: 300, y: 700},
		{text: "Flow2", x: 300, y: 680},
	}

	layout := NewAdvancedTextLayout(600, 800, items)
	err := layout.Coalesce()
	if err != nil {
		t.Fatalf("Coalesce failed: %v", err)
	}

	flows := layout.GetFlows()
	if len(flows) < 1 {
		t.Error("Expected at least 1 flow")
	}
}
