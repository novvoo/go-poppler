// Package pdf provides XFA form rendering, JavaScript execution, and OC layer operations
package pdf

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// ============================================================================
// 1. 完整 XFA 表单渲染
// ============================================================================

// XFAForm 表示 XFA 表单
type XFAForm struct {
	doc           *Document
	Template      *XFATemplate
	Data          *XFAData
	Config        *XFAConfig
	Datasets      *XFADatasets
	LocaleSet     *XFALocaleSet
	Stylesheet    *XFAStylesheet
	ConnectionSet *XFAConnectionSet
	rawPackets    map[string][]byte
}

// XFATemplate XFA 模板
type XFATemplate struct {
	XMLName   xml.Name      `xml:"template"`
	Subforms  []XFASubform  `xml:"subform"`
	Scripts   []XFAScript   `xml:"script"`
	Variables []XFAVariable `xml:"variables"`
}

// XFASubform XFA 子表单
type XFASubform struct {
	Name       string         `xml:"name,attr"`
	Layout     string         `xml:"layout,attr"`
	X          string         `xml:"x,attr"`
	Y          string         `xml:"y,attr"`
	W          string         `xml:"w,attr"`
	H          string         `xml:"h,attr"`
	Fields     []XFAField     `xml:"field"`
	Draws      []XFADraw      `xml:"draw"`
	Subforms   []XFASubform   `xml:"subform"`
	Areas      []XFAArea      `xml:"area"`
	ExclGroups []XFAExclGroup `xml:"exclGroup"`
	Events     []XFAEvent     `xml:"event"`
}

// XFAField XFA 字段
type XFAField struct {
	Name      string       `xml:"name,attr"`
	X         string       `xml:"x,attr"`
	Y         string       `xml:"y,attr"`
	W         string       `xml:"w,attr"`
	H         string       `xml:"h,attr"`
	Access    string       `xml:"access,attr"`
	Presence  string       `xml:"presence,attr"`
	UI        XFAUI        `xml:"ui"`
	Caption   XFACaption   `xml:"caption"`
	Value     XFAValue     `xml:"value"`
	Items     []XFAItems   `xml:"items"`
	Validate  XFAValidate  `xml:"validate"`
	Calculate XFACalculate `xml:"calculate"`
	Events    []XFAEvent   `xml:"event"`
	Bind      XFABind      `xml:"bind"`
	Font      XFAFont      `xml:"font"`
	Para      XFAPara      `xml:"para"`
	Margin    XFAMargin    `xml:"margin"`
	Border    XFABorder    `xml:"border"`
	Assist    XFAAssist    `xml:"assist"`
}

// XFAUI XFA UI 元素
type XFAUI struct {
	TextEdit     *XFATextEdit     `xml:"textEdit"`
	NumericEdit  *XFANumericEdit  `xml:"numericEdit"`
	DateTimeEdit *XFADateTimeEdit `xml:"dateTimeEdit"`
	ChoiceList   *XFAChoiceList   `xml:"choiceList"`
	CheckButton  *XFACheckButton  `xml:"checkButton"`
	Button       *XFAButton       `xml:"button"`
	Signature    *XFASignature    `xml:"signature"`
	ImageEdit    *XFAImageEdit    `xml:"imageEdit"`
	PasswordEdit *XFAPasswordEdit `xml:"passwordEdit"`
	BarCode      *XFABarCode      `xml:"barcode"`
}

// XFATextEdit 文本编辑控件
type XFATextEdit struct {
	MultiLine     string    `xml:"multiLine,attr"`
	HScrollPolicy string    `xml:"hScrollPolicy,attr"`
	VScrollPolicy string    `xml:"vScrollPolicy,attr"`
	Border        XFABorder `xml:"border"`
	Margin        XFAMargin `xml:"margin"`
	Comb          *XFAComb  `xml:"comb"`
}

// XFANumericEdit 数字编辑控件
type XFANumericEdit struct {
	HScrollPolicy string    `xml:"hScrollPolicy,attr"`
	Border        XFABorder `xml:"border"`
}

// XFADateTimeEdit 日期时间编辑控件
type XFADateTimeEdit struct {
	Picker string    `xml:"picker,attr"`
	Border XFABorder `xml:"border"`
}

// XFAChoiceList 选择列表控件
type XFAChoiceList struct {
	Open      string    `xml:"open,attr"`
	CommitOn  string    `xml:"commitOn,attr"`
	TextEntry string    `xml:"textEntry,attr"`
	Border    XFABorder `xml:"border"`
}

// XFACheckButton 复选按钮控件
type XFACheckButton struct {
	Shape  string    `xml:"shape,attr"`
	Size   string    `xml:"size,attr"`
	Mark   string    `xml:"mark,attr"`
	Border XFABorder `xml:"border"`
}

// XFAButton 按钮控件
type XFAButton struct {
	Highlight string `xml:"highlight,attr"`
}

// XFASignature 签名控件
type XFASignature struct {
	Type   string    `xml:"type,attr"`
	Border XFABorder `xml:"border"`
}

// XFAImageEdit 图像编辑控件
type XFAImageEdit struct {
	Data   string    `xml:"data,attr"`
	Border XFABorder `xml:"border"`
}

// XFAPasswordEdit 密码编辑控件
type XFAPasswordEdit struct {
	PasswordChar  string    `xml:"passwordChar,attr"`
	HScrollPolicy string    `xml:"hScrollPolicy,attr"`
	Border        XFABorder `xml:"border"`
}

// XFABarCode 条形码控件
type XFABarCode struct {
	Type            string `xml:"type,attr"`
	DataLength      string `xml:"dataLength,attr"`
	ModuleWidth     string `xml:"moduleWidth,attr"`
	ModuleHeight    string `xml:"moduleHeight,attr"`
	PrintCheckDigit string `xml:"printCheckDigit,attr"`
}

// XFAComb 梳状控件
type XFAComb struct {
	NumberOfCells int `xml:"numberOfCells,attr"`
}

// XFACaption 标题
type XFACaption struct {
	Placement string   `xml:"placement,attr"`
	Reserve   string   `xml:"reserve,attr"`
	Value     XFAValue `xml:"value"`
	Font      XFAFont  `xml:"font"`
	Para      XFAPara  `xml:"para"`
}

// XFAValue 值
type XFAValue struct {
	Text     string     `xml:"text"`
	Integer  string     `xml:"integer"`
	Decimal  string     `xml:"decimal"`
	Float    string     `xml:"float"`
	Date     string     `xml:"date"`
	Time     string     `xml:"time"`
	DateTime string     `xml:"dateTime"`
	Boolean  string     `xml:"boolean"`
	Image    *XFAImage  `xml:"image"`
	ExData   *XFAExData `xml:"exData"`
}

// XFAImage 图像
type XFAImage struct {
	ContentType      string `xml:"contentType,attr"`
	Href             string `xml:"href,attr"`
	TransferEncoding string `xml:"transferEncoding,attr"`
	Content          string `xml:",chardata"`
}

// XFAExData 外部数据
type XFAExData struct {
	ContentType string `xml:"contentType,attr"`
	Content     string `xml:",innerxml"`
}

// XFAItems 项目列表
type XFAItems struct {
	Save     string   `xml:"save,attr"`
	Ref      string   `xml:"ref,attr"`
	Texts    []string `xml:"text"`
	Integers []string `xml:"integer"`
	Floats   []string `xml:"float"`
	Dates    []string `xml:"date"`
}

// XFAValidate 验证
type XFAValidate struct {
	FormatTest string     `xml:"formatTest,attr"`
	NullTest   string     `xml:"nullTest,attr"`
	ScriptTest string     `xml:"scriptTest,attr"`
	Message    XFAMessage `xml:"message"`
	Script     XFAScript  `xml:"script"`
	Picture    string     `xml:"picture"`
}

// XFAMessage 消息
type XFAMessage struct {
	Text string `xml:"text"`
}

// XFACalculate 计算
type XFACalculate struct {
	Override string    `xml:"override,attr"`
	Script   XFAScript `xml:"script"`
}

// XFAEvent 事件
type XFAEvent struct {
	Activity string     `xml:"activity,attr"`
	Name     string     `xml:"name,attr"`
	Ref      string     `xml:"ref,attr"`
	Script   XFAScript  `xml:"script"`
	Submit   *XFASubmit `xml:"submit"`
}

// XFASubmit 提交
type XFASubmit struct {
	Format       string `xml:"format,attr"`
	Target       string `xml:"target,attr"`
	TextEncoding string `xml:"textEncoding,attr"`
	XDPContent   string `xml:"xdpContent,attr"`
}

// XFAScript 脚本
type XFAScript struct {
	ContentType string `xml:"contentType,attr"`
	RunAt       string `xml:"runAt,attr"`
	Content     string `xml:",chardata"`
}

// XFABind 绑定
type XFABind struct {
	Match   string `xml:"match,attr"`
	Ref     string `xml:"ref,attr"`
	Picture string `xml:"picture"`
}

// XFAFont 字体
type XFAFont struct {
	Typeface    string  `xml:"typeface,attr"`
	Size        string  `xml:"size,attr"`
	Weight      string  `xml:"weight,attr"`
	Posture     string  `xml:"posture,attr"`
	Underline   string  `xml:"underline,attr"`
	LineThrough string  `xml:"lineThrough,attr"`
	Fill        XFAFill `xml:"fill"`
}

// XFAFill 填充
type XFAFill struct {
	Color XFAColor `xml:"color"`
}

// XFAColor 颜色
type XFAColor struct {
	Value string `xml:"value,attr"`
}

// XFAPara 段落
type XFAPara struct {
	HAlign      string `xml:"hAlign,attr"`
	VAlign      string `xml:"vAlign,attr"`
	SpaceBefore string `xml:"spaceBefore,attr"`
	SpaceAfter  string `xml:"spaceAfter,attr"`
	TextIndent  string `xml:"textIndent,attr"`
	MarginLeft  string `xml:"marginLeft,attr"`
	MarginRight string `xml:"marginRight,attr"`
}

// XFAMargin 边距
type XFAMargin struct {
	TopInset    string `xml:"topInset,attr"`
	BottomInset string `xml:"bottomInset,attr"`
	LeftInset   string `xml:"leftInset,attr"`
	RightInset  string `xml:"rightInset,attr"`
}

// XFABorder 边框
type XFABorder struct {
	Presence string      `xml:"presence,attr"`
	Hand     string      `xml:"hand,attr"`
	Edges    []XFAEdge   `xml:"edge"`
	Corner   []XFACorner `xml:"corner"`
	Fill     XFAFill     `xml:"fill"`
}

// XFAEdge 边
type XFAEdge struct {
	Presence  string   `xml:"presence,attr"`
	Stroke    string   `xml:"stroke,attr"`
	Thickness string   `xml:"thickness,attr"`
	Color     XFAColor `xml:"color"`
}

// XFACorner 角
type XFACorner struct {
	Presence  string   `xml:"presence,attr"`
	Stroke    string   `xml:"stroke,attr"`
	Thickness string   `xml:"thickness,attr"`
	Radius    string   `xml:"radius,attr"`
	Join      string   `xml:"join,attr"`
	Color     XFAColor `xml:"color"`
}

// XFAAssist 辅助
type XFAAssist struct {
	Role    string   `xml:"role,attr"`
	ToolTip string   `xml:"toolTip"`
	Speak   XFASpeak `xml:"speak"`
}

// XFASpeak 语音
type XFASpeak struct {
	Priority string `xml:"priority,attr"`
	Content  string `xml:",chardata"`
}

// XFADraw 绘制元素
type XFADraw struct {
	Name     string    `xml:"name,attr"`
	X        string    `xml:"x,attr"`
	Y        string    `xml:"y,attr"`
	W        string    `xml:"w,attr"`
	H        string    `xml:"h,attr"`
	Presence string    `xml:"presence,attr"`
	Value    XFAValue  `xml:"value"`
	Font     XFAFont   `xml:"font"`
	Para     XFAPara   `xml:"para"`
	Border   XFABorder `xml:"border"`
	Margin   XFAMargin `xml:"margin"`
	UI       XFAUI     `xml:"ui"`
}

// XFAArea 区域
type XFAArea struct {
	Name     string       `xml:"name,attr"`
	X        string       `xml:"x,attr"`
	Y        string       `xml:"y,attr"`
	Fields   []XFAField   `xml:"field"`
	Draws    []XFADraw    `xml:"draw"`
	Subforms []XFASubform `xml:"subform"`
}

// XFAExclGroup 互斥组
type XFAExclGroup struct {
	Name   string     `xml:"name,attr"`
	Layout string     `xml:"layout,attr"`
	Fields []XFAField `xml:"field"`
	Border XFABorder  `xml:"border"`
}

// XFAVariable 变量
type XFAVariable struct {
	Name   string    `xml:"name,attr"`
	Script XFAScript `xml:"script"`
	Text   string    `xml:"text"`
}

// XFAData XFA 数据
type XFAData struct {
	XMLName xml.Name `xml:"data"`
	Content []byte   `xml:",innerxml"`
}

// XFAConfig XFA 配置
type XFAConfig struct {
	XMLName xml.Name   `xml:"config"`
	Present XFAPresent `xml:"present"`
	Acrobat XFAAcrobat `xml:"acrobat"`
}

// XFAPresent 呈现配置
type XFAPresent struct {
	PDF    XFAPDF       `xml:"pdf"`
	Common XFACommon    `xml:"common"`
	Script XFAScriptCfg `xml:"script"`
}

// XFAPDF PDF 配置
type XFAPDF struct {
	Version             string `xml:"version"`
	AdobeExtensionLevel string `xml:"adobeExtensionLevel"`
	RenderPolicy        string `xml:"renderPolicy"`
	ScriptModel         string `xml:"scriptModel"`
	Interactive         string `xml:"interactive"`
}

// XFACommon 通用配置
type XFACommon struct {
	Data     XFADataCfg     `xml:"data"`
	Locale   string         `xml:"locale"`
	Template XFATemplateCfg `xml:"template"`
}

// XFADataCfg 数据配置
type XFADataCfg struct {
	OutputXSL  XFAOutputXSL `xml:"outputXSL"`
	AdjustData string       `xml:"adjustData"`
}

// XFAOutputXSL 输出 XSL
type XFAOutputXSL struct {
	URI string `xml:"uri,attr"`
}

// XFATemplateCfg 模板配置
type XFATemplateCfg struct {
	Base string `xml:"base,attr"`
}

// XFAScriptCfg 脚本配置
type XFAScriptCfg struct {
	CurrentPage string `xml:"currentPage"`
	Exclude     string `xml:"exclude"`
	RunScripts  string `xml:"runScripts"`
}

// XFAAcrobat Acrobat 配置
type XFAAcrobat struct {
	Acrobat7 XFAAcrobat7 `xml:"acrobat7"`
}

// XFAAcrobat7 Acrobat 7 配置
type XFAAcrobat7 struct {
	DynamicRender string `xml:"dynamicRender"`
}

// XFADatasets XFA 数据集
type XFADatasets struct {
	XMLName xml.Name `xml:"datasets"`
	Data    XFAData  `xml:"data"`
}

// XFALocaleSet XFA 区域设置
type XFALocaleSet struct {
	XMLName xml.Name    `xml:"localeSet"`
	Locales []XFALocale `xml:"locale"`
}

// XFALocale 区域
type XFALocale struct {
	Name            string             `xml:"name,attr"`
	Desc            string             `xml:"desc,attr"`
	CalendarSymbols XFACalendarSymbols `xml:"calendarSymbols"`
	DatePatterns    XFADatePatterns    `xml:"datePatterns"`
	TimePatterns    XFATimePatterns    `xml:"timePatterns"`
	NumberPatterns  XFANumberPatterns  `xml:"numberPatterns"`
	NumberSymbols   XFANumberSymbols   `xml:"numberSymbols"`
	CurrencySymbols XFACurrencySymbols `xml:"currencySymbols"`
}

// XFACalendarSymbols 日历符号
type XFACalendarSymbols struct {
	DayNames   []XFADayNames   `xml:"dayNames"`
	MonthNames []XFAMonthNames `xml:"monthNames"`
	EraNames   XFAEraNames     `xml:"eraNames"`
}

// XFADayNames 日名称
type XFADayNames struct {
	Abbr string   `xml:"abbr,attr"`
	Days []string `xml:"day"`
}

// XFAMonthNames 月名称
type XFAMonthNames struct {
	Abbr   string   `xml:"abbr,attr"`
	Months []string `xml:"month"`
}

// XFAEraNames 纪元名称
type XFAEraNames struct {
	Eras []string `xml:"era"`
}

// XFADatePatterns 日期模式
type XFADatePatterns struct {
	Patterns []XFAPattern `xml:"datePattern"`
}

// XFATimePatterns 时间模式
type XFATimePatterns struct {
	Patterns []XFAPattern `xml:"timePattern"`
}

// XFANumberPatterns 数字模式
type XFANumberPatterns struct {
	Patterns []XFAPattern `xml:"numberPattern"`
}

// XFAPattern 模式
type XFAPattern struct {
	Name    string `xml:"name,attr"`
	Pattern string `xml:",chardata"`
}

// XFANumberSymbols 数字符号
type XFANumberSymbols struct {
	Decimal  string `xml:"decimal"`
	Grouping string `xml:"grouping"`
	Percent  string `xml:"percent"`
	Minus    string `xml:"minus"`
	Zero     string `xml:"zero"`
}

// XFACurrencySymbols 货币符号
type XFACurrencySymbols struct {
	Symbol  string `xml:"symbol"`
	ISOName string `xml:"isoname"`
	Decimal string `xml:"decimal"`
}

// XFAStylesheet XFA 样式表
type XFAStylesheet struct {
	XMLName xml.Name `xml:"stylesheet"`
	Content []byte   `xml:",innerxml"`
}

// XFAConnectionSet XFA 连接集
type XFAConnectionSet struct {
	XMLName     xml.Name        `xml:"connectionSet"`
	Connections []XFAConnection `xml:"wsdlConnection"`
}

// XFAConnection XFA 连接
type XFAConnection struct {
	Name            string `xml:"name,attr"`
	DataDescription string `xml:"dataDescription,attr"`
	WSDLAddress     string `xml:"wsdlAddress"`
	SoapAddress     string `xml:"soapAddress"`
	SoapAction      string `xml:"soapAction"`
}

// NewXFAForm 创建 XFA 表单解析器
func NewXFAForm(doc *Document) (*XFAForm, error) {
	xfa := &XFAForm{
		doc:        doc,
		rawPackets: make(map[string][]byte),
	}

	if err := xfa.parse(); err != nil {
		return nil, err
	}

	return xfa, nil
}

func (xfa *XFAForm) parse() error {
	acroFormRef := xfa.doc.Root.Get("AcroForm")
	if acroFormRef == nil {
		return fmt.Errorf("no AcroForm found")
	}

	acroFormObj, err := xfa.doc.ResolveObject(acroFormRef)
	if err != nil {
		return err
	}

	acroForm, ok := acroFormObj.(Dictionary)
	if !ok {
		return fmt.Errorf("invalid AcroForm")
	}

	xfaRef := acroForm.Get("XFA")
	if xfaRef == nil {
		return fmt.Errorf("no XFA data found")
	}

	xfaObj, err := xfa.doc.ResolveObject(xfaRef)
	if err != nil {
		return err
	}

	switch v := xfaObj.(type) {
	case Stream:
		return xfa.parseXFAStream(v)
	case Array:
		return xfa.parseXFAArray(v)
	default:
		return fmt.Errorf("invalid XFA data type")
	}
}

func (xfa *XFAForm) parseXFAStream(stream Stream) error {
	data, err := stream.Decode()
	if err != nil {
		return err
	}
	return xfa.parseXFAPackets(data)
}

func (xfa *XFAForm) parseXFAArray(arr Array) error {
	var fullData bytes.Buffer

	for i := 0; i < len(arr); i += 2 {
		var name string
		if i < len(arr) {
			if n, ok := arr[i].(String); ok {
				name = string(n.Value)
			}
		}

		if i+1 < len(arr) {
			streamObj, err := xfa.doc.ResolveObject(arr[i+1])
			if err != nil {
				continue
			}

			if stream, ok := streamObj.(Stream); ok {
				data, err := stream.Decode()
				if err != nil {
					continue
				}
				xfa.rawPackets[name] = data
				fullData.Write(data)
			}
		}
	}

	return xfa.parseXFAPackets(fullData.Bytes())
}

func (xfa *XFAForm) parseXFAPackets(data []byte) error {
	// 解析模板
	if templateData := xfa.extractPacket(data, "template"); templateData != nil {
		xfa.Template = &XFATemplate{}
		if err := xml.Unmarshal(templateData, xfa.Template); err != nil {
			// 忽略解析错误，继续处理其他包
		}
	}

	// 解析数据
	if dataData := xfa.extractPacket(data, "data"); dataData != nil {
		xfa.Data = &XFAData{Content: dataData}
	}

	// 解析配置
	if configData := xfa.extractPacket(data, "config"); configData != nil {
		xfa.Config = &XFAConfig{}
		xml.Unmarshal(configData, xfa.Config)
	}

	// 解析数据集
	if datasetsData := xfa.extractPacket(data, "datasets"); datasetsData != nil {
		xfa.Datasets = &XFADatasets{}
		xml.Unmarshal(datasetsData, xfa.Datasets)
	}

	// 解析区域设置
	if localeData := xfa.extractPacket(data, "localeSet"); localeData != nil {
		xfa.LocaleSet = &XFALocaleSet{}
		xml.Unmarshal(localeData, xfa.LocaleSet)
	}

	// 解析样式表
	if stylesheetData := xfa.extractPacket(data, "stylesheet"); stylesheetData != nil {
		xfa.Stylesheet = &XFAStylesheet{Content: stylesheetData}
	}

	// 解析连接集
	if connectionData := xfa.extractPacket(data, "connectionSet"); connectionData != nil {
		xfa.ConnectionSet = &XFAConnectionSet{}
		xml.Unmarshal(connectionData, xfa.ConnectionSet)
	}

	return nil
}

func (xfa *XFAForm) extractPacket(data []byte, name string) []byte {
	// 查找 XFA 包
	startTag := fmt.Sprintf("<%s", name)
	endTag := fmt.Sprintf("</%s>", name)

	startIdx := bytes.Index(data, []byte(startTag))
	if startIdx == -1 {
		return nil
	}

	endIdx := bytes.Index(data[startIdx:], []byte(endTag))
	if endIdx == -1 {
		return nil
	}

	return data[startIdx : startIdx+endIdx+len(endTag)]
}

// GetFields 获取所有 XFA 字段
func (xfa *XFAForm) GetFields() []XFAField {
	var fields []XFAField
	if xfa.Template == nil {
		return fields
	}

	for _, subform := range xfa.Template.Subforms {
		fields = append(fields, xfa.collectFields(subform)...)
	}

	return fields
}

func (xfa *XFAForm) collectFields(subform XFASubform) []XFAField {
	var fields []XFAField
	fields = append(fields, subform.Fields...)

	for _, area := range subform.Areas {
		fields = append(fields, area.Fields...)
	}

	for _, exclGroup := range subform.ExclGroups {
		fields = append(fields, exclGroup.Fields...)
	}

	for _, child := range subform.Subforms {
		fields = append(fields, xfa.collectFields(child)...)
	}

	return fields
}

// GetFieldValue 获取字段值
func (xfa *XFAForm) GetFieldValue(fieldName string) string {
	if xfa.Data == nil {
		return ""
	}

	// 在数据中查找字段值
	pattern := fmt.Sprintf("<%s>([^<]*)</%s>", fieldName, fieldName)
	re := regexp.MustCompile(pattern)
	matches := re.FindSubmatch(xfa.Data.Content)
	if len(matches) > 1 {
		return string(matches[1])
	}

	return ""
}

// SetFieldValue 设置字段值
func (xfa *XFAForm) SetFieldValue(fieldName, value string) error {
	if xfa.Data == nil {
		xfa.Data = &XFAData{}
	}

	// 更新数据中的字段值
	pattern := fmt.Sprintf("<%s>[^<]*</%s>", fieldName, fieldName)
	re := regexp.MustCompile(pattern)
	replacement := fmt.Sprintf("<%s>%s</%s>", fieldName, value, fieldName)

	if re.Match(xfa.Data.Content) {
		xfa.Data.Content = re.ReplaceAll(xfa.Data.Content, []byte(replacement))
	} else {
		// 添加新字段
		xfa.Data.Content = append(xfa.Data.Content, []byte(replacement)...)
	}

	return nil
}

// RenderToHTML 将 XFA 表单渲染为 HTML
func (xfa *XFAForm) RenderToHTML() (string, error) {
	var buf bytes.Buffer

	buf.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	buf.WriteString("<meta charset=\"UTF-8\">\n")
	buf.WriteString("<style>\n")
	buf.WriteString(xfa.generateCSS())
	buf.WriteString("</style>\n")
	buf.WriteString("</head>\n<body>\n")

	if xfa.Template != nil {
		for _, subform := range xfa.Template.Subforms {
			buf.WriteString(xfa.renderSubform(subform))
		}
	}

	buf.WriteString("</body>\n</html>")

	return buf.String(), nil
}

func (xfa *XFAForm) generateCSS() string {
	return `
.xfa-subform { position: relative; margin: 10px; padding: 10px; border: 1px solid #ccc; }
.xfa-field { margin: 5px 0; }
.xfa-field label { display: inline-block; min-width: 150px; }
.xfa-field input, .xfa-field select, .xfa-field textarea { padding: 5px; border: 1px solid #999; }
.xfa-draw { margin: 5px 0; }
.xfa-button { padding: 8px 16px; background: #007bff; color: white; border: none; cursor: pointer; }
.xfa-button:hover { background: #0056b3; }
`
}

func (xfa *XFAForm) renderSubform(subform XFASubform) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("<div class=\"xfa-subform\" id=\"%s\">\n", subform.Name))

	// 渲染绘制元素
	for _, draw := range subform.Draws {
		buf.WriteString(xfa.renderDraw(draw))
	}

	// 渲染字段
	for _, field := range subform.Fields {
		buf.WriteString(xfa.renderField(field))
	}

	// 渲染区域
	for _, area := range subform.Areas {
		buf.WriteString(fmt.Sprintf("<div class=\"xfa-area\" id=\"%s\">\n", area.Name))
		for _, field := range area.Fields {
			buf.WriteString(xfa.renderField(field))
		}
		buf.WriteString("</div>\n")
	}

	// 渲染互斥组
	for _, exclGroup := range subform.ExclGroups {
		buf.WriteString(fmt.Sprintf("<div class=\"xfa-exclgroup\" id=\"%s\">\n", exclGroup.Name))
		for _, field := range exclGroup.Fields {
			buf.WriteString(xfa.renderField(field))
		}
		buf.WriteString("</div>\n")
	}

	// 渲染子表单
	for _, child := range subform.Subforms {
		buf.WriteString(xfa.renderSubform(child))
	}

	buf.WriteString("</div>\n")

	return buf.String()
}

func (xfa *XFAForm) renderField(field XFAField) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("<div class=\"xfa-field\" id=\"%s\">\n", field.Name))

	// 渲染标题
	if field.Caption.Value.Text != "" {
		buf.WriteString(fmt.Sprintf("<label>%s</label>\n", field.Caption.Value.Text))
	}

	// 根据 UI 类型渲染控件
	if field.UI.TextEdit != nil {
		if field.UI.TextEdit.MultiLine == "1" {
			buf.WriteString(fmt.Sprintf("<textarea name=\"%s\">%s</textarea>\n",
				field.Name, xfa.GetFieldValue(field.Name)))
		} else {
			buf.WriteString(fmt.Sprintf("<input type=\"text\" name=\"%s\" value=\"%s\">\n",
				field.Name, xfa.GetFieldValue(field.Name)))
		}
	} else if field.UI.NumericEdit != nil {
		buf.WriteString(fmt.Sprintf("<input type=\"number\" name=\"%s\" value=\"%s\">\n",
			field.Name, xfa.GetFieldValue(field.Name)))
	} else if field.UI.DateTimeEdit != nil {
		buf.WriteString(fmt.Sprintf("<input type=\"datetime-local\" name=\"%s\" value=\"%s\">\n",
			field.Name, xfa.GetFieldValue(field.Name)))
	} else if field.UI.ChoiceList != nil {
		buf.WriteString(fmt.Sprintf("<select name=\"%s\">\n", field.Name))
		for _, items := range field.Items {
			for _, text := range items.Texts {
				selected := ""
				if text == xfa.GetFieldValue(field.Name) {
					selected = " selected"
				}
				buf.WriteString(fmt.Sprintf("<option value=\"%s\"%s>%s</option>\n", text, selected, text))
			}
		}
		buf.WriteString("</select>\n")
	} else if field.UI.CheckButton != nil {
		checked := ""
		if xfa.GetFieldValue(field.Name) == "1" {
			checked = " checked"
		}
		buf.WriteString(fmt.Sprintf("<input type=\"checkbox\" name=\"%s\"%s>\n", field.Name, checked))
	} else if field.UI.Button != nil {
		buf.WriteString(fmt.Sprintf("<button class=\"xfa-button\" name=\"%s\">%s</button>\n",
			field.Name, field.Caption.Value.Text))
	} else if field.UI.PasswordEdit != nil {
		buf.WriteString(fmt.Sprintf("<input type=\"password\" name=\"%s\">\n", field.Name))
	} else {
		// 默认文本输入
		buf.WriteString(fmt.Sprintf("<input type=\"text\" name=\"%s\" value=\"%s\">\n",
			field.Name, xfa.GetFieldValue(field.Name)))
	}

	buf.WriteString("</div>\n")

	return buf.String()
}

func (xfa *XFAForm) renderDraw(draw XFADraw) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("<div class=\"xfa-draw\" id=\"%s\">\n", draw.Name))

	if draw.Value.Text != "" {
		buf.WriteString(fmt.Sprintf("<span>%s</span>\n", draw.Value.Text))
	}

	if draw.Value.Image != nil && draw.Value.Image.Content != "" {
		buf.WriteString(fmt.Sprintf("<img src=\"data:%s;base64,%s\">\n",
			draw.Value.Image.ContentType, draw.Value.Image.Content))
	}

	buf.WriteString("</div>\n")

	return buf.String()
}

// ============================================================================
// 2. JavaScript 执行引擎
// ============================================================================

// JSEngine JavaScript 执行引擎
type JSEngine struct {
	doc       *Document
	variables map[string]interface{}
	functions map[string]JSFunction
	events    map[string][]JSEventHandler
	console   *JSConsole
	mu        sync.RWMutex
}

// JSFunction JavaScript 函数
type JSFunction func(args []interface{}) interface{}

// JSEventHandler JavaScript 事件处理器
type JSEventHandler struct {
	Name     string
	Script   string
	Priority int
}

// JSConsole JavaScript 控制台
type JSConsole struct {
	logs    []JSLogEntry
	maxLogs int
	mu      sync.Mutex
}

// JSLogEntry 日志条目
type JSLogEntry struct {
	Level   string
	Message string
	Time    int64
}

// NewJSEngine 创建 JavaScript 引擎
func NewJSEngine(doc *Document) *JSEngine {
	engine := &JSEngine{
		doc:       doc,
		variables: make(map[string]interface{}),
		functions: make(map[string]JSFunction),
		events:    make(map[string][]JSEventHandler),
		console:   NewJSConsole(1000),
	}

	engine.registerBuiltinFunctions()
	return engine
}

// NewJSConsole 创建控制台
func NewJSConsole(maxLogs int) *JSConsole {
	return &JSConsole{
		logs:    make([]JSLogEntry, 0),
		maxLogs: maxLogs,
	}
}

func (c *JSConsole) log(level, message string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := JSLogEntry{
		Level:   level,
		Message: message,
		Time:    getCurrentTimestamp(),
	}

	c.logs = append(c.logs, entry)
	if len(c.logs) > c.maxLogs {
		c.logs = c.logs[1:]
	}
}

// Log 记录日志
func (c *JSConsole) Log(message string) {
	c.log("log", message)
}

// Warn 记录警告
func (c *JSConsole) Warn(message string) {
	c.log("warn", message)
}

// Error 记录错误
func (c *JSConsole) Error(message string) {
	c.log("error", message)
}

// GetLogs 获取日志
func (c *JSConsole) GetLogs() []JSLogEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]JSLogEntry{}, c.logs...)
}

// Clear 清除日志
func (c *JSConsole) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logs = c.logs[:0]
}

func (e *JSEngine) registerBuiltinFunctions() {
	// 文档操作函数
	e.functions["getField"] = e.jsGetField
	e.functions["getPageNum"] = e.jsGetPageNum
	e.functions["getPageCount"] = e.jsGetPageCount
	e.functions["getDocumentTitle"] = e.jsGetDocumentTitle
	e.functions["getDocumentAuthor"] = e.jsGetDocumentAuthor

	// 字段操作函数
	e.functions["setFieldValue"] = e.jsSetFieldValue
	e.functions["getFieldValue"] = e.jsGetFieldValue
	e.functions["setFieldHidden"] = e.jsSetFieldHidden
	e.functions["setFieldReadOnly"] = e.jsSetFieldReadOnly

	// 数学函数
	e.functions["abs"] = e.jsAbs
	e.functions["ceil"] = e.jsCeil
	e.functions["floor"] = e.jsFloor
	e.functions["round"] = e.jsRound
	e.functions["max"] = e.jsMax
	e.functions["min"] = e.jsMin
	e.functions["pow"] = e.jsPow
	e.functions["sqrt"] = e.jsSqrt

	// 字符串函数
	e.functions["strlen"] = e.jsStrlen
	e.functions["substr"] = e.jsSubstr
	e.functions["indexOf"] = e.jsIndexOf
	e.functions["toUpperCase"] = e.jsToUpperCase
	e.functions["toLowerCase"] = e.jsToLowerCase
	e.functions["trim"] = e.jsTrim
	e.functions["replace"] = e.jsReplace

	// 日期函数
	e.functions["now"] = e.jsNow
	e.functions["formatDate"] = e.jsFormatDate
	e.functions["parseDate"] = e.jsParseDate

	// 验证函数
	e.functions["isNumber"] = e.jsIsNumber
	e.functions["isEmail"] = e.jsIsEmail
	e.functions["isPhone"] = e.jsIsPhone

	// 对话框函数
	e.functions["alert"] = e.jsAlert
	e.functions["confirm"] = e.jsConfirm
	e.functions["prompt"] = e.jsPrompt

	// 打印函数
	e.functions["print"] = e.jsPrint
}

// 文档操作函数实现
func (e *JSEngine) jsGetField(args []interface{}) interface{} {
	if len(args) < 1 {
		return nil
	}
	fieldName, ok := args[0].(string)
	if !ok {
		return nil
	}

	fields := e.doc.GetFormFields()
	for _, field := range fields {
		if field.Name == fieldName {
			return map[string]interface{}{
				"name":     field.Name,
				"type":     field.Type,
				"value":    field.Value,
				"readOnly": field.ReadOnly,
				"required": field.Required,
			}
		}
	}
	return nil
}

func (e *JSEngine) jsGetPageNum(args []interface{}) interface{} {
	return 1 // 当前页码
}

func (e *JSEngine) jsGetPageCount(args []interface{}) interface{} {
	return len(e.doc.Pages)
}

func (e *JSEngine) jsGetDocumentTitle(args []interface{}) interface{} {
	if title := e.doc.Info.Get("Title"); title != nil {
		if s, ok := title.(String); ok {
			return s.Text()
		}
	}
	return ""
}

func (e *JSEngine) jsGetDocumentAuthor(args []interface{}) interface{} {
	if author := e.doc.Info.Get("Author"); author != nil {
		if s, ok := author.(String); ok {
			return s.Text()
		}
	}
	return ""
}

// 字段操作函数实现
func (e *JSEngine) jsSetFieldValue(args []interface{}) interface{} {
	if len(args) < 2 {
		return false
	}
	fieldName, ok := args[0].(string)
	if !ok {
		return false
	}
	value := fmt.Sprintf("%v", args[1])

	e.mu.Lock()
	e.variables["field_"+fieldName] = value
	e.mu.Unlock()

	return true
}

func (e *JSEngine) jsGetFieldValue(args []interface{}) interface{} {
	if len(args) < 1 {
		return ""
	}
	fieldName, ok := args[0].(string)
	if !ok {
		return ""
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	if val, ok := e.variables["field_"+fieldName]; ok {
		return val
	}

	fields := e.doc.GetFormFields()
	for _, field := range fields {
		if field.Name == fieldName {
			return field.Value
		}
	}
	return ""
}

func (e *JSEngine) jsSetFieldHidden(args []interface{}) interface{} {
	if len(args) < 2 {
		return false
	}
	fieldName, ok := args[0].(string)
	if !ok {
		return false
	}
	hidden, ok := args[1].(bool)
	if !ok {
		return false
	}

	e.mu.Lock()
	e.variables["hidden_"+fieldName] = hidden
	e.mu.Unlock()

	return true
}

func (e *JSEngine) jsSetFieldReadOnly(args []interface{}) interface{} {
	if len(args) < 2 {
		return false
	}
	fieldName, ok := args[0].(string)
	if !ok {
		return false
	}
	readOnly, ok := args[1].(bool)
	if !ok {
		return false
	}

	e.mu.Lock()
	e.variables["readonly_"+fieldName] = readOnly
	e.mu.Unlock()

	return true
}

// 数学函数实现
func (e *JSEngine) jsAbs(args []interface{}) interface{} {
	if len(args) < 1 {
		return 0.0
	}
	n := toFloat64(args[0])
	if n < 0 {
		return -n
	}
	return n
}

func (e *JSEngine) jsCeil(args []interface{}) interface{} {
	if len(args) < 1 {
		return 0.0
	}
	n := toFloat64(args[0])
	return float64(int(n) + 1)
}

func (e *JSEngine) jsFloor(args []interface{}) interface{} {
	if len(args) < 1 {
		return 0.0
	}
	n := toFloat64(args[0])
	return float64(int(n))
}

func (e *JSEngine) jsRound(args []interface{}) interface{} {
	if len(args) < 1 {
		return 0.0
	}
	n := toFloat64(args[0])
	return float64(int(n + 0.5))
}

func (e *JSEngine) jsMax(args []interface{}) interface{} {
	if len(args) < 2 {
		return 0.0
	}
	a := toFloat64(args[0])
	b := toFloat64(args[1])
	if a > b {
		return a
	}
	return b
}

func (e *JSEngine) jsMin(args []interface{}) interface{} {
	if len(args) < 2 {
		return 0.0
	}
	a := toFloat64(args[0])
	b := toFloat64(args[1])
	if a < b {
		return a
	}
	return b
}

func (e *JSEngine) jsPow(args []interface{}) interface{} {
	if len(args) < 2 {
		return 0.0
	}
	base := toFloat64(args[0])
	exp := toFloat64(args[1])
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

func (e *JSEngine) jsSqrt(args []interface{}) interface{} {
	if len(args) < 1 {
		return 0.0
	}
	n := toFloat64(args[0])
	if n < 0 {
		return 0.0
	}
	// 牛顿迭代法
	x := n
	for i := 0; i < 100; i++ {
		x = (x + n/x) / 2
	}
	return x
}

// 字符串函数实现
func (e *JSEngine) jsStrlen(args []interface{}) interface{} {
	if len(args) < 1 {
		return 0
	}
	s, ok := args[0].(string)
	if !ok {
		return 0
	}
	return len(s)
}

func (e *JSEngine) jsSubstr(args []interface{}) interface{} {
	if len(args) < 2 {
		return ""
	}
	s, ok := args[0].(string)
	if !ok {
		return ""
	}
	start := int(toFloat64(args[1]))
	if start < 0 || start >= len(s) {
		return ""
	}

	if len(args) >= 3 {
		length := int(toFloat64(args[2]))
		end := start + length
		if end > len(s) {
			end = len(s)
		}
		return s[start:end]
	}
	return s[start:]
}

func (e *JSEngine) jsIndexOf(args []interface{}) interface{} {
	if len(args) < 2 {
		return -1
	}
	s, ok := args[0].(string)
	if !ok {
		return -1
	}
	substr, ok := args[1].(string)
	if !ok {
		return -1
	}
	return strings.Index(s, substr)
}

func (e *JSEngine) jsToUpperCase(args []interface{}) interface{} {
	if len(args) < 1 {
		return ""
	}
	s, ok := args[0].(string)
	if !ok {
		return ""
	}
	return strings.ToUpper(s)
}

func (e *JSEngine) jsToLowerCase(args []interface{}) interface{} {
	if len(args) < 1 {
		return ""
	}
	s, ok := args[0].(string)
	if !ok {
		return ""
	}
	return strings.ToLower(s)
}

func (e *JSEngine) jsTrim(args []interface{}) interface{} {
	if len(args) < 1 {
		return ""
	}
	s, ok := args[0].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func (e *JSEngine) jsReplace(args []interface{}) interface{} {
	if len(args) < 3 {
		return ""
	}
	s, ok := args[0].(string)
	if !ok {
		return ""
	}
	old, ok := args[1].(string)
	if !ok {
		return s
	}
	new, ok := args[2].(string)
	if !ok {
		return s
	}
	return strings.Replace(s, old, new, -1)
}

// 日期函数实现
func (e *JSEngine) jsNow(args []interface{}) interface{} {
	return getCurrentTimestamp()
}

func (e *JSEngine) jsFormatDate(args []interface{}) interface{} {
	if len(args) < 2 {
		return ""
	}
	timestamp := int64(toFloat64(args[0]))
	format, ok := args[1].(string)
	if !ok {
		return ""
	}

	// 简单的日期格式化
	_ = timestamp
	_ = format
	return fmt.Sprintf("%d", timestamp)
}

func (e *JSEngine) jsParseDate(args []interface{}) interface{} {
	if len(args) < 1 {
		return 0
	}
	dateStr, ok := args[0].(string)
	if !ok {
		return 0
	}
	_ = dateStr
	return getCurrentTimestamp()
}

// 验证函数实现
func (e *JSEngine) jsIsNumber(args []interface{}) interface{} {
	if len(args) < 1 {
		return false
	}
	s, ok := args[0].(string)
	if !ok {
		return false
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func (e *JSEngine) jsIsEmail(args []interface{}) interface{} {
	if len(args) < 1 {
		return false
	}
	s, ok := args[0].(string)
	if !ok {
		return false
	}
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(s)
}

func (e *JSEngine) jsIsPhone(args []interface{}) interface{} {
	if len(args) < 1 {
		return false
	}
	s, ok := args[0].(string)
	if !ok {
		return false
	}
	phoneRegex := regexp.MustCompile(`^[\d\s\-\+\(\)]{7,20}$`)
	return phoneRegex.MatchString(s)
}

// 对话框函数实现
func (e *JSEngine) jsAlert(args []interface{}) interface{} {
	if len(args) < 1 {
		return nil
	}
	message := fmt.Sprintf("%v", args[0])
	e.console.Log("ALERT: " + message)
	return nil
}

func (e *JSEngine) jsConfirm(args []interface{}) interface{} {
	if len(args) < 1 {
		return false
	}
	message := fmt.Sprintf("%v", args[0])
	e.console.Log("CONFIRM: " + message)
	return true // 默认返回 true
}

func (e *JSEngine) jsPrompt(args []interface{}) interface{} {
	if len(args) < 1 {
		return ""
	}
	message := fmt.Sprintf("%v", args[0])
	e.console.Log("PROMPT: " + message)
	if len(args) >= 2 {
		return args[1] // 返回默认值
	}
	return ""
}

func (e *JSEngine) jsPrint(args []interface{}) interface{} {
	e.console.Log("PRINT: Document print requested")
	return nil
}

// Execute 执行 JavaScript 代码
func (e *JSEngine) Execute(script string) (interface{}, error) {
	// 简单的脚本解析和执行
	script = strings.TrimSpace(script)

	// 处理变量赋值
	if strings.Contains(script, "=") && !strings.Contains(script, "==") {
		parts := strings.SplitN(script, "=", 2)
		if len(parts) == 2 {
			varName := strings.TrimSpace(parts[0])
			varName = strings.TrimPrefix(varName, "var ")
			varName = strings.TrimPrefix(varName, "let ")
			varName = strings.TrimPrefix(varName, "const ")
			varName = strings.TrimSpace(varName)

			value := strings.TrimSpace(parts[1])
			value = strings.TrimSuffix(value, ";")

			result, err := e.evaluateExpression(value)
			if err != nil {
				return nil, err
			}

			e.mu.Lock()
			e.variables[varName] = result
			e.mu.Unlock()

			return result, nil
		}
	}

	// 处理函数调用
	if strings.Contains(script, "(") && strings.Contains(script, ")") {
		return e.executeFunction(script)
	}

	// 处理变量引用
	e.mu.RLock()
	if val, ok := e.variables[script]; ok {
		e.mu.RUnlock()
		return val, nil
	}
	e.mu.RUnlock()

	return nil, nil
}

func (e *JSEngine) evaluateExpression(expr string) (interface{}, error) {
	expr = strings.TrimSpace(expr)

	// 字符串字面量
	if (strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"")) ||
		(strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'")) {
		return expr[1 : len(expr)-1], nil
	}

	// 数字字面量
	if n, err := strconv.ParseFloat(expr, 64); err == nil {
		return n, nil
	}

	// 布尔字面量
	if expr == "true" {
		return true, nil
	}
	if expr == "false" {
		return false, nil
	}

	// null/undefined
	if expr == "null" || expr == "undefined" {
		return nil, nil
	}

	// 函数调用
	if strings.Contains(expr, "(") {
		return e.executeFunction(expr)
	}

	// 变量引用
	e.mu.RLock()
	if val, ok := e.variables[expr]; ok {
		e.mu.RUnlock()
		return val, nil
	}
	e.mu.RUnlock()

	return expr, nil
}

func (e *JSEngine) executeFunction(script string) (interface{}, error) {
	// 解析函数名和参数
	parenIdx := strings.Index(script, "(")
	if parenIdx == -1 {
		return nil, fmt.Errorf("invalid function call")
	}

	funcName := strings.TrimSpace(script[:parenIdx])
	argsStr := script[parenIdx+1:]
	argsStr = strings.TrimSuffix(argsStr, ")")
	argsStr = strings.TrimSuffix(argsStr, ";")

	// 解析参数
	var args []interface{}
	if argsStr != "" {
		argParts := splitArgs(argsStr)
		for _, arg := range argParts {
			val, err := e.evaluateExpression(strings.TrimSpace(arg))
			if err != nil {
				return nil, err
			}
			args = append(args, val)
		}
	}

	// 查找并执行函数
	if fn, ok := e.functions[funcName]; ok {
		return fn(args), nil
	}

	return nil, fmt.Errorf("unknown function: %s", funcName)
}

func splitArgs(argsStr string) []string {
	var args []string
	var current strings.Builder
	depth := 0
	inString := false
	stringChar := byte(0)

	for i := 0; i < len(argsStr); i++ {
		c := argsStr[i]

		if !inString && (c == '"' || c == '\'') {
			inString = true
			stringChar = c
			current.WriteByte(c)
		} else if inString && c == stringChar {
			inString = false
			current.WriteByte(c)
		} else if !inString && c == '(' {
			depth++
			current.WriteByte(c)
		} else if !inString && c == ')' {
			depth--
			current.WriteByte(c)
		} else if !inString && c == ',' && depth == 0 {
			args = append(args, current.String())
			current.Reset()
		} else {
			current.WriteByte(c)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// SetVariable 设置变量
func (e *JSEngine) SetVariable(name string, value interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.variables[name] = value
}

// GetVariable 获取变量
func (e *JSEngine) GetVariable(name string) interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.variables[name]
}

// RegisterFunction 注册自定义函数
func (e *JSEngine) RegisterFunction(name string, fn JSFunction) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.functions[name] = fn
}

// AddEventHandler 添加事件处理器
func (e *JSEngine) AddEventHandler(event string, handler JSEventHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events[event] = append(e.events[event], handler)
}

// TriggerEvent 触发事件
func (e *JSEngine) TriggerEvent(event string) []interface{} {
	e.mu.RLock()
	handlers := e.events[event]
	e.mu.RUnlock()

	var results []interface{}
	for _, handler := range handlers {
		result, _ := e.Execute(handler.Script)
		results = append(results, result)
	}
	return results
}

// GetConsole 获取控制台
func (e *JSEngine) GetConsole() *JSConsole {
	return e.console
}

// ============================================================================
// 3. 低层可选内容（OC）层操作
// ============================================================================

// OCLayerManager 可选内容层管理器
type OCLayerManager struct {
	doc           *Document
	ocgs          []*OCGroup
	ocgMap        map[string]*OCGroup
	defaultConfig *OCConfig
	configs       []*OCConfig
	usage         map[string]*OCUsage
	mu            sync.RWMutex
}

// OCGroup 可选内容组
type OCGroup struct {
	ID          string
	Name        string
	Intent      []string
	Visible     bool
	Locked      bool
	PrintState  string // "ON", "OFF", "Unchanged"
	ViewState   string // "ON", "OFF", "Unchanged"
	ExportState string // "ON", "OFF", "Unchanged"
	Usage       *OCUsage
	Reference   Reference
}

// OCConfig 可选内容配置
type OCConfig struct {
	Name      string
	Creator   string
	BaseState string // "ON", "OFF", "Unchanged"
	OnList    []string
	OffList   []string
	Intent    []string
	AS        []*OCAutoState
	Order     []interface{}
	ListMode  string // "AllPages", "VisiblePages"
	RBGroups  [][]string
	Locked    []string
}

// OCAutoState 自动状态
type OCAutoState struct {
	Event    string // "View", "Print", "Export"
	Category []string
	OCGs     []string
}

// OCUsage 使用属性
type OCUsage struct {
	CreatorInfo *OCCreatorInfo
	Language    *OCLanguage
	Export      *OCExport
	Zoom        *OCZoom
	Print       *OCPrint
	View        *OCView
	User        *OCUser
	PageElement *OCPageElement
}

// OCCreatorInfo 创建者信息
type OCCreatorInfo struct {
	Creator string
	Subtype string
}

// OCLanguage 语言
type OCLanguage struct {
	Lang      string
	Preferred string
}

// OCExport 导出
type OCExport struct {
	ExportState string
}

// OCZoom 缩放
type OCZoom struct {
	Min float64
	Max float64
}

// OCPrint 打印
type OCPrint struct {
	Subtype    string
	PrintState string
}

// OCView 视图
type OCView struct {
	ViewState string
}

// OCUser 用户
type OCUser struct {
	Type string
	Name []string
}

// OCPageElement 页面元素
type OCPageElement struct {
	Subtype string
}

// NewOCLayerManager 创建 OC 层管理器
func NewOCLayerManager(doc *Document) *OCLayerManager {
	mgr := &OCLayerManager{
		doc:    doc,
		ocgs:   make([]*OCGroup, 0),
		ocgMap: make(map[string]*OCGroup),
		usage:  make(map[string]*OCUsage),
	}

	mgr.parse()
	return mgr
}

func (mgr *OCLayerManager) parse() {
	ocPropsRef := mgr.doc.Root.Get("OCProperties")
	if ocPropsRef == nil {
		return
	}

	ocPropsObj, err := mgr.doc.ResolveObject(ocPropsRef)
	if err != nil {
		return
	}

	ocProps, ok := ocPropsObj.(Dictionary)
	if !ok {
		return
	}

	// 解析 OCGs
	if ocgsRef := ocProps.Get("OCGs"); ocgsRef != nil {
		mgr.parseOCGs(ocgsRef)
	}

	// 解析默认配置
	if dRef := ocProps.Get("D"); dRef != nil {
		mgr.defaultConfig = mgr.parseConfig(dRef)
		mgr.applyConfig(mgr.defaultConfig)
	}

	// 解析其他配置
	if configsRef := ocProps.Get("Configs"); configsRef != nil {
		mgr.parseConfigs(configsRef)
	}
}

func (mgr *OCLayerManager) parseOCGs(ref Object) {
	obj, err := mgr.doc.ResolveObject(ref)
	if err != nil {
		return
	}

	arr, ok := obj.(Array)
	if !ok {
		return
	}

	for _, ocgRef := range arr {
		ocg := mgr.parseOCG(ocgRef)
		if ocg != nil {
			mgr.ocgs = append(mgr.ocgs, ocg)
			mgr.ocgMap[ocg.ID] = ocg
		}
	}
}

func (mgr *OCLayerManager) parseOCG(ref Object) *OCGroup {
	ocgObj, err := mgr.doc.ResolveObject(ref)
	if err != nil {
		return nil
	}

	ocgDict, ok := ocgObj.(Dictionary)
	if !ok {
		return nil
	}

	ocg := &OCGroup{
		Visible:     true,
		PrintState:  "Unchanged",
		ViewState:   "Unchanged",
		ExportState: "Unchanged",
	}

	// 获取引用
	if r, ok := ref.(Reference); ok {
		ocg.Reference = r
		ocg.ID = fmt.Sprintf("OCG_%d_%d", r.ObjectNumber, r.GenerationNumber)
	}

	// 获取名称
	if name := ocgDict.Get("Name"); name != nil {
		ocg.Name = objectToString(name)
	}

	// 获取意图
	if intent := ocgDict.Get("Intent"); intent != nil {
		switch v := intent.(type) {
		case Name:
			ocg.Intent = []string{string(v)}
		case Array:
			for _, item := range v {
				if n, ok := item.(Name); ok {
					ocg.Intent = append(ocg.Intent, string(n))
				}
			}
		}
	}

	// 解析使用属性
	if usageRef := ocgDict.Get("Usage"); usageRef != nil {
		ocg.Usage = mgr.parseUsage(usageRef)
		mgr.usage[ocg.ID] = ocg.Usage
	}

	return ocg
}

func (mgr *OCLayerManager) parseUsage(ref Object) *OCUsage {
	obj, err := mgr.doc.ResolveObject(ref)
	if err != nil {
		return nil
	}

	usageDict, ok := obj.(Dictionary)
	if !ok {
		return nil
	}

	usage := &OCUsage{}

	// 解析 CreatorInfo
	if ciRef := usageDict.Get("CreatorInfo"); ciRef != nil {
		if ciObj, err := mgr.doc.ResolveObject(ciRef); err == nil {
			if ciDict, ok := ciObj.(Dictionary); ok {
				usage.CreatorInfo = &OCCreatorInfo{}
				if creator := ciDict.Get("Creator"); creator != nil {
					usage.CreatorInfo.Creator = objectToString(creator)
				}
				if subtype, ok := ciDict.GetName("Subtype"); ok {
					usage.CreatorInfo.Subtype = string(subtype)
				}
			}
		}
	}

	// 解析 Language
	if langRef := usageDict.Get("Language"); langRef != nil {
		if langObj, err := mgr.doc.ResolveObject(langRef); err == nil {
			if langDict, ok := langObj.(Dictionary); ok {
				usage.Language = &OCLanguage{}
				if lang := langDict.Get("Lang"); lang != nil {
					usage.Language.Lang = objectToString(lang)
				}
				if pref, ok := langDict.GetName("Preferred"); ok {
					usage.Language.Preferred = string(pref)
				}
			}
		}
	}

	// 解析 Export
	if exportRef := usageDict.Get("Export"); exportRef != nil {
		if exportObj, err := mgr.doc.ResolveObject(exportRef); err == nil {
			if exportDict, ok := exportObj.(Dictionary); ok {
				usage.Export = &OCExport{}
				if state, ok := exportDict.GetName("ExportState"); ok {
					usage.Export.ExportState = string(state)
				}
			}
		}
	}

	// 解析 Zoom
	if zoomRef := usageDict.Get("Zoom"); zoomRef != nil {
		if zoomObj, err := mgr.doc.ResolveObject(zoomRef); err == nil {
			if zoomDict, ok := zoomObj.(Dictionary); ok {
				usage.Zoom = &OCZoom{}
				if min := zoomDict.Get("min"); min != nil {
					usage.Zoom.Min = objectToFloat(min)
				}
				if max := zoomDict.Get("max"); max != nil {
					usage.Zoom.Max = objectToFloat(max)
				}
			}
		}
	}

	// 解析 Print
	if printRef := usageDict.Get("Print"); printRef != nil {
		if printObj, err := mgr.doc.ResolveObject(printRef); err == nil {
			if printDict, ok := printObj.(Dictionary); ok {
				usage.Print = &OCPrint{}
				if subtype, ok := printDict.GetName("Subtype"); ok {
					usage.Print.Subtype = string(subtype)
				}
				if state, ok := printDict.GetName("PrintState"); ok {
					usage.Print.PrintState = string(state)
				}
			}
		}
	}

	// 解析 View
	if viewRef := usageDict.Get("View"); viewRef != nil {
		if viewObj, err := mgr.doc.ResolveObject(viewRef); err == nil {
			if viewDict, ok := viewObj.(Dictionary); ok {
				usage.View = &OCView{}
				if state, ok := viewDict.GetName("ViewState"); ok {
					usage.View.ViewState = string(state)
				}
			}
		}
	}

	// 解析 User
	if userRef := usageDict.Get("User"); userRef != nil {
		if userObj, err := mgr.doc.ResolveObject(userRef); err == nil {
			if userDict, ok := userObj.(Dictionary); ok {
				usage.User = &OCUser{}
				if t, ok := userDict.GetName("Type"); ok {
					usage.User.Type = string(t)
				}
				if name := userDict.Get("Name"); name != nil {
					switch v := name.(type) {
					case String:
						usage.User.Name = []string{string(v.Value)}
					case Array:
						for _, item := range v {
							if s, ok := item.(String); ok {
								usage.User.Name = append(usage.User.Name, string(s.Value))
							}
						}
					}
				}
			}
		}
	}

	// 解析 PageElement
	if peRef := usageDict.Get("PageElement"); peRef != nil {
		if peObj, err := mgr.doc.ResolveObject(peRef); err == nil {
			if peDict, ok := peObj.(Dictionary); ok {
				usage.PageElement = &OCPageElement{}
				if subtype, ok := peDict.GetName("Subtype"); ok {
					usage.PageElement.Subtype = string(subtype)
				}
			}
		}
	}

	return usage
}

func (mgr *OCLayerManager) parseConfig(ref Object) *OCConfig {
	obj, err := mgr.doc.ResolveObject(ref)
	if err != nil {
		return nil
	}

	configDict, ok := obj.(Dictionary)
	if !ok {
		return nil
	}

	config := &OCConfig{
		BaseState: "ON",
		ListMode:  "AllPages",
	}

	// 获取名称
	if name := configDict.Get("Name"); name != nil {
		config.Name = objectToString(name)
	}

	// 获取创建者
	if creator := configDict.Get("Creator"); creator != nil {
		config.Creator = objectToString(creator)
	}

	// 获取基础状态
	if baseState, ok := configDict.GetName("BaseState"); ok {
		config.BaseState = string(baseState)
	}

	// 获取 ON 列表
	if onRef := configDict.Get("ON"); onRef != nil {
		config.OnList = mgr.parseOCGList(onRef)
	}

	// 获取 OFF 列表
	if offRef := configDict.Get("OFF"); offRef != nil {
		config.OffList = mgr.parseOCGList(offRef)
	}

	// 获取意图
	if intent := configDict.Get("Intent"); intent != nil {
		switch v := intent.(type) {
		case Name:
			config.Intent = []string{string(v)}
		case Array:
			for _, item := range v {
				if n, ok := item.(Name); ok {
					config.Intent = append(config.Intent, string(n))
				}
			}
		}
	}

	// 获取 AS（自动状态）
	if asRef := configDict.Get("AS"); asRef != nil {
		config.AS = mgr.parseAutoStates(asRef)
	}

	// 获取列表模式
	if listMode, ok := configDict.GetName("ListMode"); ok {
		config.ListMode = string(listMode)
	}

	// 获取 RBGroups（单选按钮组）
	if rbRef := configDict.Get("RBGroups"); rbRef != nil {
		config.RBGroups = mgr.parseRBGroups(rbRef)
	}

	// 获取锁定列表
	if lockedRef := configDict.Get("Locked"); lockedRef != nil {
		config.Locked = mgr.parseOCGList(lockedRef)
	}

	return config
}

func (mgr *OCLayerManager) parseOCGList(ref Object) []string {
	var list []string

	obj, err := mgr.doc.ResolveObject(ref)
	if err != nil {
		return list
	}

	arr, ok := obj.(Array)
	if !ok {
		return list
	}

	for _, item := range arr {
		if r, ok := item.(Reference); ok {
			list = append(list, fmt.Sprintf("OCG_%d_%d", r.ObjectNumber, r.GenerationNumber))
		}
	}

	return list
}

func (mgr *OCLayerManager) parseAutoStates(ref Object) []*OCAutoState {
	var states []*OCAutoState

	obj, err := mgr.doc.ResolveObject(ref)
	if err != nil {
		return states
	}

	arr, ok := obj.(Array)
	if !ok {
		return states
	}

	for _, item := range arr {
		asObj, err := mgr.doc.ResolveObject(item)
		if err != nil {
			continue
		}

		asDict, ok := asObj.(Dictionary)
		if !ok {
			continue
		}

		as := &OCAutoState{}

		if event, ok := asDict.GetName("Event"); ok {
			as.Event = string(event)
		}

		if catRef := asDict.Get("Category"); catRef != nil {
			if catObj, err := mgr.doc.ResolveObject(catRef); err == nil {
				if catArr, ok := catObj.(Array); ok {
					for _, c := range catArr {
						if n, ok := c.(Name); ok {
							as.Category = append(as.Category, string(n))
						}
					}
				}
			}
		}

		if ocgsRef := asDict.Get("OCGs"); ocgsRef != nil {
			as.OCGs = mgr.parseOCGList(ocgsRef)
		}

		states = append(states, as)
	}

	return states
}

func (mgr *OCLayerManager) parseRBGroups(ref Object) [][]string {
	var groups [][]string

	obj, err := mgr.doc.ResolveObject(ref)
	if err != nil {
		return groups
	}

	arr, ok := obj.(Array)
	if !ok {
		return groups
	}

	for _, item := range arr {
		groupObj, err := mgr.doc.ResolveObject(item)
		if err != nil {
			continue
		}

		groupArr, ok := groupObj.(Array)
		if !ok {
			continue
		}

		var group []string
		for _, g := range groupArr {
			if r, ok := g.(Reference); ok {
				group = append(group, fmt.Sprintf("OCG_%d_%d", r.ObjectNumber, r.GenerationNumber))
			}
		}

		if len(group) > 0 {
			groups = append(groups, group)
		}
	}

	return groups
}

func (mgr *OCLayerManager) parseConfigs(ref Object) {
	obj, err := mgr.doc.ResolveObject(ref)
	if err != nil {
		return
	}

	arr, ok := obj.(Array)
	if !ok {
		return
	}

	for _, item := range arr {
		config := mgr.parseConfig(item)
		if config != nil {
			mgr.configs = append(mgr.configs, config)
		}
	}
}

func (mgr *OCLayerManager) applyConfig(config *OCConfig) {
	if config == nil {
		return
	}

	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	// 应用基础状态
	defaultVisible := config.BaseState != "OFF"
	for _, ocg := range mgr.ocgs {
		ocg.Visible = defaultVisible
	}

	// 应用 ON 列表
	for _, id := range config.OnList {
		if ocg, ok := mgr.ocgMap[id]; ok {
			ocg.Visible = true
		}
	}

	// 应用 OFF 列表
	for _, id := range config.OffList {
		if ocg, ok := mgr.ocgMap[id]; ok {
			ocg.Visible = false
		}
	}

	// 应用锁定列表
	for _, id := range config.Locked {
		if ocg, ok := mgr.ocgMap[id]; ok {
			ocg.Locked = true
		}
	}
}

// GetOCGroups 获取所有 OC 组
func (mgr *OCLayerManager) GetOCGroups() []*OCGroup {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	return append([]*OCGroup{}, mgr.ocgs...)
}

// GetOCGroup 根据 ID 获取 OC 组
func (mgr *OCLayerManager) GetOCGroup(id string) *OCGroup {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	return mgr.ocgMap[id]
}

// GetOCGroupByName 根据名称获取 OC 组
func (mgr *OCLayerManager) GetOCGroupByName(name string) *OCGroup {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	for _, ocg := range mgr.ocgs {
		if ocg.Name == name {
			return ocg
		}
	}
	return nil
}

// SetOCGroupVisibility 设置 OC 组可见性
func (mgr *OCLayerManager) SetOCGroupVisibility(id string, visible bool) error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	ocg, ok := mgr.ocgMap[id]
	if !ok {
		return fmt.Errorf("OCG not found: %s", id)
	}

	if ocg.Locked {
		return fmt.Errorf("OCG is locked: %s", id)
	}

	ocg.Visible = visible

	// 处理单选按钮组
	if visible && mgr.defaultConfig != nil {
		for _, group := range mgr.defaultConfig.RBGroups {
			inGroup := false
			for _, gid := range group {
				if gid == id {
					inGroup = true
					break
				}
			}

			if inGroup {
				for _, gid := range group {
					if gid != id {
						if other, ok := mgr.ocgMap[gid]; ok && !other.Locked {
							other.Visible = false
						}
					}
				}
			}
		}
	}

	return nil
}

// SetOCGroupPrintState 设置 OC 组打印状态
func (mgr *OCLayerManager) SetOCGroupPrintState(id string, state string) error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	ocg, ok := mgr.ocgMap[id]
	if !ok {
		return fmt.Errorf("OCG not found: %s", id)
	}

	if state != "ON" && state != "OFF" && state != "Unchanged" {
		return fmt.Errorf("invalid print state: %s", state)
	}

	ocg.PrintState = state
	return nil
}

// SetOCGroupViewState 设置 OC 组视图状态
func (mgr *OCLayerManager) SetOCGroupViewState(id string, state string) error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	ocg, ok := mgr.ocgMap[id]
	if !ok {
		return fmt.Errorf("OCG not found: %s", id)
	}

	if state != "ON" && state != "OFF" && state != "Unchanged" {
		return fmt.Errorf("invalid view state: %s", state)
	}

	ocg.ViewState = state
	return nil
}

// SetOCGroupExportState 设置 OC 组导出状态
func (mgr *OCLayerManager) SetOCGroupExportState(id string, state string) error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	ocg, ok := mgr.ocgMap[id]
	if !ok {
		return fmt.Errorf("OCG not found: %s", id)
	}

	if state != "ON" && state != "OFF" && state != "Unchanged" {
		return fmt.Errorf("invalid export state: %s", state)
	}

	ocg.ExportState = state
	return nil
}

// LockOCGroup 锁定 OC 组
func (mgr *OCLayerManager) LockOCGroup(id string) error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	ocg, ok := mgr.ocgMap[id]
	if !ok {
		return fmt.Errorf("OCG not found: %s", id)
	}

	ocg.Locked = true
	return nil
}

// UnlockOCGroup 解锁 OC 组
func (mgr *OCLayerManager) UnlockOCGroup(id string) error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	ocg, ok := mgr.ocgMap[id]
	if !ok {
		return fmt.Errorf("OCG not found: %s", id)
	}

	ocg.Locked = false
	return nil
}

// GetConfigs 获取所有配置
func (mgr *OCLayerManager) GetConfigs() []*OCConfig {
	return append([]*OCConfig{}, mgr.configs...)
}

// GetDefaultConfig 获取默认配置
func (mgr *OCLayerManager) GetDefaultConfig() *OCConfig {
	return mgr.defaultConfig
}

// ApplyConfig 应用配置
func (mgr *OCLayerManager) ApplyConfig(config *OCConfig) {
	mgr.applyConfig(config)
}

// ApplyConfigByName 按名称应用配置
func (mgr *OCLayerManager) ApplyConfigByName(name string) error {
	for _, config := range mgr.configs {
		if config.Name == name {
			mgr.applyConfig(config)
			return nil
		}
	}
	return fmt.Errorf("config not found: %s", name)
}

// GetVisibleOCGroups 获取可见的 OC 组
func (mgr *OCLayerManager) GetVisibleOCGroups() []*OCGroup {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	var visible []*OCGroup
	for _, ocg := range mgr.ocgs {
		if ocg.Visible {
			visible = append(visible, ocg)
		}
	}
	return visible
}

// GetPrintableOCGroups 获取可打印的 OC 组
func (mgr *OCLayerManager) GetPrintableOCGroups() []*OCGroup {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	var printable []*OCGroup
	for _, ocg := range mgr.ocgs {
		if ocg.PrintState == "ON" || (ocg.PrintState == "Unchanged" && ocg.Visible) {
			printable = append(printable, ocg)
		}
	}
	return printable
}

// GetExportableOCGroups 获取可导出的 OC 组
func (mgr *OCLayerManager) GetExportableOCGroups() []*OCGroup {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	var exportable []*OCGroup
	for _, ocg := range mgr.ocgs {
		if ocg.ExportState == "ON" || (ocg.ExportState == "Unchanged" && ocg.Visible) {
			exportable = append(exportable, ocg)
		}
	}
	return exportable
}

// IsContentVisible 检查内容是否可见
func (mgr *OCLayerManager) IsContentVisible(ocgRefs []Reference) bool {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	if len(ocgRefs) == 0 {
		return true
	}

	for _, ref := range ocgRefs {
		id := fmt.Sprintf("OCG_%d_%d", ref.ObjectNumber, ref.GenerationNumber)
		if ocg, ok := mgr.ocgMap[id]; ok {
			if !ocg.Visible {
				return false
			}
		}
	}

	return true
}

// ============================================================================
// 辅助函数
// ============================================================================

func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return f
	}
	return 0
}

func getCurrentTimestamp() int64 {
	return 0 // 简化实现
}
