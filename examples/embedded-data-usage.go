package main

import (
	"fmt"
	"log"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

// 示例：如何使用嵌入的 poppler-data
func main() {
	fmt.Println("=== 嵌入式 Poppler Data 使用示例 ===\n")

	// 示例 1: 读取 CID to Unicode 映射
	fmt.Println("1. 读取 CID to Unicode 映射（简体中文）")
	gbData, err := pdf.FindCIDToUnicodeFile("Adobe-GB1")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ 成功读取 Adobe-GB1: %d 字节\n\n", len(gbData))

	// 示例 2: 读取 CMap 文件
	fmt.Println("2. 读取 CMap 文件")
	cmapData, err := pdf.FindCMapFile("GBK-EUC-H")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ 成功读取 GBK-EUC-H CMap: %d 字节\n\n", len(cmapData))

	// 示例 3: 列出所有 CID to Unicode 文件
	fmt.Println("3. 列出所有 CID to Unicode 文件")
	cidFiles, err := pdf.ListPopplerDataFiles("cidToUnicode")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   找到 %d 个文件:\n", len(cidFiles))
	for _, file := range cidFiles {
		fmt.Printf("   - %s\n", file)
	}
	fmt.Println()

	// 示例 4: 读取 Unicode Map
	fmt.Println("4. 读取 Unicode Map")
	unicodeMap, err := pdf.FindUnicodeMapFile("GBK")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ 成功读取 GBK Unicode Map: %d 字节\n\n", len(unicodeMap))

	// 示例 5: 读取 Name to Unicode 映射
	fmt.Println("5. 读取 Name to Unicode 映射")
	greekMap, err := pdf.FindNameToUnicodeFile("Greek")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ 成功读取 Greek Name to Unicode: %d 字节\n\n", len(greekMap))

	// 示例 6: 直接访问文件系统
	fmt.Println("6. 直接访问嵌入的文件系统")
	fsys := pdf.GetPopplerDataFS()
	if fsys != nil {
		fmt.Println("   ✓ 文件系统可用")

		// 可以使用标准的 io/fs 接口
		data, err := pdf.ReadPopplerDataFile("cidToUnicode/Adobe-Japan1")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("   ✓ 直接读取 Adobe-Japan1: %d 字节\n", len(data))
	}

	fmt.Println("\n=== 所有示例执行成功 ===")
	fmt.Println("\n提示：这些数据都已嵌入到二进制文件中，")
	fmt.Println("无需任何外部数据文件即可运行！")
}
