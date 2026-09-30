package brandcheck

// BrandColor 是一个品牌色及其目标面积占比。
type BrandColor struct {
	// Hex 为 "#RRGGBB" 形式的十六进制 RGB 颜色。
	Hex string
	// TargetPercent 为目标面积占比（正整数），一份配置中全部颜色合计必须为 100。
	TargetPercent int
}

// Config 为一份品牌配色配置。
type Config struct {
	Colors []BrandColor
}

// Report 为一次配色审查的结果，由调用方在内存中持有。
type Report struct {
	// Score 为 0 至 100 的整数颜色分。
	Score int
	// NeedsReview 为 true 表示分数低于 70，需要人工复核。
	NeedsReview bool
	// TotalWeight 为全图面积权重之和（各像素透明度之和）。
	TotalWeight uint64
	// MeanDiff 为最小总色差除以全图总权重得到的平均色差。
	MeanDiff float64
}
