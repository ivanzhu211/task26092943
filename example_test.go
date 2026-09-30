package brandcheck_test

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"brandcheck"
)

// 调用示例：品牌配置与报告均在内存中持有，图片为本地 PNG 文件。
func Example() {
	// 准备一张待审图片（实际使用中为已有的本地 PNG 文件）。
	path := filepath.Join(os.TempDir(), "brandcheck-example.png")
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255}) // 纯红
	img.SetNRGBA(1, 0, color.NRGBA{B: 255, A: 128}) // 半透明蓝
	f, _ := os.Create(path)
	png.Encode(f, img)
	f.Close()
	defer os.Remove(path)

	cfg := brandcheck.Config{Colors: []brandcheck.BrandColor{
		{Hex: "#FF0000", TargetPercent: 60},
		{Hex: "#0000FF", TargetPercent: 40},
	}}
	rep, err := brandcheck.Review(cfg, path)
	if err != nil {
		panic(err)
	}
	fmt.Printf("颜色分: %d, 需要人工复核: %v\n", rep.Score, rep.NeedsReview)
	// Output:
	// 颜色分: 96, 需要人工复核: false
}
