package pdf

import (
	"math"
	"sort"
)

// PopplerTextOutputDev 完全复刻 Poppler 的 TextOutputDev 实现
type PopplerTextOutputDev struct {
	doc      *Document
	page     *Page
	curWord  *PopplerTextWord
	rawWords *PopplerTextWord
	rawLastWord *PopplerTextWord
	pools    [4]*PopplerTextPool
	
	// 当前字体信息
	curFont     *PopplerTextFontInfo
	curFontSize float64
	
	// 字符位置
	charPos int
	
	// 页面尺寸
	pageWidth  float64
	pageHeight float64
	
	// 控制标志
	rawOrder bool
	diagonal bool
	discardDiag bool
	lastCharOverlap bool
	nest int
	nTinyChars int
	
	// 合并组合字符
	mergeCombining bool
	
	// 常量（来自 Poppler）
	minDupBreakOverlap  float64
	dupMaxPriDelta      float64
	dupMaxSecDelta      float64
	minWordBreakSpace   float64
}

// PopplerTextFontInfo 字体信息（对应 Poppler 的 TextFontInfo）
type PopplerTextFontInfo struct {
	font     *Font
	fontDict Dictionary
	gfxFont  interface{} // 对应 GfxFont
	ascent   float64
	descent  float64
	wMode    int // 书写模式：0=水平，1=垂直
}

// PopplerCharInfo 字符信息（对应 Poppler 的 CharInfo）
type PopplerCharInfo struct {
	unicode  rune
	charCode uint16
	charPos  int
	edge     float64
	font     *PopplerTextFontInfo
	textMat  [6]float64
}

// PopplerTextWord 单词（对应 Poppler 的 TextWord）
type PopplerTextWord struct {
	chars    []PopplerCharInfo
	xMin     float64
	xMax     float64
	yMin     float64
	yMax     float64
	base     float64
	fontSize float64
	rot      int // 旋转：0, 1, 2, 3
	wMode    int // 书写模式
	next     *PopplerTextWord
}

// PopplerTextPool 文本池（对应 Poppler 的 TextPool）
type PopplerTextPool struct {
	minBaseIdx int
	maxBaseIdx int
	pool       map[int]*PopplerTextWord
	cursor     map[int]*PopplerTextWord
	fontSize   float64
}

// NewPopplerTextOutputDev 创建新的 Poppler 风格文本输出设备
func NewPopplerTextOutputDev(doc *Document, page *Page) *PopplerTextOutputDev {
	dev := &PopplerTextOutputDev{
		doc:        doc,
		page:       page,
		pageWidth:  page.Width(),
		pageHeight: page.Height(),
		rawOrder:   false,
		
		// Poppler 的默认常量
		minDupBreakOverlap: 0.1,
		dupMaxPriDelta:     0.5,
		dupMaxSecDelta:     0.5,
		minWordBreakSpace:  0.1,
		
		mergeCombining: true,
	}
	
	// 初始化文本池
	for i := 0; i < 4; i++ {
		dev.pools[i] = &PopplerTextPool{
			minBaseIdx: math.MaxInt32,
			maxBaseIdx: math.MinInt32,
			pool:       make(map[int]*PopplerTextWord),
			cursor:     make(map[int]*PopplerTextWord),
		}
	}
	
	return dev
}

// AddChar 添加字符（对应 Poppler 的 TextPage::addChar）
func (dev *PopplerTextOutputDev) AddChar(state *TextGraphicsState, x, y, dx, dy float64, c uint16, u rune) {
	// 1. 减去字符和单词间距
	sp := state.CharSpace
	if c == 0x20 {
		sp += state.WordSpace
	}
	
	dx2, dy2 := state.TextTransformDelta(sp * state.Scale / 100, 0)
	dx -= dx2
	dy -= dy2
	
	// 2. 转换到设备坐标
	x1, y1 := state.Transform(x, y)
	w1, h1 := state.TransformDelta(dx, dy)
	
	// 3. 检查是否在页面范围内
	if x1+w1 < 0 || x1 > dev.pageWidth || y1+h1 < 0 || y1 > dev.pageHeight {
		dev.charPos++
		return
	}
	
	// 4. 检查微小字符限制
	if math.Abs(w1) < 3 && math.Abs(h1) < 3 {
		dev.nTinyChars++
		if dev.nTinyChars > 50000 {
			dev.charPos++
			return
		}
	}
	
	// 5. 在空格处断词
	if u == ' ' || u == '\t' || u == '\n' || u == '\r' {
		dev.charPos++
		dev.EndWord()
		return
	}
	
	// 6. 忽略空字符
	if u == 0 {
		dev.charPos++
		return
	}
	
	// 7. 获取字体变换矩阵
	var textMat [6]float64
	textMat[0] = state.TextMatrix[0] * state.Scale / 100
	textMat[1] = state.TextMatrix[1] * state.Scale / 100
	textMat[2] = state.TextMatrix[2]
	textMat[3] = state.TextMatrix[3]
	textMat[4] = x1
	textMat[5] = y1
	
	// 8. 检查是否需要开始新单词
	if dev.curWord != nil && len(dev.curWord.chars) > 0 {
		var base, sp, delta float64
		lastChar := &dev.curWord.chars[len(dev.curWord.chars)-1]
		
		switch dev.curWord.rot {
		case 0:
			base = y1
			sp = x1 - dev.curWord.xMax
			delta = x1 - lastChar.edge
		case 1:
			base = x1
			sp = y1 - dev.curWord.yMax
			delta = y1 - lastChar.edge
		case 2:
			base = y1
			sp = dev.curWord.xMin - x1
			delta = lastChar.edge - x1
		case 3:
			base = x1
			sp = dev.curWord.yMin - y1
			delta = lastChar.edge - y1
		}
		
		overlap := math.Abs(delta) < dev.dupMaxPriDelta*dev.curWord.fontSize &&
			math.Abs(base-dev.curWord.base) < dev.dupMaxSecDelta*dev.curWord.fontSize
		
		wMode := 0
		if dev.curFont != nil {
			wMode = dev.curFont.wMode
		}
		
		// 判断是否需要断词
		if overlap || dev.lastCharOverlap ||
			sp < -dev.minDupBreakOverlap*dev.curWord.fontSize ||
			sp > dev.minWordBreakSpace*dev.curWord.fontSize ||
			math.Abs(base-dev.curWord.base) > 0.5 ||
			dev.curFontSize != dev.curWord.fontSize ||
			wMode != dev.curWord.wMode {
			dev.EndWord()
		}
		
		dev.lastCharOverlap = overlap
	} else {
		dev.lastCharOverlap = false
	}
	
	// 9. 如果需要，开始新单词
	if dev.curWord == nil {
		dev.BeginWord(state, x1, y1)
	}
	
	// 10. 处理反向文本
	if (dev.curWord.rot == 0 && w1 < 0) ||
		(dev.curWord.rot == 1 && h1 < 0) ||
		(dev.curWord.rot == 2 && w1 > 0) ||
		(dev.curWord.rot == 3 && h1 > 0) {
		dev.EndWord()
		dev.BeginWord(state, x1, y1)
		x1 += w1
		y1 += h1
		w1 = -w1
		h1 = -h1
	}
	
	// 11. 添加字符到当前单词
	dev.curWord.AddChar(state, dev.curFont, x1, y1, w1, h1, dev.charPos, c, u, textMat)
	dev.charPos++
}

// BeginWord 开始新单词（对应 Poppler 的 TextPage::beginWord）
func (dev *PopplerTextOutputDev) BeginWord(state *TextGraphicsState, x, y float64) {
	// 确定旋转角度
	rot := state.GetRotation()
	
	dev.curWord = &PopplerTextWord{
		chars:    make([]PopplerCharInfo, 0),
		fontSize: dev.curFontSize,
		rot:      rot,
		xMin:     x,
		xMax:     x,
		yMin:     y,
		yMax:     y,
		base:     y,
	}
	
	if dev.curFont != nil {
		dev.curWord.wMode = dev.curFont.wMode
	}
}

// EndWord 结束当前单词（对应 Poppler 的 TextPage::endWord）
func (dev *PopplerTextOutputDev) EndWord() {
	if dev.nest > 0 {
		dev.nest--
		return
	}
	
	if dev.curWord != nil {
		dev.AddWord(dev.curWord)
		dev.curWord = nil
	}
}

// AddWord 添加单词到池或原始列表（对应 Poppler 的 TextPage::addWord）
func (dev *PopplerTextOutputDev) AddWord(word *PopplerTextWord) {
	// 丢弃零长度单词
	if len(word.chars) == 0 {
		return
	}
	
	if dev.rawOrder {
		if dev.rawLastWord != nil {
			dev.rawLastWord.next = word
		} else {
			dev.rawWords = word
		}
		dev.rawLastWord = word
	} else {
		dev.pools[word.rot].AddWord(word)
	}
}

// AddChar 添加字符到单词（对应 Poppler 的 TextWord::addChar）
func (w *PopplerTextWord) AddChar(state *TextGraphicsState, font *PopplerTextFontInfo, x, y, dx, dy float64, charPos int, c uint16, u rune, textMat [6]float64) {
	// 计算边缘位置
	var edge float64
	switch w.rot {
	case 0:
		edge = x + dx
	case 1:
		edge = y + dy
	case 2:
		edge = x
	case 3:
		edge = y
	}
	
	// 添加字符
	w.chars = append(w.chars, PopplerCharInfo{
		unicode:  u,
		charCode: c,
		charPos:  charPos,
		edge:     edge,
		font:     font,
		textMat:  textMat,
	})
	
	// 更新边界框
	if x < w.xMin {
		w.xMin = x
	}
	if x+dx > w.xMax {
		w.xMax = x + dx
	}
	if y < w.yMin {
		w.yMin = y
	}
	if y+dy > w.yMax {
		w.yMax = y + dy
	}
	
	// 更新基线
	switch w.rot {
	case 0:
		w.base = y
	case 1:
		w.base = x
	case 2:
		w.base = y
	case 3:
		w.base = x
	}
}

// AddWord 添加单词到池（对应 Poppler 的 TextPool::addWord）
func (pool *PopplerTextPool) AddWord(word *PopplerTextWord) {
	// 计算基线索引
	baseIdx := int(math.Floor(word.base / 2.0))
	
	// 更新索引范围
	if baseIdx < pool.minBaseIdx {
		pool.minBaseIdx = baseIdx
	}
	if baseIdx > pool.maxBaseIdx {
		pool.maxBaseIdx = baseIdx
	}
	
	// 添加到池中
	if pool.pool[baseIdx] == nil {
		pool.pool[baseIdx] = word
		pool.cursor[baseIdx] = word
	} else {
		pool.cursor[baseIdx].next = word
		pool.cursor[baseIdx] = word
	}
}

// GetPool 获取指定基线索引的单词（对应 Poppler 的 TextPool::getPool）
func (pool *PopplerTextPool) GetPool(baseIdx int) *PopplerTextWord {
	return pool.pool[baseIdx]
}

// SetPool 设置指定基线索引的单词（对应 Poppler 的 TextPool::setPool）
func (pool *PopplerTextPool) SetPool(baseIdx int, word *PopplerTextWord) {
	pool.pool[baseIdx] = word
	if word != nil {
		pool.cursor[baseIdx] = word
	}
}

// GetBaseIdx 计算基线索引（对应 Poppler 的 TextPool::getBaseIdx）
func (pool *PopplerTextPool) GetBaseIdx(base float64) int {
	return int(math.Floor(base / 2.0))
}

// BuildText 构建文本（简化版的 coalesce + getText）
func (dev *PopplerTextOutputDev) BuildText() string {
	// 确保最后的单词被添加
	dev.EndWord()
	
	// 收集所有单词
	var allWords []*PopplerTextWord
	
	if dev.rawOrder {
		// 原始顺序
		for word := dev.rawWords; word != nil; word = word.next {
			allWords = append(allWords, word)
		}
	} else {
		// 从池中收集
		for rot := 0; rot < 4; rot++ {
			pool := dev.pools[rot]
			for baseIdx := pool.minBaseIdx; baseIdx <= pool.maxBaseIdx; baseIdx++ {
				for word := pool.GetPool(baseIdx); word != nil; word = word.next {
					allWords = append(allWords, word)
				}
			}
		}
	}
	
	// 按 Y 坐标排序（从上到下）
	sort.Slice(allWords, func(i, j int) bool {
		// 使用基线排序
		if math.Abs(allWords[i].base-allWords[j].base) > 2 {
			return allWords[i].base > allWords[j].base
		}
		// 同一行内按 X 坐标排序
		return allWords[i].xMin < allWords[j].xMin
	})
	
	// 构建文本
	var result string
	var lastBase float64
	firstWord := true
	
	for _, word := range allWords {
		// 检查是否需要换行
		if !firstWord && math.Abs(word.base-lastBase) > 2 {
			result += "\n"
		}
		
		// 添加单词文本
		for _, char := range word.chars {
			result += string(char.unicode)
		}
		result += " "
		
		lastBase = word.base
		firstWord = false
	}
	
	return result
}
