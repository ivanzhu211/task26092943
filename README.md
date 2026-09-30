# brandcheck

品牌配色审查库，由上层直接以 Go 包形式调用，无接口层。
待审图片为本地 PNG 文件（8 位 RGB 或 RGBA），品牌配置与审查报告由调用方在内存中持有。

## 用法

```go
cfg := brandcheck.Config{Colors: []brandcheck.BrandColor{
    {Hex: "#FF0000", TargetPercent: 60},
    {Hex: "#0000FF", TargetPercent: 40},
}}
rep, err := brandcheck.Review(cfg, "/path/to/image.png")
// rep.Score 为 0-100 整数颜色分；rep.NeedsReview 表示分数低于 70 需人工复核。
```

完整可运行示例见 `example_test.go`。

## 评分规则

- 图片与品牌色的每个 RGB 通道取原值除以 16 的整数部分（0-15）进行量化；
  色差为三个量化通道差值的平方和（0-675）。
- 像素面积权重等于其透明度（0-255），无透明通道时为 255；
  RGB 使用未乘透明度的原始值；全透明像素不参与评分。
- 全部权重在满足各品牌色目标占比（目标百分比 x 总权重，不取整）的约束下，
  以最小总色差分配（运输问题，最小费用流求解，允许拆分单个像素的权重）。
- 平均色差 / 675 为偏离比例，100 x (1 - 偏离比例) 四舍五入为整数分；
  低于 70 需人工复核；全图透明时 0 分且需复核。

## 测试

```sh
go test ./...
```
