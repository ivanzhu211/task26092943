# task26092943

品牌配色审查库（Go），由上层直接调用，不做接口层。

## 结构

- `brandaudit/`：库本体
  - `types.go`：品牌配置（`Config`/`BrandColor`/`Color`）与审查报告（`Report`）类型
  - `review.go`：公开入口 `Review`（内存图片）与 `ReviewFile`（本地 PNG 文件）
  - `flow.go`：最小费用流，求解满足各品牌色目标占比的最小总色差权重分配
- `example/`：调用示例，`go run ./example [png路径]`

## 规则摘要

- 颜色按通道量化为原值除以 16 的整数部分（0~15），色差为三通道量化差值平方和（0~675）
- 像素面积权重等于透明度（0~255），无透明通道按 255，全透明像素不参与
- 全部权重按品牌色目标百分比精确分配（权重放大 100 倍保持整数，不取整），
  取满足占比约束的最小总色差计算评分
- 评分 = round(100 × (1 − 平均色差/675))；低于 70 需人工复核；全图透明为 0 分并需复核

## 使用

```go
cfg := brandaudit.Config{Colors: []brandaudit.BrandColor{
    {Color: brandaudit.Color{R: 0xFF, G: 0x00, B: 0x00}, Percent: 50},
    {Color: brandaudit.Color{R: 0x00, G: 0x00, B: 0xFF}, Percent: 50},
}}
report, err := brandaudit.ReviewFile(cfg, "banner.png")
```

## 测试

```sh
go test ./...
```
