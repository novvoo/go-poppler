package pdf

import "math"

// TextGraphicsState 表示文本提取时的 PDF 图形状态
type TextGraphicsState struct {
	// 变换矩阵
	CTM [6]float64 // 当前变换矩阵 (Current Transformation Matrix)

	// 文本状态
	TextMatrix [6]float64 // 文本矩阵
	LineMatrix [6]float64 // 文本行矩阵
	FontSize   float64
	CharSpace  float64 // 字符间距
	WordSpace  float64 // 单词间距
	Scale      float64 // 水平缩放
	Leading    float64 // 行间距
	Rise       float64 // 文本上升

	// 字体
	Font     *Font
	FontDict Dictionary
}

// NewTextGraphicsState 创建新的图形状态
func NewTextGraphicsState() *TextGraphicsState {
	return &TextGraphicsState{
		CTM:        [6]float64{1, 0, 0, 1, 0, 0}, // 单位矩阵
		TextMatrix: [6]float64{1, 0, 0, 1, 0, 0},
		LineMatrix: [6]float64{1, 0, 0, 1, 0, 0},
		Scale:      100,
		FontSize:   12,
	}
}

// Clone 克隆图形状态
func (gs *TextGraphicsState) Clone() *TextGraphicsState {
	clone := *gs
	return &clone
}

// Transform 应用 CTM 转换坐标
func (gs *TextGraphicsState) Transform(x, y float64) (float64, float64) {
	tx := gs.CTM[0]*x + gs.CTM[2]*y + gs.CTM[4]
	ty := gs.CTM[1]*x + gs.CTM[3]*y + gs.CTM[5]
	return tx, ty
}

// TransformDelta 转换增量（不包括平移）
func (gs *TextGraphicsState) TransformDelta(dx, dy float64) (float64, float64) {
	tdx := gs.CTM[0]*dx + gs.CTM[2]*dy
	tdy := gs.CTM[1]*dx + gs.CTM[3]*dy
	return tdx, tdy
}

// ConcatCTM 连接变换矩阵到 CTM
func (gs *TextGraphicsState) ConcatCTM(matrix [6]float64) {
	gs.CTM = multiplyMatrix(gs.CTM, matrix)
}

// SetTextMatrix 设置文本矩阵
func (gs *TextGraphicsState) SetTextMatrix(matrix [6]float64) {
	gs.TextMatrix = matrix
	gs.LineMatrix = matrix
}

// TranslateTextMatrix 平移文本矩阵
func (gs *TextGraphicsState) TranslateTextMatrix(tx, ty float64) {
	translation := [6]float64{1, 0, 0, 1, tx, ty}
	gs.LineMatrix = multiplyMatrix(gs.LineMatrix, translation)
	gs.TextMatrix = gs.LineMatrix
}

// GetTextPosition 获取当前文本位置（设备空间坐标）
func (gs *TextGraphicsState) GetTextPosition() (float64, float64) {
	// 文本矩阵的平移部分
	textX := gs.TextMatrix[4]
	textY := gs.TextMatrix[5]

	// 应用 CTM 转换到设备空间
	return gs.Transform(textX, textY)
}

// TextTransformDelta 转换文本空间的增量到用户空间
func (gs *TextGraphicsState) TextTransformDelta(dx, dy float64) (float64, float64) {
	// 应用文本矩阵（不包括平移）
	tdx := gs.TextMatrix[0]*dx + gs.TextMatrix[2]*dy
	tdy := gs.TextMatrix[1]*dx + gs.TextMatrix[3]*dy
	return tdx, tdy
}

// GetRotation 获取文本旋转角度（0, 1, 2, 3 对应 0°, 90°, 180°, 270°）
func (gs *TextGraphicsState) GetRotation() int {
	// 计算文本矩阵的旋转角度
	angle := math.Atan2(gs.TextMatrix[1], gs.TextMatrix[0]) * 180 / math.Pi

	if angle >= -45 && angle < 45 {
		return 0
	} else if angle >= 45 && angle < 135 {
		return 1
	} else if angle >= 135 || angle < -135 {
		return 2
	} else {
		return 3
	}
}

// GetRenderMode 获取文本渲染模式
func (gs *TextGraphicsState) GetRenderMode() int {
	// 简化实现，默认返回填充模式
	return 0
}
