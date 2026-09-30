package brandcheck

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"math"
	"os"
	"strconv"
	"strings"
)

// maxChannelDiff 为量化后三通道差值平方和的最大值：3 * 15 * 15。
const maxChannelDiff = 675

// reviewThreshold 为需要人工复核的分数线（低于该值需要复核）。
const reviewThreshold = 70

// Review 审查本地 PNG 图片是否符合品牌配色配置，返回评分报告。
// 图片须为 8 位 RGB 或 RGBA 的 PNG；配置与报告均由调用方在内存中持有。
func Review(cfg Config, pngPath string) (Report, error) {
	brand, err := parseConfig(cfg)
	if err != nil {
		return Report{}, err
	}

	f, err := os.Open(pngPath)
	if err != nil {
		return Report{}, fmt.Errorf("brandcheck: 打开图片失败: %w", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return Report{}, fmt.Errorf("brandcheck: 解码 PNG 失败: %w", err)
	}

	// 按量化后的颜色聚合面积权重，供应点数量最多 16^3。
	type qcolor struct{ r, g, b uint8 }
	weights := make(map[qcolor]uint64)
	var totalWeight uint64
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if c.A == 0 {
				continue
			}
			weights[qcolor{c.R / 16, c.G / 16, c.B / 16}] += uint64(c.A)
			totalWeight += uint64(c.A)
		}
	}
	if totalWeight == 0 {
		return Report{Score: 0, NeedsReview: true}, nil
	}

	// 权重放大 100 倍，使“目标百分比 x 总权重”仍为整数，避免取整丢失占比。
	supplies := make([]uint64, 0, len(weights))
	keys := make([]qcolor, 0, len(weights))
	for k, w := range weights {
		keys = append(keys, k)
		supplies = append(supplies, w*100)
	}
	demands := make([]uint64, len(brand))
	for i, b := range brand {
		demands[i] = uint64(b.TargetPercent) * totalWeight
	}

	cost := make([][]uint64, len(keys))
	for i, k := range keys {
		cost[i] = make([]uint64, len(brand))
		for j, b := range brand {
			dr := int(k.r) - int(b.r)
			dg := int(k.g) - int(b.g)
			db := int(k.b) - int(b.b)
			cost[i][j] = uint64(dr*dr + dg*dg + db*db)
		}
	}

	minCost := minTransportCost(supplies, demands, cost)
	meanDiff := float64(minCost) / (100 * float64(totalWeight))
	deviation := meanDiff / maxChannelDiff
	score := int(math.Round(100 * (1 - deviation)))

	return Report{
		Score:       score,
		NeedsReview: score < reviewThreshold,
		TotalWeight: totalWeight,
		MeanDiff:    meanDiff,
	}, nil
}

type quantBrand struct {
	r, g, b uint8
	BrandColor
}

func parseConfig(cfg Config) ([]quantBrand, error) {
	if len(cfg.Colors) == 0 {
		return nil, errors.New("brandcheck: 品牌配置不能为空")
	}
	out := make([]quantBrand, len(cfg.Colors))
	sum := 0
	for i, c := range cfg.Colors {
		if c.TargetPercent <= 0 {
			return nil, fmt.Errorf("brandcheck: 颜色 %s 的目标百分比必须为正整数", c.Hex)
		}
		sum += c.TargetPercent
		r, g, b, err := parseHex(c.Hex)
		if err != nil {
			return nil, err
		}
		out[i] = quantBrand{r: r / 16, g: g / 16, b: b / 16, BrandColor: c}
	}
	if sum != 100 {
		return nil, fmt.Errorf("brandcheck: 目标百分比合计必须为 100，实际为 %d", sum)
	}
	return out, nil
}

func parseHex(s string) (r, g, b uint8, err error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return 0, 0, 0, fmt.Errorf("brandcheck: 非法颜色 %q，须为 #RRGGBB", s)
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("brandcheck: 非法颜色 %q，须为 #RRGGBB", s)
	}
	return uint8(v >> 16), uint8(v >> 8), uint8(v), nil
}
