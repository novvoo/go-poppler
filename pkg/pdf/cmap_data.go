package pdf

// CMapRegistry provides access to predefined CMap data
type CMapRegistry struct {
	cmaps map[string][]byte
}

var globalCMapRegistry *CMapRegistry

// GetCMapRegistry returns the global CMap registry
func GetCMapRegistry() *CMapRegistry {
	if globalCMapRegistry == nil {
		globalCMapRegistry = &CMapRegistry{
			cmaps: make(map[string][]byte),
		}
		globalCMapRegistry.initBuiltinCMaps()
	}
	return globalCMapRegistry
}

// GetCMap retrieves a CMap by name
func (r *CMapRegistry) GetCMap(name string) []byte {
	return r.cmaps[name]
}

// initBuiltinCMaps initializes commonly used CMaps
func (r *CMapRegistry) initBuiltinCMaps() {
	// Adobe-GB1 (Simplified Chinese)
	r.cmaps["Adobe-GB1-UCS2"] = []byte(adobeGB1UCS2CMap)
	r.cmaps["GBK-EUC-H"] = []byte(gbkEUCHCMap)
	r.cmaps["GBpc-EUC-H"] = []byte(gbpcEUCHCMap)

	// Adobe-CNS1 (Traditional Chinese)
	r.cmaps["Adobe-CNS1-UCS2"] = []byte(adobeCNS1UCS2CMap)
	r.cmaps["B5pc-H"] = []byte(b5pcHCMap)

	// Adobe-Japan1 (Japanese)
	r.cmaps["Adobe-Japan1-UCS2"] = []byte(adobeJapan1UCS2CMap)
	r.cmaps["90ms-RKSJ-H"] = []byte(ms90RKSJHCMap)

	// Adobe-Korea1 (Korean)
	r.cmaps["Adobe-Korea1-UCS2"] = []byte(adobeKorea1UCS2CMap)
	r.cmaps["KSCms-UHC-H"] = []byte(kscmsUHCHCMap)

	// Identity
	r.cmaps["Identity-H"] = []byte(identityHCMap)
	r.cmaps["Identity-V"] = []byte(identityVCMap)
}

// Minimal CMap definitions for common encodings
// These are simplified versions - full CMaps are much larger

const adobeGB1UCS2CMap = `%!PS-Adobe-3.0 Resource-CMap
%%DocumentNeededResources: ProcSet (CIDInit)
%%IncludeResource: ProcSet (CIDInit)
%%BeginResource: CMap (Adobe-GB1-UCS2)
%%Title: (Adobe-GB1-UCS2 Adobe GB1 0)
%%Version: 1.0
%%Copyright: Copyright 1990-2009 Adobe Systems Incorporated.
/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CIDSystemInfo 3 dict dup begin
  /Registry (Adobe) def
  /Ordering (GB1) def
  /Supplement 0 def
end def
/CMapName /Adobe-GB1-UCS2 def
/CMapVersion 1.0 def
/CMapType 1 def
/WMode 0 def
1 begincodespacerange
  <0000> <FFFF>
endcodespacerange
endcmap
CMapName currentdict /CMap defineresource pop
end
end
%%EndResource
%%EOF
`

const gbkEUCHCMap = `%!PS-Adobe-3.0 Resource-CMap
%%BeginResource: CMap (GBK-EUC-H)
/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CIDSystemInfo 3 dict dup begin
  /Registry (Adobe) def
  /Ordering (GB1) def
  /Supplement 0 def
end def
/CMapName /GBK-EUC-H def
/CMapType 1 def
/WMode 0 def
1 begincodespacerange
  <00> <FF>
endcodespacerange
endcmap
CMapName currentdict /CMap defineresource pop
end
end
%%EndResource
%%EOF
`

const gbpcEUCHCMap = gbkEUCHCMap // Simplified

const adobeCNS1UCS2CMap = `%!PS-Adobe-3.0 Resource-CMap
%%BeginResource: CMap (Adobe-CNS1-UCS2)
/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CIDSystemInfo 3 dict dup begin
  /Registry (Adobe) def
  /Ordering (CNS1) def
  /Supplement 0 def
end def
/CMapName /Adobe-CNS1-UCS2 def
/CMapType 1 def
/WMode 0 def
1 begincodespacerange
  <0000> <FFFF>
endcodespacerange
endcmap
CMapName currentdict /CMap defineresource pop
end
end
%%EndResource
%%EOF
`

const b5pcHCMap = adobeCNS1UCS2CMap // Simplified

const adobeJapan1UCS2CMap = `%!PS-Adobe-3.0 Resource-CMap
%%BeginResource: CMap (Adobe-Japan1-UCS2)
/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CIDSystemInfo 3 dict dup begin
  /Registry (Adobe) def
  /Ordering (Japan1) def
  /Supplement 0 def
end def
/CMapName /Adobe-Japan1-UCS2 def
/CMapType 1 def
/WMode 0 def
1 begincodespacerange
  <0000> <FFFF>
endcodespacerange
endcmap
CMapName currentdict /CMap defineresource pop
end
end
%%EndResource
%%EOF
`

const ms90RKSJHCMap = adobeJapan1UCS2CMap // Simplified

const adobeKorea1UCS2CMap = `%!PS-Adobe-3.0 Resource-CMap
%%BeginResource: CMap (Adobe-Korea1-UCS2)
/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CIDSystemInfo 3 dict dup begin
  /Registry (Adobe) def
  /Ordering (Korea1) def
  /Supplement 0 def
end def
/CMapName /Adobe-Korea1-UCS2 def
/CMapType 1 def
/WMode 0 def
1 begincodespacerange
  <0000> <FFFF>
endcodespacerange
endcmap
CMapName currentdict /CMap defineresource pop
end
end
%%EndResource
%%EOF
`

const kscmsUHCHCMap = adobeKorea1UCS2CMap // Simplified

const identityHCMap = `%!PS-Adobe-3.0 Resource-CMap
%%BeginResource: CMap (Identity-H)
/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CIDSystemInfo 3 dict dup begin
  /Registry (Adobe) def
  /Ordering (Identity) def
  /Supplement 0 def
end def
/CMapName /Identity-H def
/CMapType 1 def
/WMode 0 def
1 begincodespacerange
  <0000> <FFFF>
endcodespacerange
endcmap
CMapName currentdict /CMap defineresource pop
end
end
%%EndResource
%%EOF
`

const identityVCMap = `%!PS-Adobe-3.0 Resource-CMap
%%BeginResource: CMap (Identity-V)
/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CIDSystemInfo 3 dict dup begin
  /Registry (Adobe) def
  /Ordering (Identity) def
  /Supplement 0 def
end def
/CMapName /Identity-V def
/CMapType 1 def
/WMode 1 def
1 begincodespacerange
  <0000> <FFFF>
endcodespacerange
endcmap
CMapName currentdict /CMap defineresource pop
end
end
%%EndResource
%%EOF
`
