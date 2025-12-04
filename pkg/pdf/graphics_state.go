package pdf

import "math"

// TextGraphicsState 表示文本提取时的 PDF 图形状态
// 参考 Poppler 的 GfxState 实现
type TextGraphicsState struct {
	// 变换矩阵 (参考 Poppler GfxState)
	CTM [6]float64 // 当前变换矩阵 (Current Transformation Matrix)

	// 文本状态 (参考 Poppler 文本状态管理)
	TextMatrix [6]float64 // 文本矩阵 - 控制文本位置和方向
	LineMatrix [6]float64 // 文本行矩阵 - 用于换行
	FontSize   float64    // 字体大小
	CharSpace  float64    // 字符间距 (Tc)
	WordSpace  float64    // 单词间距 (Tw)
	Scale      float64    // 水平缩放 (Tz) - 百分比
	Leading    float64    // 行间距 (TL)
	Rise       float64    // 文本上升 (Ts) - 用于上标/下标

	// 文本位置 (参考 Poppler 的文本位置跟踪)
	CurTextX float64 // 当前文本 X 坐标
	CurTextY float64 // 当前文本 Y 坐标

	// 字体
	Font     *Font
	FontDict Dictionary

	// 渲染模式 (参考 Poppler)
	RenderMode int // 0=填充, 1=描边, 2=填充+描边, 3=不可见, 4=填充+裁剪...
}

// NewTextGraphicsState 创建新的图形状态
// 参考 Poppler 的默认状态初始化
func NewTextGraphicsState() *TextGraphicsState {
	return &TextGraphicsState{
		CTM:        [6]float64{1, 0, 0, 1, 0, 0}, // 单位矩阵
		TextMatrix: [6]float64{1, 0, 0, 1, 0, 0},
		LineMatrix: [6]float64{1, 0, 0, 1, 0, 0},
		Scale:      100, // 100% 水平缩放
		FontSize:   12,  // 默认字体大小
		CharSpace:  0,   // 无额外字符间距
		WordSpace:  0,   // 无额外单词间距
		Leading:    0,   // 无行间距
		Rise:       0,   // 无文本上升
		CurTextX:   0,   // 初始文本位置
		CurTextY:   0,
		RenderMode: 0, // 默认填充模式
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
// 参考 Poppler 的 state->getCurTextX/Y() + CTM 变换
func (gs *TextGraphicsState) GetTextPosition() (float64, float64) {
	// 使用当前文本位置（已经在用户空间）
	textX := gs.CurTextX
	textY := gs.CurTextY

	// 应用 CTM 转换到设备空间
	return gs.Transform(textX, textY)
}

// GetTextPositionWithRise 获取包含文本上升的文本位置
// 参考 Poppler 的 riseX/riseY 计算
func (gs *TextGraphicsState) GetTextPositionWithRise() (float64, float64) {
	// 计算文本上升的偏移
	riseX, riseY := gs.TextTransformDelta(0, gs.Rise)

	// 应用到当前位置
	x := gs.CurTextX + riseX
	y := gs.CurTextY + riseY

	// 转换到设备空间
	return gs.Transform(x, y)
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
// 参考 Poppler 的渲染模式定义
func (gs *TextGraphicsState) GetRenderMode() int {
	return gs.RenderMode
}

// AdvanceTextPosition 前进文本位置
// 参考 Poppler 的 state->textShiftWithUserCoords()
func (gs *TextGraphicsState) AdvanceTextPosition(dx, dy float64) {
	// 应用文本矩阵变换
	tdx, tdy := gs.TextTransformDelta(dx, dy)

	// 更新当前文本位置
	gs.CurTextX += tdx
	gs.CurTextY += tdy

	// 同时更新文本矩阵的平移部分
	gs.TextMatrix[4] += tdx
	gs.TextMatrix[5] += tdy
}

// CalculateCharAdvance 计算字符前进量
// 参考 Poppler 的 doShowText 中的字符前进计算
func (gs *TextGraphicsState) CalculateCharAdvance(charWidth float64, isSpace bool) (float64, float64) {
	// 基础宽度 * 字体大小
	dx := charWidth * gs.FontSize

	// 添加字符间距
	dx += gs.CharSpace

	// 如果是空格，添加单词间距
	if isSpace {
		dx += gs.WordSpace
	}

	// 应用水平缩放
	dx *= gs.Scale / 100.0

	// 垂直方向通常为 0（水平书写模式）
	dy := 0.0

	return dx, dy
}

// IsSingularMatrix 检查 CTM 是否奇异（不可逆）
// 参考 Poppler 的奇异矩阵检测
func (gs *TextGraphicsState) IsSingularMatrix() bool {
	// 计算行列式
	det := gs.CTM[0]*gs.CTM[3] - gs.CTM[1]*gs.CTM[2]

	// 如果行列式接近 0，矩阵奇异
	return math.Abs(det) < 0.000001
}

// 矩阵运算辅助函数
// 参考 Poppler 的 Matrix 类

// matrixDeterminant 计算矩阵行列式
// 参考 Poppler 的 Matrix::determinant()
func matrixDeterminant(m [6]float64) float64 {
	return m[0]*m[3] - m[1]*m[2]
}

// getRotationFromMatrix 从矩阵获取旋转角度
// 参考 Poppler 的旋转检测
func getRotationFromMatrix(m [6]float64) int {
	angle := math.Atan2(m[1], m[0]) * 180 / math.Pi

	if angle >= -45 && angle < 45 {
		return 0 // 0°
	} else if angle >= 45 && angle < 135 {
		return 1 // 90°
	} else if angle >= 135 || angle < -135 {
		return 2 // 180°
	} else {
		return 3 // 270°
	}
}
