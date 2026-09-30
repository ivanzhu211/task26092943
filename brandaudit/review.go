package brandaudit

import (
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"os"
)

// reviewThreshold 为人工复核分数线，低于该分数需要复核。
const reviewThreshold = 70

// maxColorDistance 为量化后两种颜色的最大色差（15^2 * 3）。
const maxColorDistance = 675

// Review 对图片执行品牌配色审查，返回审查报告。
// 图片须为 8 位 RGB 或 RGBA；其他颜色模型返回错误。
func Review(cfg Config, img image.Image) (Report, error) {
	weights, err := collectWeights(img)
	if err != nil {
		return Report{}, err
	}
	return score(cfg, weights)
}

// ReviewFile 从本地 PNG 文件读取图片并执行品牌配色审查。
func ReviewFile(cfg Config, path string) (Report, error) {
	f, err := os.Open(path)
	if err != nil {
		return Report{}, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return Report{}, fmt.Errorf("brandaudit: 解码 PNG 失败: %w", err)
	}
	return Review(cfg, img)
}

// quantKey 为量化后的颜色，三个通道均为 0 至 15。
type quantKey struct {
	r, g, b uint8
}

func quantize(v uint8) uint8 { return v / 16 }

// collectWeights 统计图片中各量化颜色的面积权重（透明度值之和）。
// 透明度为 0 的像素不参与评分。
func collectWeights(img image.Image) (map[quantKey]int64, error) {
	weights := make(map[quantKey]int64)
	switch src := img.(type) {
	case *image.RGBA:
		b := src.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				i := src.PixOffset(x, y)
				a := src.Pix[i+3]
				if a == 0 {
					continue
				}
				k := quantKey{quantize(src.Pix[i]), quantize(src.Pix[i+1]), quantize(src.Pix[i+2])}
				weights[k] += int64(a)
			}
		}
	case *image.NRGBA:
		b := src.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				i := src.PixOffset(x, y)
				a := src.Pix[i+3]
				if a == 0 {
					continue
				}
				k := quantKey{quantize(src.Pix[i]), quantize(src.Pix[i+1]), quantize(src.Pix[i+2])}
				weights[k] += int64(a)
			}
		}
	default:
		return nil, fmt.Errorf("brandaudit: 不支持的图片类型 %T，仅支持 8 位 RGB/RGBA 的 PNG", img)
	}
	return weights, nil
}

func validate(cfg Config) error {
	if len(cfg.Colors) == 0 {
		return errors.New("brandaudit: 品牌配置至少包含一种颜色")
	}
	total := 0
	for i, c := range cfg.Colors {
		if c.Percent <= 0 {
			return fmt.Errorf("brandaudit: 第 %d 个品牌色的目标百分比必须为正整数", i)
		}
		total += c.Percent
	}
	if total != 100 {
		return fmt.Errorf("brandaudit: 目标百分比合计必须为 100，实际为 %d", total)
	}
	return nil
}

// score 根据量化颜色权重与品牌配置计算审查报告。
func score(cfg Config, weights map[quantKey]int64) (Report, error) {
	if err := validate(cfg); err != nil {
		return Report{}, err
	}
	var totalWeight int64
	for _, w := range weights {
		totalWeight += w
	}
	if totalWeight == 0 {
		return Report{Score: 0, NeedsReview: true}, nil
	}

	keys := make([]quantKey, 0, len(weights))
	for k := range weights {
		keys = append(keys, k)
	}
	// 面积权重放大 100 倍，使“目标百分比乘以总权重”保持整数，
	// 既不取整丢失精度，也能区分量化后相同的品牌色。
	supply := make([]int64, len(keys))
	for i, k := range keys {
		supply[i] = weights[k] * 100
	}
	demands := make([]int64, len(cfg.Colors))
	for j, c := range cfg.Colors {
		demands[j] = int64(c.Percent) * totalWeight
	}
	cost := make([][]int64, len(keys))
	for i, k := range keys {
		cost[i] = make([]int64, len(cfg.Colors))
		for j, c := range cfg.Colors {
			dr := int64(k.r) - int64(quantize(c.Color.R))
			dg := int64(k.g) - int64(quantize(c.Color.G))
			db := int64(k.b) - int64(quantize(c.Color.B))
			cost[i][j] = dr*dr + dg*dg + db*db
		}
	}

	scaledCost := minTotalCost(supply, demands, cost)
	// 平均色差 = scaledCost / (100 * totalWeight)，
	// 分数 = round(100 * (1 - 平均色差 / 675))，用整数运算完成四舍五入。
	num := 100*maxColorDistance*totalWeight - scaledCost
	den := maxColorDistance * totalWeight
	score := int((2*num + den) / (2 * den))
	return Report{Score: score, NeedsReview: score < reviewThreshold}, nil
}
