// Package brandaudit 提供品牌配色合规审查能力：
// 根据品牌配置中的颜色与目标面积占比，对 PNG 图片计算颜色合规评分。
package brandaudit

// Color 表示一个 8 位 RGB 颜色。
type Color struct {
	R uint8
	G uint8
	B uint8
}

// BrandColor 表示品牌配置中的一种颜色及其目标面积百分比。
type BrandColor struct {
	Color Color
	// Percent 为目标面积百分比，正整数，同一份配置内合计必须为 100。
	Percent int
}

// Config 表示一份品牌配色配置。
type Config struct {
	Colors []BrandColor
}

// Report 表示一次配色审查的结果。
type Report struct {
	// Score 为 0 至 100 的整数颜色分。
	Score int
	// NeedsReview 为 true 表示分数低于 70（或全图透明），需要人工复核。
	NeedsReview bool
}
