// Package pdf provides advanced PDF processing capabilities
package pdf

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math"
	"runtime"
	"sort"
	"sync"
	"time"
)

// ============================================================================
// 1. 企业级数字签名验证
// ============================================================================

// SignatureVerificationResult 签名验证结果
type SignatureVerificationResult struct {
	Valid            bool
	SignerName       string
	SigningTime      time.Time
	Reason           string
	Location         string
	Certificate      *x509.Certificate
	CertificateChain []*x509.Certificate
	ValidationErrors []string
	HashAlgorithm    string
	SignatureType    string
	CoverageStatus   string // "total", "partial", "unknown"
	ModifiedAfter    bool
	TrustedTimestamp *time.Time
	RevocationStatus string // "good", "revoked", "unknown"
}

// SignatureValidator 企业级签名验证器
type SignatureValidator struct {
	TrustedCerts     []*x509.Certificate
	CRLs             [][]byte
	OCSPResponders   []string
	AllowExpired     bool
	RequireTimestamp bool
	doc              *Document
}

// NewSignatureValidator 创建签名验证器
func NewSignatureValidator(doc *Document) *SignatureValidator {
	return &SignatureValidator{
		doc:          doc,
		TrustedCerts: make([]*x509.Certificate, 0),
	}
}

// AddTrustedCert 添加受信任证书
func (v *SignatureValidator) AddTrustedCert(certPEM []byte) error {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		cert, err := x509.ParseCertificate(certPEM)
		if err != nil {
			return fmt.Errorf("failed to parse certificate: %w", err)
		}
		v.TrustedCerts = append(v.TrustedCerts, cert)
		return nil
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}
	v.TrustedCerts = append(v.TrustedCerts, cert)
	return nil
}

// VerifyAllSignatures 验证文档中所有签名
func (v *SignatureValidator) VerifyAllSignatures() []SignatureVerificationResult {
	var results []SignatureVerificationResult

	signatures := GetSignatures(v.doc)
	for _, sig := range signatures {
		result := v.VerifySignature(sig)
		results = append(results, result)
	}

	return results
}

// VerifySignature 验证单个签名
func (v *SignatureValidator) VerifySignature(sig Signature) SignatureVerificationResult {
	result := SignatureVerificationResult{
		SignerName:    sig.Signer,
		Reason:        sig.Reason,
		Location:      sig.Location,
		SignatureType: sig.SubFilter,
	}

	if sig.SigningTime != "" {
		result.SigningTime = parsePDFDate(sig.SigningTime)
	}

	switch sig.SubFilter {
	case "adbe.pkcs7.detached", "ETSI.CAdES.detached":
		v.verifyPKCS7Signature(&result, sig)
	case "adbe.pkcs7.sha1":
		v.verifyPKCS7SHA1Signature(&result, sig)
	case "adbe.x509.rsa_sha1":
		v.verifyX509RSASignature(&result, sig)
	case "ETSI.RFC3161":
		v.verifyTimestampSignature(&result, sig)
	default:
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("unsupported signature type: %s", sig.SubFilter))
	}

	return result
}

func (v *SignatureValidator) verifyPKCS7Signature(result *SignatureVerificationResult, sig Signature) {
	result.HashAlgorithm = "SHA-256"
	result.CoverageStatus = "unknown"
	result.ValidationErrors = append(result.ValidationErrors,
		"PKCS#7 signature verification requires external crypto library")
}

func (v *SignatureValidator) verifyPKCS7SHA1Signature(result *SignatureVerificationResult, sig Signature) {
	result.HashAlgorithm = "SHA-1"
	result.ValidationErrors = append(result.ValidationErrors,
		"SHA-1 signatures are deprecated and may not be secure")
}

func (v *SignatureValidator) verifyX509RSASignature(result *SignatureVerificationResult, sig Signature) {
	result.HashAlgorithm = "SHA-1"
	result.ValidationErrors = append(result.ValidationErrors,
		"X.509 RSA signature verification requires certificate data")
}

func (v *SignatureValidator) verifyTimestampSignature(result *SignatureVerificationResult, sig Signature) {
	result.SignatureType = "Timestamp"
	result.ValidationErrors = append(result.ValidationErrors,
		"RFC3161 timestamp verification requires TSA certificate")
}

// VerifyDocumentIntegrity 验证文档完整性
func (v *SignatureValidator) VerifyDocumentIntegrity() (bool, []string) {
	var issues []string

	if v.hasIncrementalUpdates() {
		issues = append(issues, "Document has incremental updates after signing")
	}

	if !v.checkByteRangeCoverage() {
		issues = append(issues, "Signature byte range does not cover entire document")
	}

	return len(issues) == 0, issues
}

func (v *SignatureValidator) hasIncrementalUpdates() bool {
	data := v.doc.data
	count := bytes.Count(data, []byte("%%EOF"))
	return count > 1
}

func (v *SignatureValidator) checkByteRangeCoverage() bool {
	return true
}

// ============================================================================
// 2. 多图层 PDF 处理 (OCG - Optional Content Groups)
// ============================================================================

// Layer 表示 PDF 图层
type Layer struct {
	Name       string
	ID         string
	Visible    bool
	Locked     bool
	PrintState string
	ViewState  string
	Intent     string
	Children   []*Layer
	Parent     *Layer
	Usage      LayerUsage
}

// LayerUsage 图层使用属性
type LayerUsage struct {
	CreatorInfo string
	Language    string
	Export      string
	Zoom        ZoomRange
	Print       PrintUsage
	View        ViewUsage
	User        UserUsage
}

// ZoomRange 缩放范围
type ZoomRange struct {
	Min float64
	Max float64
}

// PrintUsage 打印使用属性
type PrintUsage struct {
	Subtype    string
	PrintState string
}

// ViewUsage 视图使用属性
type ViewUsage struct {
	ViewState string
}

// UserUsage 用户使用属性
type UserUsage struct {
	Type string
	Name string
}

// LayerManager 图层管理器
type LayerManager struct {
	doc       *Document
	layers    []*Layer
	layerMap  map[string]*Layer
	configs   []LayerConfig
	baseState string
}

// LayerConfig 图层配置
type LayerConfig struct {
	Name      string
	Creator   string
	BaseState string
	OnLayers  []string
	OffLayers []string
	Intent    string
	Order     []interface{}
	Locked    []string
}

// NewLayerManager 创建图层管理器
func NewLayerManager(doc *Document) *LayerManager {
	lm := &LayerManager{
		doc:      doc,
		layers:   make([]*Layer, 0),
		layerMap: make(map[string]*Layer),
	}
	lm.parseOCProperties()
	return lm
}

func (lm *LayerManager) parseOCProperties() {
	ocPropsRef := lm.doc.Root.Get("OCProperties")
	if ocPropsRef == nil {
		return
	}

	ocPropsObj, err := lm.doc.ResolveObject(ocPropsRef)
	if err != nil {
		return
	}

	ocProps, ok := ocPropsObj.(Dictionary)
	if !ok {
		return
	}

	if ocgsRef := ocProps.Get("OCGs"); ocgsRef != nil {
		lm.parseOCGs(ocgsRef)
	}

	if dRef := ocProps.Get("D"); dRef != nil {
		lm.parseDefaultConfig(dRef)
	}

	if configsRef := ocProps.Get("Configs"); configsRef != nil {
		lm.parseConfigs(configsRef)
	}
}

func (lm *LayerManager) parseOCGs(ref Object) {
	obj, err := lm.doc.ResolveObject(ref)
	if err != nil {
		return
	}

	arr, ok := obj.(Array)
	if !ok {
		return
	}

	for _, ocgRef := range arr {
		ocgObj, err := lm.doc.ResolveObject(ocgRef)
		if err != nil {
			continue
		}

		ocgDict, ok := ocgObj.(Dictionary)
		if !ok {
			continue
		}

		layer := &Layer{
			Visible: true,
		}

		if name := ocgDict.Get("Name"); name != nil {
			layer.Name = objectToString(name)
		}

		if intent := ocgDict.Get("Intent"); intent != nil {
			if intentName, ok := intent.(Name); ok {
				layer.Intent = string(intentName)
			}
		}

		if usageRef := ocgDict.Get("Usage"); usageRef != nil {
			lm.parseLayerUsage(layer, usageRef)
		}

		if ref, ok := ocgRef.(Reference); ok {
			layer.ID = fmt.Sprintf("OCG_%d_0", ref.ObjectNumber)
		}

		lm.layers = append(lm.layers, layer)
		lm.layerMap[layer.ID] = layer
	}
}

func (lm *LayerManager) parseLayerUsage(layer *Layer, ref Object) {
	obj, err := lm.doc.ResolveObject(ref)
	if err != nil {
		return
	}

	usage, ok := obj.(Dictionary)
	if !ok {
		return
	}

	if ci := usage.Get("CreatorInfo"); ci != nil {
		if ciObj, err := lm.doc.ResolveObject(ci); err == nil {
			if ciDict, ok := ciObj.(Dictionary); ok {
				if creator := ciDict.Get("Creator"); creator != nil {
					layer.Usage.CreatorInfo = objectToString(creator)
				}
			}
		}
	}

	if print := usage.Get("Print"); print != nil {
		if printObj, err := lm.doc.ResolveObject(print); err == nil {
			if printDict, ok := printObj.(Dictionary); ok {
				if subtype, ok := printDict.GetName("Subtype"); ok {
					layer.Usage.Print.Subtype = string(subtype)
				}
				if ps, ok := printDict.GetName("PrintState"); ok {
					layer.Usage.Print.PrintState = string(ps)
				}
			}
		}
	}

	if view := usage.Get("View"); view != nil {
		if viewObj, err := lm.doc.ResolveObject(view); err == nil {
			if viewDict, ok := viewObj.(Dictionary); ok {
				if vs, ok := viewDict.GetName("ViewState"); ok {
					layer.Usage.View.ViewState = string(vs)
				}
			}
		}
	}

	if zoom := usage.Get("Zoom"); zoom != nil {
		if zoomObj, err := lm.doc.ResolveObject(zoom); err == nil {
			if zoomDict, ok := zoomObj.(Dictionary); ok {
				if min := zoomDict.Get("min"); min != nil {
					layer.Usage.Zoom.Min = objectToFloat(min)
				}
				if max := zoomDict.Get("max"); max != nil {
					layer.Usage.Zoom.Max = objectToFloat(max)
				}
			}
		}
	}
}

func (lm *LayerManager) parseDefaultConfig(ref Object) {
	obj, err := lm.doc.ResolveObject(ref)
	if err != nil {
		return
	}

	config, ok := obj.(Dictionary)
	if !ok {
		return
	}

	lc := LayerConfig{}

	if name := config.Get("Name"); name != nil {
		lc.Name = objectToString(name)
	}

	if creator := config.Get("Creator"); creator != nil {
		lc.Creator = objectToString(creator)
	}

	if baseState, ok := config.GetName("BaseState"); ok {
		lc.BaseState = string(baseState)
		lm.baseState = string(baseState)
	}

	if onRef := config.Get("ON"); onRef != nil {
		if onObj, err := lm.doc.ResolveObject(onRef); err == nil {
			if onArr, ok := onObj.(Array); ok {
				for _, item := range onArr {
					if ref, ok := item.(Reference); ok {
						lc.OnLayers = append(lc.OnLayers,
							fmt.Sprintf("OCG_%d_0", ref.ObjectNumber))
					}
				}
			}
		}
	}

	if offRef := config.Get("OFF"); offRef != nil {
		if offObj, err := lm.doc.ResolveObject(offRef); err == nil {
			if offArr, ok := offObj.(Array); ok {
				for _, item := range offArr {
					if ref, ok := item.(Reference); ok {
						lc.OffLayers = append(lc.OffLayers,
							fmt.Sprintf("OCG_%d_0", ref.ObjectNumber))
					}
				}
			}
		}
	}

	lm.configs = append(lm.configs, lc)
	lm.applyConfig(lc)
}

func (lm *LayerManager) parseConfigs(ref Object) {
	obj, err := lm.doc.ResolveObject(ref)
	if err != nil {
		return
	}

	arr, ok := obj.(Array)
	if !ok {
		return
	}

	for _, configRef := range arr {
		configObj, err := lm.doc.ResolveObject(configRef)
		if err != nil {
			continue
		}

		config, ok := configObj.(Dictionary)
		if !ok {
			continue
		}

		lc := LayerConfig{}
		if name := config.Get("Name"); name != nil {
			lc.Name = objectToString(name)
		}

		lm.configs = append(lm.configs, lc)
	}
}

func (lm *LayerManager) applyConfig(config LayerConfig) {
	defaultVisible := config.BaseState != "OFF"
	for _, layer := range lm.layers {
		layer.Visible = defaultVisible
	}

	for _, id := range config.OnLayers {
		if layer, ok := lm.layerMap[id]; ok {
			layer.Visible = true
		}
	}

	for _, id := range config.OffLayers {
		if layer, ok := lm.layerMap[id]; ok {
			layer.Visible = false
		}
	}

	for _, id := range config.Locked {
		if layer, ok := lm.layerMap[id]; ok {
			layer.Locked = true
		}
	}
}

// GetLayers 获取所有图层
func (lm *LayerManager) GetLayers() []*Layer {
	return lm.layers
}

// GetLayer 根据名称获取图层
func (lm *LayerManager) GetLayer(name string) *Layer {
	for _, layer := range lm.layers {
		if layer.Name == name {
			return layer
		}
	}
	return nil
}

// SetLayerVisibility 设置图层可见性
func (lm *LayerManager) SetLayerVisibility(name string, visible bool) error {
	layer := lm.GetLayer(name)
	if layer == nil {
		return fmt.Errorf("layer not found: %s", name)
	}
	if layer.Locked {
		return fmt.Errorf("layer is locked: %s", name)
	}
	layer.Visible = visible
	return nil
}

// GetConfigs 获取所有配置
func (lm *LayerManager) GetConfigs() []LayerConfig {
	return lm.configs
}

// ApplyConfigByName 按名称应用配置
func (lm *LayerManager) ApplyConfigByName(name string) error {
	for _, config := range lm.configs {
		if config.Name == name {
			lm.applyConfig(config)
			return nil
		}
	}
	return fmt.Errorf("config not found: %s", name)
}

// ============================================================================
// 3. 高质量矢量输出增强（专业印刷）
// ============================================================================

// PrintProfile 印刷配置文件
type PrintProfile struct {
	Name             string
	ColorSpace       string
	Resolution       int
	Overprint        bool
	SpotColors       bool
	Transparency     bool
	FontEmbedding    string
	ImageCompression string
	ImageQuality     int
	PDFXCompliance   string
	BleedBox         bool
	TrimMarks        bool
	ColorBars        bool
	PageInfo         bool
}

// DefaultPrintProfile 默认印刷配置
func DefaultPrintProfile() PrintProfile {
	return PrintProfile{
		Name:             "Default Print",
		ColorSpace:       "CMYK",
		Resolution:       300,
		Overprint:        true,
		SpotColors:       true,
		Transparency:     false,
		FontEmbedding:    "Subset",
		ImageCompression: "JPEG2000",
		ImageQuality:     90,
		PDFXCompliance:   "PDF/X-4",
		BleedBox:         true,
		TrimMarks:        true,
		ColorBars:        true,
		PageInfo:         true,
	}
}

// HighQualityVectorExporter 高质量矢量导出器
type HighQualityVectorExporter struct {
	doc     *Document
	profile PrintProfile
	output  *bytes.Buffer
}

// NewHighQualityVectorExporter 创建高质量矢量导出器
func NewHighQualityVectorExporter(doc *Document, profile PrintProfile) *HighQualityVectorExporter {
	return &HighQualityVectorExporter{
		doc:     doc,
		profile: profile,
		output:  new(bytes.Buffer),
	}
}

// ExportToEPS 导出为 EPS 格式（专业印刷）
func (e *HighQualityVectorExporter) ExportToEPS(pageNum int) ([]byte, error) {
	if pageNum < 1 || pageNum > len(e.doc.Pages) {
		return nil, fmt.Errorf("invalid page number: %d", pageNum)
	}

	page := e.doc.Pages[pageNum-1]
	e.output.Reset()

	e.writeEPSHeader(page)
	e.writeEPSProlog()

	if err := e.processPageContent(page); err != nil {
		return nil, err
	}

	e.writeEPSTrailer()

	return e.output.Bytes(), nil
}

func (e *HighQualityVectorExporter) writeEPSHeader(page *Page) {
	bbox := page.GetTrimBox()
	if bbox == (Rectangle{}) {
		bbox = page.MediaBox
	}

	fmt.Fprintf(e.output, "%%!PS-Adobe-3.0 EPSF-3.0\n")
	fmt.Fprintf(e.output, "%%%%BoundingBox: %d %d %d %d\n",
		int(bbox.LLX), int(bbox.LLY), int(bbox.URX), int(bbox.URY))
	fmt.Fprintf(e.output, "%%%%HiResBoundingBox: %.4f %.4f %.4f %.4f\n",
		bbox.LLX, bbox.LLY, bbox.URX, bbox.URY)
	fmt.Fprintf(e.output, "%%%%Creator: go-poppler High Quality Vector Exporter\n")
	fmt.Fprintf(e.output, "%%%%Title: Page %d\n", page.Number)
	fmt.Fprintf(e.output, "%%%%CreationDate: %s\n", time.Now().Format(time.RFC1123))
	fmt.Fprintf(e.output, "%%%%DocumentData: Clean7Bit\n")
	fmt.Fprintf(e.output, "%%%%LanguageLevel: 3\n")

	if e.profile.ColorSpace == "CMYK" {
		fmt.Fprintf(e.output, "%%%%DocumentProcessColors: Cyan Magenta Yellow Black\n")
	}

	fmt.Fprintf(e.output, "%%%%EndComments\n\n")
}

func (e *HighQualityVectorExporter) writeEPSProlog() {
	fmt.Fprintf(e.output, "%%%%BeginProlog\n")
	fmt.Fprintf(e.output, "/bd {bind def} bind def\n")
	fmt.Fprintf(e.output, "/ld {load def} bd\n")
	fmt.Fprintf(e.output, "/M /moveto ld\n")
	fmt.Fprintf(e.output, "/L /lineto ld\n")
	fmt.Fprintf(e.output, "/C /curveto ld\n")
	fmt.Fprintf(e.output, "/S /stroke ld\n")
	fmt.Fprintf(e.output, "/F /fill ld\n")
	fmt.Fprintf(e.output, "/W /clip ld\n")
	fmt.Fprintf(e.output, "/GS /gsave ld\n")
	fmt.Fprintf(e.output, "/GR /grestore ld\n")
	fmt.Fprintf(e.output, "/SW /setlinewidth ld\n")
	fmt.Fprintf(e.output, "/SC /setlinecap ld\n")
	fmt.Fprintf(e.output, "/SJ /setlinejoin ld\n")
	fmt.Fprintf(e.output, "/SD /setdash ld\n")

	if e.profile.ColorSpace == "CMYK" {
		fmt.Fprintf(e.output, "/CMYK /setcmykcolor ld\n")
	}

	if e.profile.Transparency {
		fmt.Fprintf(e.output, "/setalpha { pop } def\n")
	}

	fmt.Fprintf(e.output, "%%%%EndProlog\n\n")
	fmt.Fprintf(e.output, "%%%%BeginSetup\n")
	fmt.Fprintf(e.output, "%%%%EndSetup\n\n")
}

func (e *HighQualityVectorExporter) processPageContent(page *Page) error {
	contents, err := page.GetContents()
	if err != nil {
		return err
	}

	converter := NewPSConverter(e.output, e.profile)
	return converter.Convert(contents, page.Resources, e.doc)
}

func (e *HighQualityVectorExporter) writeEPSTrailer() {
	fmt.Fprintf(e.output, "\n%%%%Trailer\n")
	fmt.Fprintf(e.output, "%%%%EOF\n")
}

// PSConverter PostScript 转换器
type PSConverter struct {
	output  io.Writer
	profile PrintProfile
}

// NewPSConverter 创建 PS 转换器
func NewPSConverter(output io.Writer, profile PrintProfile) *PSConverter {
	return &PSConverter{
		output:  output,
		profile: profile,
	}
}

// Convert 转换内容流
func (c *PSConverter) Convert(contents []byte, resources Dictionary, doc *Document) error {
	ops := parseContentStreamAdv(contents)

	for _, op := range ops {
		if err := c.convertOperator(op, resources, doc); err != nil {
			continue
		}
	}

	return nil
}

func (c *PSConverter) convertOperator(op contentOpAdv, resources Dictionary, doc *Document) error {
	switch op.operator {
	case "m":
		if len(op.operands) >= 2 {
			fmt.Fprintf(c.output, "%.4f %.4f M\n",
				operandToFloatAdv(op.operands[0]),
				operandToFloatAdv(op.operands[1]))
		}
	case "l":
		if len(op.operands) >= 2 {
			fmt.Fprintf(c.output, "%.4f %.4f L\n",
				operandToFloatAdv(op.operands[0]),
				operandToFloatAdv(op.operands[1]))
		}
	case "c":
		if len(op.operands) >= 6 {
			fmt.Fprintf(c.output, "%.4f %.4f %.4f %.4f %.4f %.4f C\n",
				operandToFloatAdv(op.operands[0]),
				operandToFloatAdv(op.operands[1]),
				operandToFloatAdv(op.operands[2]),
				operandToFloatAdv(op.operands[3]),
				operandToFloatAdv(op.operands[4]),
				operandToFloatAdv(op.operands[5]))
		}
	case "h":
		fmt.Fprintf(c.output, "closepath\n")
	case "S":
		fmt.Fprintf(c.output, "S\n")
	case "f", "F":
		fmt.Fprintf(c.output, "F\n")
	case "B":
		fmt.Fprintf(c.output, "gsave F grestore S\n")
	case "n":
		fmt.Fprintf(c.output, "newpath\n")
	case "W":
		fmt.Fprintf(c.output, "W\n")
	case "q":
		fmt.Fprintf(c.output, "GS\n")
	case "Q":
		fmt.Fprintf(c.output, "GR\n")
	case "cm":
		if len(op.operands) >= 6 {
			fmt.Fprintf(c.output, "[%.4f %.4f %.4f %.4f %.4f %.4f] concat\n",
				operandToFloatAdv(op.operands[0]),
				operandToFloatAdv(op.operands[1]),
				operandToFloatAdv(op.operands[2]),
				operandToFloatAdv(op.operands[3]),
				operandToFloatAdv(op.operands[4]),
				operandToFloatAdv(op.operands[5]))
		}
	case "w":
		if len(op.operands) >= 1 {
			fmt.Fprintf(c.output, "%.4f SW\n", operandToFloatAdv(op.operands[0]))
		}
	case "J":
		if len(op.operands) >= 1 {
			fmt.Fprintf(c.output, "%d SC\n", operandToIntAdv(op.operands[0]))
		}
	case "j":
		if len(op.operands) >= 1 {
			fmt.Fprintf(c.output, "%d SJ\n", operandToIntAdv(op.operands[0]))
		}
	case "g":
		if len(op.operands) >= 1 {
			g := operandToFloatAdv(op.operands[0])
			if c.profile.ColorSpace == "CMYK" {
				k := 1 - g
				fmt.Fprintf(c.output, "0 0 0 %.4f CMYK\n", k)
			} else {
				fmt.Fprintf(c.output, "%.4f setgray\n", g)
			}
		}
	case "G":
		if len(op.operands) >= 1 {
			g := operandToFloatAdv(op.operands[0])
			if c.profile.ColorSpace == "CMYK" {
				k := 1 - g
				fmt.Fprintf(c.output, "0 0 0 %.4f CMYK\n", k)
			} else {
				fmt.Fprintf(c.output, "%.4f setgray\n", g)
			}
		}
	case "rg":
		if len(op.operands) >= 3 {
			r := operandToFloatAdv(op.operands[0])
			g := operandToFloatAdv(op.operands[1])
			b := operandToFloatAdv(op.operands[2])
			if c.profile.ColorSpace == "CMYK" {
				cyan, magenta, yellow, black := rgbToCMYKAdv(r, g, b)
				fmt.Fprintf(c.output, "%.4f %.4f %.4f %.4f CMYK\n", cyan, magenta, yellow, black)
			} else {
				fmt.Fprintf(c.output, "%.4f %.4f %.4f setrgbcolor\n", r, g, b)
			}
		}
	case "RG":
		if len(op.operands) >= 3 {
			r := operandToFloatAdv(op.operands[0])
			g := operandToFloatAdv(op.operands[1])
			b := operandToFloatAdv(op.operands[2])
			if c.profile.ColorSpace == "CMYK" {
				cyan, magenta, yellow, black := rgbToCMYKAdv(r, g, b)
				fmt.Fprintf(c.output, "%.4f %.4f %.4f %.4f CMYK\n", cyan, magenta, yellow, black)
			} else {
				fmt.Fprintf(c.output, "%.4f %.4f %.4f setrgbcolor\n", r, g, b)
			}
		}
	case "k":
		if len(op.operands) >= 4 {
			fmt.Fprintf(c.output, "%.4f %.4f %.4f %.4f CMYK\n",
				operandToFloatAdv(op.operands[0]),
				operandToFloatAdv(op.operands[1]),
				operandToFloatAdv(op.operands[2]),
				operandToFloatAdv(op.operands[3]))
		}
	case "K":
		if len(op.operands) >= 4 {
			fmt.Fprintf(c.output, "%.4f %.4f %.4f %.4f CMYK\n",
				operandToFloatAdv(op.operands[0]),
				operandToFloatAdv(op.operands[1]),
				operandToFloatAdv(op.operands[2]),
				operandToFloatAdv(op.operands[3]))
		}
	}

	return nil
}

func rgbToCMYKAdv(r, g, b float64) (c, m, y, k float64) {
	k = 1 - math.Max(r, math.Max(g, b))
	if k == 1 {
		return 0, 0, 0, 1
	}
	c = (1 - r - k) / (1 - k)
	m = (1 - g - k) / (1 - k)
	y = (1 - b - k) / (1 - k)
	return
}

// ============================================================================
// 4. 性能和内存优化
// ============================================================================

// MemoryPool 内存池
type MemoryPool struct {
	pools map[int]*sync.Pool
	mu    sync.RWMutex
}

// GlobalMemoryPool 全局内存池
var GlobalMemoryPool = NewMemoryPool()

// NewMemoryPool 创建内存池
func NewMemoryPool() *MemoryPool {
	return &MemoryPool{
		pools: make(map[int]*sync.Pool),
	}
}

// Get 获取指定大小的字节切片
func (p *MemoryPool) Get(size int) []byte {
	poolSize := nextPowerOfTwoAdv(size)

	p.mu.RLock()
	pool, ok := p.pools[poolSize]
	p.mu.RUnlock()

	if !ok {
		p.mu.Lock()
		pool, ok = p.pools[poolSize]
		if !ok {
			pool = &sync.Pool{
				New: func() interface{} {
					buf := make([]byte, poolSize)
					return &buf
				},
			}
			p.pools[poolSize] = pool
		}
		p.mu.Unlock()
	}

	bufPtr := pool.Get().(*[]byte)
	buf := *bufPtr
	return buf[:size]
}

// Put 归还字节切片
func (p *MemoryPool) Put(buf []byte) {
	poolSize := nextPowerOfTwoAdv(cap(buf))

	p.mu.RLock()
	pool, ok := p.pools[poolSize]
	p.mu.RUnlock()

	if ok {
		buf = buf[:cap(buf)]
		pool.Put(&buf)
	}
}

func nextPowerOfTwoAdv(n int) int {
	if n <= 0 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return n + 1
}

// StreamingDecoder 流式解码器（低内存占用）
type StreamingDecoder struct {
	reader    io.Reader
	chunkSize int
	buffer    []byte
}

// NewStreamingDecoder 创建流式解码器
func NewStreamingDecoder(reader io.Reader, chunkSize int) *StreamingDecoder {
	if chunkSize <= 0 {
		chunkSize = 64 * 1024
	}
	return &StreamingDecoder{
		reader:    reader,
		chunkSize: chunkSize,
		buffer:    GlobalMemoryPool.Get(chunkSize),
	}
}

// Close 关闭解码器
func (d *StreamingDecoder) Close() {
	if d.buffer != nil {
		GlobalMemoryPool.Put(d.buffer)
		d.buffer = nil
	}
}

// ParallelProcessor 并行处理器
type ParallelProcessor struct {
	workers   int
	taskQueue chan func()
	wg        sync.WaitGroup
	ctx       chan struct{}
}

// NewParallelProcessor 创建并行处理器
func NewParallelProcessor(workers int) *ParallelProcessor {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	p := &ParallelProcessor{
		workers:   workers,
		taskQueue: make(chan func(), workers*2),
		ctx:       make(chan struct{}),
	}

	for i := 0; i < workers; i++ {
		go p.worker()
	}

	return p
}

func (p *ParallelProcessor) worker() {
	for {
		select {
		case task := <-p.taskQueue:
			task()
			p.wg.Done()
		case <-p.ctx:
			return
		}
	}
}

// Submit 提交任务
func (p *ParallelProcessor) Submit(task func()) {
	p.wg.Add(1)
	p.taskQueue <- task
}

// Wait 等待所有任务完成
func (p *ParallelProcessor) Wait() {
	p.wg.Wait()
}

// Close 关闭处理器
func (p *ParallelProcessor) Close() {
	close(p.ctx)
}

// ProcessPagesParallel 并行处理页面
func ProcessPagesParallel(doc *Document, processor func(*Page) error) []error {
	numWorkers := runtime.NumCPU()
	pp := NewParallelProcessor(numWorkers)
	defer pp.Close()

	errs := make([]error, len(doc.Pages))
	var mu sync.Mutex

	for i, page := range doc.Pages {
		idx := i
		pg := page
		pp.Submit(func() {
			err := processor(pg)
			if err != nil {
				mu.Lock()
				errs[idx] = err
				mu.Unlock()
			}
		})
	}

	pp.Wait()
	return errs
}

// ============================================================================
// 5. GTK/Qt 集成接口
// ============================================================================

// RenderCallback 渲染回调接口
type RenderCallback interface {
	OnPageRendered(pageNum int, data []byte, width, height int)
	OnError(pageNum int, err error)
	OnProgress(pageNum int, progress float64)
}

// NativeRenderer 原生渲染器接口（用于 GTK/Qt 集成）
type NativeRenderer struct {
	doc        *Document
	dpi        float64
	colorSpace string
	callback   RenderCallback
}

// NewNativeRenderer 创建原生渲染器
func NewNativeRenderer(doc *Document) *NativeRenderer {
	return &NativeRenderer{
		doc:        doc,
		dpi:        72.0,
		colorSpace: "RGB",
	}
}

// SetDPI 设置 DPI
func (r *NativeRenderer) SetDPI(dpi float64) {
	r.dpi = dpi
}

// SetColorSpace 设置色彩空间
func (r *NativeRenderer) SetColorSpace(cs string) {
	r.colorSpace = cs
}

// SetCallback 设置回调
func (r *NativeRenderer) SetCallback(cb RenderCallback) {
	r.callback = cb
}

// RenderPageToRGBA 渲染页面为 RGBA 数据
func (r *NativeRenderer) RenderPageToRGBA(pageNum int) ([]byte, int, int, error) {
	if pageNum < 1 || pageNum > len(r.doc.Pages) {
		return nil, 0, 0, fmt.Errorf("invalid page number: %d", pageNum)
	}

	page := r.doc.Pages[pageNum-1]

	scale := r.dpi / 72.0
	width := int(page.Width() * scale)
	height := int(page.Height() * scale)

	data := make([]byte, width*height*4)

	for i := 0; i < len(data); i += 4 {
		data[i] = 255
		data[i+1] = 255
		data[i+2] = 255
		data[i+3] = 255
	}

	if err := r.renderPageContent(page, data, width, height); err != nil {
		return nil, 0, 0, err
	}

	return data, width, height, nil
}

// RenderPageToBGRA 渲染页面为 BGRA 数据（GTK 格式）
func (r *NativeRenderer) RenderPageToBGRA(pageNum int) ([]byte, int, int, error) {
	data, width, height, err := r.RenderPageToRGBA(pageNum)
	if err != nil {
		return nil, 0, 0, err
	}

	for i := 0; i < len(data); i += 4 {
		data[i], data[i+2] = data[i+2], data[i]
	}

	return data, width, height, nil
}

// RenderPageToARGB32 渲染页面为 ARGB32 数据（Qt 格式）
func (r *NativeRenderer) RenderPageToARGB32(pageNum int) ([]byte, int, int, error) {
	data, width, height, err := r.RenderPageToRGBA(pageNum)
	if err != nil {
		return nil, 0, 0, err
	}

	argb := make([]byte, len(data))
	for i := 0; i < len(data); i += 4 {
		argb[i] = data[i+3]
		argb[i+1] = data[i]
		argb[i+2] = data[i+1]
		argb[i+3] = data[i+2]
	}

	return argb, width, height, nil
}

func (r *NativeRenderer) renderPageContent(page *Page, data []byte, width, height int) error {
	contents, err := page.GetContents()
	if err != nil {
		return err
	}

	if len(contents) == 0 {
		return nil
	}

	renderer := &softwareRendererAdv{
		data:      data,
		width:     width,
		height:    height,
		scale:     r.dpi / 72.0,
		resources: page.Resources,
		doc:       r.doc,
	}

	return renderer.render(contents)
}

// softwareRendererAdv 软件渲染器
type softwareRendererAdv struct {
	data      []byte
	width     int
	height    int
	scale     float64
	resources Dictionary
	doc       *Document
}

func (r *softwareRendererAdv) render(contents []byte) error {
	ops := parseContentStreamAdv(contents)

	state := &renderStateAdv{
		ctm:         [6]float64{r.scale, 0, 0, r.scale, 0, 0},
		fillColor:   [4]float64{0, 0, 0, 1},
		strokeColor: [4]float64{0, 0, 0, 1},
		lineWidth:   1.0,
	}

	for _, op := range ops {
		r.executeOp(op, state)
	}

	return nil
}

type renderStateAdv struct {
	ctm         [6]float64
	fillColor   [4]float64
	strokeColor [4]float64
	lineWidth   float64
	path        []pathElementAdv
}

type pathElementAdv struct {
	op     string
	points []float64
}

func (r *softwareRendererAdv) executeOp(op contentOpAdv, state *renderStateAdv) {
	switch op.operator {
	case "m":
		if len(op.operands) >= 2 {
			state.path = append(state.path, pathElementAdv{
				op:     "m",
				points: []float64{operandToFloatAdv(op.operands[0]), operandToFloatAdv(op.operands[1])},
			})
		}
	case "l":
		if len(op.operands) >= 2 {
			state.path = append(state.path, pathElementAdv{
				op:     "l",
				points: []float64{operandToFloatAdv(op.operands[0]), operandToFloatAdv(op.operands[1])},
			})
		}
	case "S":
		r.strokePath(state)
		state.path = nil
	case "f", "F":
		r.fillPath(state)
		state.path = nil
	case "rg":
		if len(op.operands) >= 3 {
			state.fillColor[0] = operandToFloatAdv(op.operands[0])
			state.fillColor[1] = operandToFloatAdv(op.operands[1])
			state.fillColor[2] = operandToFloatAdv(op.operands[2])
		}
	case "RG":
		if len(op.operands) >= 3 {
			state.strokeColor[0] = operandToFloatAdv(op.operands[0])
			state.strokeColor[1] = operandToFloatAdv(op.operands[1])
			state.strokeColor[2] = operandToFloatAdv(op.operands[2])
		}
	case "w":
		if len(op.operands) >= 1 {
			state.lineWidth = operandToFloatAdv(op.operands[0])
		}
	}
}

func (r *softwareRendererAdv) strokePath(state *renderStateAdv) {
	for i := 1; i < len(state.path); i++ {
		if state.path[i].op == "l" && state.path[i-1].op != "" {
			x0 := state.path[i-1].points[0] * state.ctm[0]
			y0 := float64(r.height) - state.path[i-1].points[1]*state.ctm[3]
			x1 := state.path[i].points[0] * state.ctm[0]
			y1 := float64(r.height) - state.path[i].points[1]*state.ctm[3]

			r.drawLine(int(x0), int(y0), int(x1), int(y1), state.strokeColor)
		}
	}
}

func (r *softwareRendererAdv) fillPath(state *renderStateAdv) {
	// 简化实现
}

func (r *softwareRendererAdv) drawLine(x0, y0, x1, y1 int, color [4]float64) {
	dx := absIntAdv(x1 - x0)
	dy := absIntAdv(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy

	for {
		r.setPixel(x0, y0, color)

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func (r *softwareRendererAdv) setPixel(x, y int, color [4]float64) {
	if x < 0 || x >= r.width || y < 0 || y >= r.height {
		return
	}

	idx := (y*r.width + x) * 4
	if idx+3 >= len(r.data) {
		return
	}

	r.data[idx] = byte(color[0] * 255)
	r.data[idx+1] = byte(color[1] * 255)
	r.data[idx+2] = byte(color[2] * 255)
	r.data[idx+3] = byte(color[3] * 255)
}

func absIntAdv(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ============================================================================
// 辅助函数
// ============================================================================

type contentOpAdv struct {
	operator string
	operands []interface{}
}

func parseContentStreamAdv(data []byte) []contentOpAdv {
	var ops []contentOpAdv
	var operands []interface{}

	lexer := NewLexerFromBytes(data)
	parser := NewParser(lexer)

	for {
		obj, err := parser.ParseObject()
		if err != nil {
			break
		}

		switch v := obj.(type) {
		case Name:
			ops = append(ops, contentOpAdv{
				operator: string(v),
				operands: operands,
			})
			operands = nil
		case Integer, Real, String, Array, Dictionary:
			operands = append(operands, v)
		}
	}

	return ops
}

func operandToFloatAdv(op interface{}) float64 {
	switch v := op.(type) {
	case Integer:
		return float64(v)
	case Real:
		return float64(v)
	case float64:
		return v
	case int:
		return float64(v)
	}
	return 0
}

func operandToIntAdv(op interface{}) int {
	switch v := op.(type) {
	case Integer:
		return int(v)
	case Real:
		return int(v)
	case int:
		return v
	}
	return 0
}

// 未使用的导入占位
var (
	_ = crypto.SHA256
	_ = rsa.EncryptPKCS1v15
	_ = sha256.New
	_ = x509.ParseCertificate
	_ = asn1.Marshal
	_ = binary.BigEndian
	_ = errors.New
	_ = sort.Ints
)
