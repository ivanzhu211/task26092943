// 调用示例：加载本地 PNG 图片，按品牌配置审查配色合规性。
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"brandcheck/brandaudit"
)

func main() {
	// 品牌配置由调用方在内存中持有：主色红、辅色蓝，各占 50%。
	cfg := brandaudit.Config{Colors: []brandaudit.BrandColor{
		{Color: brandaudit.Color{R: 0xFF, G: 0x00, B: 0x00}, Percent: 50},
		{Color: brandaudit.Color{R: 0x00, G: 0x00, B: 0xFF}, Percent: 50},
	}}

	path := "sample.png"
	if len(os.Args) > 1 {
		path = os.Args[1]
	} else if err := writeSample(path); err != nil {
		fmt.Fprintln(os.Stderr, "生成示例图片失败:", err)
		os.Exit(1)
	}

	report, err := brandaudit.ReviewFile(cfg, path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "审查失败:", err)
		os.Exit(1)
	}
	fmt.Printf("图片: %s\n颜色分: %d\n需要人工复核: %v\n", path, report.Score, report.NeedsReview)
}

// writeSample 生成一张左红右蓝的示例 PNG。
func writeSample(path string) error {
	img := image.NewNRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			c := color.NRGBA{R: 0xFF, A: 0xFF}
			if x >= 100 {
				c = color.NRGBA{B: 0xFF, A: 0xFF}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
