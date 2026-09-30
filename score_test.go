package brandcheck

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// writePNG 把像素按行优先写入 PNG 文件。opaque 为 true 时强制 alpha=255（编码为 RGB PNG）。
func writePNG(t *testing.T, path string, width, height int, pixels []color.NRGBA, opaque bool) {
	t.Helper()
	if len(pixels) != width*height {
		t.Fatalf("像素数量 %d 与尺寸 %dx%d 不符", len(pixels), width, height)
	}
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for i, p := range pixels {
		if opaque {
			p.A = 255
		}
		img.SetNRGBA(i%width, i/width, p)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestRedBlueBrandAgainstAllRedImage(t *testing.T) {
	cfg := Config{Colors: []BrandColor{
		{Hex: "#FF0000", TargetPercent: 50},
		{Hex: "#0000FF", TargetPercent: 50},
	}}
	red := color.NRGBA{R: 255, A: 255}
	pixels := []color.NRGBA{red, red, red, red}
	path := filepath.Join(t.TempDir(), "red.png")
	writePNG(t, path, 2, 2, pixels, true)

	rep, err := Review(cfg, path)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Score != 67 {
		t.Fatalf("期望 67 分，得到 %d", rep.Score)
	}
	if !rep.NeedsReview {
		t.Fatal("67 分应需要人工复核")
	}
}

// TestSplitWeightsMatchesIndependentOptimum 构造必须拆分像素权重才能满足占比的微型图片，
// 并与独立核算的全局最优值对比。
//
// 品牌色：黑 #000000（50%）、白 #FFFFFF（50%）。
// 像素：P1 黑 不透明(255)，P2 白 半透明(128)，P3 灰 #808080 不透明(255)。
// 总权重 638，两色各需 319。
// 最优分配：P1->黑 255，P2->白 128，P3 拆成 64->黑、191->白。
// 最小总色差 = 64*192 + 191*147 = 40365。
func TestSplitWeightsMatchesIndependentOptimum(t *testing.T) {
	cfg := Config{Colors: []BrandColor{
		{Hex: "#000000", TargetPercent: 50},
		{Hex: "#FFFFFF", TargetPercent: 50},
	}}
	p1 := color.NRGBA{A: 255}                         // 黑
	p2 := color.NRGBA{R: 255, G: 255, B: 255, A: 128} // 半透明白
	p3 := color.NRGBA{R: 128, G: 128, B: 128, A: 255} // 灰
	transparent := color.NRGBA{R: 200, A: 0}          // 全透明，不参与评分

	const expectedCost = 64*192 + 191*147
	const totalWeight = 255 + 128 + 255
	expectedMean := float64(expectedCost) / totalWeight
	expectedScore := int(math.Round(100 * (1 - expectedMean/675)))

	base := []color.NRGBA{p1, p2, p3}
	variants := map[string][]color.NRGBA{
		"原始排列":   {p1, p2, p3},
		"改变排列":   {p3, p1, p2},
		"含全透明像素": {p1, transparent, p2, transparent, p3},
	}
	for name, pixels := range variants {
		t.Run(name, func(t *testing.T) {
			width := len(pixels)
			path := filepath.Join(t.TempDir(), "micro.png")
			writePNG(t, path, width, 1, pixels, false)

			rep, err := Review(cfg, path)
			if err != nil {
				t.Fatal(err)
			}
			if rep.TotalWeight != totalWeight {
				t.Fatalf("期望总权重 %d，得到 %d", totalWeight, rep.TotalWeight)
			}
			if math.Abs(rep.MeanDiff-expectedMean) > 1e-9 {
				t.Fatalf("期望平均色差 %v，得到 %v", expectedMean, rep.MeanDiff)
			}
			if rep.Score != expectedScore {
				t.Fatalf("期望 %d 分，得到 %d", expectedScore, rep.Score)
			}
			if rep.NeedsReview != (expectedScore < 70) {
				t.Fatalf("复核标记与分数 %d 不一致", rep.Score)
			}
			_ = base
		})
	}
}

func TestAllTransparentImage(t *testing.T) {
	cfg := Config{Colors: []BrandColor{{Hex: "#FF0000", TargetPercent: 100}}}
	path := filepath.Join(t.TempDir(), "transparent.png")
	writePNG(t, path, 2, 1, []color.NRGBA{{A: 0}, {A: 0}}, false)

	rep, err := Review(cfg, path)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Score != 0 || !rep.NeedsReview {
		t.Fatalf("全透明图应为 0 分且需复核，得到 %+v", rep)
	}
}

// TestDuplicateQuantizedColorsKeepPercents 两个品牌色量化后相同，占比仍须各自保留。
func TestDuplicateQuantizedColorsKeepPercents(t *testing.T) {
	cfg := Config{Colors: []BrandColor{
		{Hex: "#FF0000", TargetPercent: 50},
		{Hex: "#F00000", TargetPercent: 50}, // 量化后与 #FF0000 相同
	}}
	red := color.NRGBA{R: 255, A: 255}
	path := filepath.Join(t.TempDir(), "red.png")
	writePNG(t, path, 2, 1, []color.NRGBA{red, red}, true)

	rep, err := Review(cfg, path)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Score != 100 || rep.NeedsReview {
		t.Fatalf("期望 100 分且无需复核，得到 %+v", rep)
	}
}

func TestInvalidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "red.png")
	writePNG(t, path, 1, 1, []color.NRGBA{{R: 255, A: 255}}, true)

	cases := map[string]Config{
		"空配置":    {},
		"合计非100": {Colors: []BrandColor{{Hex: "#FF0000", TargetPercent: 60}}},
		"非正百分比":  {Colors: []BrandColor{{Hex: "#FF0000", TargetPercent: 0}, {Hex: "#00FF00", TargetPercent: 100}}},
		"非法颜色":   {Colors: []BrandColor{{Hex: "red", TargetPercent: 100}}},
	}
	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Review(cfg, path); err == nil {
				t.Fatal("期望返回错误")
			}
		})
	}
}
