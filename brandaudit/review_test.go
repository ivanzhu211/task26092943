package brandaudit

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

var redBlueConfig = Config{Colors: []BrandColor{
	{Color: Color{R: 0xFF, G: 0x00, B: 0x00}, Percent: 50},
	{Color: Color{R: 0x00, G: 0x00, B: 0xFF}, Percent: 50},
}}

// newRedRGBA 生成 w*h 的全红不透明 RGBA 图片。
func newRedRGBA(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i] = 0xFF
		img.Pix[i+3] = 0xFF
	}
	return img
}

// newImage 按行优先顺序把像素填入 w*h 的 NRGBA 图片。
func newImage(w, h int, pixels []color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i, p := range pixels {
		img.SetNRGBA(i%w, i/w, p)
	}
	return img
}

func repeat(p color.NRGBA, n int) []color.NRGBA {
	pixels := make([]color.NRGBA, n)
	for i := range pixels {
		pixels[i] = p
	}
	return pixels
}

// 品牌红蓝各占 50%，全红不透明图片：一半权重被迫按色差 450 分配给蓝色，
// 平均色差 225，得分 round(100*(1-225/675)) = 67，需要人工复核。
func TestAllRedAgainstRedBlue(t *testing.T) {
	report, err := Review(redBlueConfig, newRedRGBA(4, 4))
	if err != nil {
		t.Fatalf("Review 出错: %v", err)
	}
	if report.Score != 67 {
		t.Errorf("Score = %d, 期望 67", report.Score)
	}
	if !report.NeedsReview {
		t.Error("NeedsReview = false, 期望 true")
	}
}

// 70 分不需要人工复核：白色图片对白色(50%)与量化色 (9,3,0)(50%)，
// 一半权重色差为 6^2+12^2+15^2 = 405，平均色差 202.5，得分恰为 70。
func TestScoreSeventyNeedsNoReview(t *testing.T) {
	cfg := Config{Colors: []BrandColor{
		{Color: Color{R: 0xFF, G: 0xFF, B: 0xFF}, Percent: 50},
		{Color: Color{R: 148, G: 50, B: 7}, Percent: 50}, // 量化后为 (9,3,0)
	}}
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for i := range img.Pix {
		img.Pix[i] = 0xFF
	}
	report, err := Review(cfg, img)
	if err != nil {
		t.Fatalf("Review 出错: %v", err)
	}
	if report.Score != 70 {
		t.Errorf("Score = %d, 期望 70", report.Score)
	}
	if report.NeedsReview {
		t.Error("NeedsReview = true, 期望 false")
	}
}

// 半透明微型图片：红(a=255)、绿(a=128)、黄(a=100)，品牌红绿各占 50%。
// 总权重 483，每个品牌色须恰好分到 241.5，必须拆分红色像素的权重。
// 独立核算：红色块分 241.5 给红、13.5 给绿(单位代价 450)，
// 绿块全给绿，黄块对红绿色差均为 225，总代价 = 13.5*450 + 100*225 = 28575。
// 平均色差 28575/483，得分 round(100*(1-28575/483/675)) = 91。
func TestSemiTransparentSplitting(t *testing.T) {
	cfg := Config{Colors: []BrandColor{
		{Color: Color{R: 0xFF, G: 0x00, B: 0x00}, Percent: 50},
		{Color: Color{R: 0x00, G: 0xFF, B: 0x00}, Percent: 50},
	}}
	red := color.NRGBA{R: 0xFF, A: 0xFF}
	green := color.NRGBA{G: 0xFF, A: 128}
	yellow := color.NRGBA{R: 0xFF, G: 0xFF, A: 100}
	transparent := color.NRGBA{}

	// 独立核算：直接枚举拆分点求全局最优（需求为 0.5 的整数倍，
	// 最优解的拆分点也落在 0.5 的整数倍上），与库的实现互不相同。
	bruteForceScore := func() int {
		const totalWeight = 255 + 128 + 100
		best := int64(1) << 62
		// x2, y2 为红色块、绿色块分给品牌红的权重的两倍。
		for x2 := int64(0); x2 <= 2*255; x2++ {
			for y2 := int64(0); y2 <= 2*128; y2++ {
				z2 := totalWeight - x2 - y2 // 黄色块分给品牌红的权重的两倍
				if z2 < 0 || z2 > 2*100 {
					continue
				}
				cost2 := 450*(2*255-x2) + 450*y2 + 225*2*100
				if cost2 < best {
					best = cost2
				}
			}
		}
		// best 为两倍权重下的总代价，平均色差 = best/2/483。
		num := int64(100*675*2*totalWeight) - 100*best
		den := int64(675 * 2 * totalWeight)
		return int((2*num + den) / (2 * den))
	}()
	if bruteForceScore != 91 {
		t.Fatalf("独立核算得分 = %d, 期望 91（测试本身有误）", bruteForceScore)
	}

	base := []color.NRGBA{red, green, yellow}
	want, err := Review(cfg, newImage(3, 1, base))
	if err != nil {
		t.Fatalf("Review 出错: %v", err)
	}
	if want.Score != bruteForceScore {
		t.Errorf("Score = %d, 独立核算全局最优为 %d", want.Score, bruteForceScore)
	}
	if want.NeedsReview {
		t.Error("NeedsReview = true, 期望 false")
	}

	// 改变像素排列、改变图片形状、添加全透明像素，结果必须不变。
	variants := map[string][]color.NRGBA{
		"逆序":    {yellow, green, red},
		"加透明像素": {red, transparent, green, transparent, yellow, transparent},
	}
	for name, pixels := range variants {
		got, err := Review(cfg, newImage(len(pixels), 1, pixels))
		if err != nil {
			t.Fatalf("%s: Review 出错: %v", name, err)
		}
		if got != want {
			t.Errorf("%s: 结果 %+v, 期望 %+v", name, got, want)
		}
	}
	// 换个形状再验证一次。
	got, err := Review(cfg, newImage(1, 3, base))
	if err != nil {
		t.Fatalf("Review 出错: %v", err)
	}
	if got != want {
		t.Errorf("改变形状: 结果 %+v, 期望 %+v", got, want)
	}
}

// 量化后相同的两个品牌色不能合并丢失占比：两个品牌色量化后都是 (15,0,0)，
// 全红图片应得 100 分。
func TestDuplicateQuantizedBrandColors(t *testing.T) {
	cfg := Config{Colors: []BrandColor{
		{Color: Color{R: 0xFF}, Percent: 50},
		{Color: Color{R: 0xF1}, Percent: 50}, // 0xF1/16 == 0xFF/16 == 15
	}}
	report, err := Review(cfg, newImage(2, 2, repeat(color.NRGBA{R: 0xFF, A: 0xFF}, 4)))
	if err != nil {
		t.Fatalf("Review 出错: %v", err)
	}
	if report.Score != 100 || report.NeedsReview {
		t.Errorf("结果 %+v, 期望 Score=100 且无需复核", report)
	}
}

// 全图透明：颜色分为 0 并需要复核。
func TestFullyTransparent(t *testing.T) {
	report, err := Review(redBlueConfig, newImage(2, 2, repeat(color.NRGBA{}, 4)))
	if err != nil {
		t.Fatalf("Review 出错: %v", err)
	}
	if report.Score != 0 || !report.NeedsReview {
		t.Errorf("结果 %+v, 期望 Score=0 且需要复核", report)
	}
}

func TestInvalidConfig(t *testing.T) {
	img := newImage(1, 1, []color.NRGBA{{R: 0xFF, A: 0xFF}})
	cases := map[string]Config{
		"空配置":    {},
		"合计非100": {Colors: []BrandColor{{Color: Color{R: 0xFF}, Percent: 60}}},
		"零百分比": {Colors: []BrandColor{
			{Color: Color{R: 0xFF}, Percent: 100},
			{Color: Color{B: 0xFF}, Percent: 0},
		}},
	}
	for name, cfg := range cases {
		if _, err := Review(cfg, img); err == nil {
			t.Errorf("%s: 期望返回错误", name)
		}
	}
}

func TestUnsupportedImageType(t *testing.T) {
	if _, err := Review(redBlueConfig, image.NewGray(image.Rect(0, 0, 1, 1))); err == nil {
		t.Error("灰度图: 期望返回错误")
	}
}

func TestReviewFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "red.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, newRedRGBA(4, 4)); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	report, err := ReviewFile(redBlueConfig, path)
	if err != nil {
		t.Fatalf("ReviewFile 出错: %v", err)
	}
	if report.Score != 67 || !report.NeedsReview {
		t.Errorf("结果 %+v, 期望 Score=67 且需要复核", report)
	}
	if _, err := ReviewFile(redBlueConfig, filepath.Join(t.TempDir(), "missing.png")); err == nil {
		t.Error("文件不存在: 期望返回错误")
	}
}
