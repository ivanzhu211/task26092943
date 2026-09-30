package brandcheck

import "container/heap"

// minTransportCost 求解运输问题：supplies 与 demands 为同量纲的整数权重，
// cost[i][j] 为把一单位权重从供应点 i 分配给需求点 j 的代价。
// 返回满足全部需求的最小总代价。要求 sum(supplies) == sum(demands)。
//
// 实现为最小费用最大流（SSP + Johnson 势），单位代价非负，
// 图上结构为 源->供应点->需求点->汇点。
func minTransportCost(supplies, demands []uint64, cost [][]uint64) uint64 {
	n := len(supplies)
	m := len(demands)
	if n == 0 || m == 0 {
		return 0
	}

	// 节点编号：0=源，1..n=供应点，n+1..n+m=需求点，n+m+1=汇。
	src := 0
	sink := n + m + 1
	g := newFlowGraph(sink + 1)

	total := int64(0)
	for i, s := range supplies {
		g.addEdge(src, 1+i, int64(s), 0)
		total += int64(s)
	}
	for j, d := range demands {
		g.addEdge(n+1+j, sink, int64(d), 0)
	}
	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			g.addEdge(1+i, n+1+j, total, int64(cost[i][j]))
		}
	}

	return uint64(g.minCostFlow(src, sink, total))
}

type flowEdge struct {
	to, rev int
	cap     int64
	cost    int64
}

type flowGraph struct {
	g [][]flowEdge
}

func newFlowGraph(n int) *flowGraph {
	return &flowGraph{g: make([][]flowEdge, n)}
}

func (f *flowGraph) addEdge(from, to int, cap, cost int64) {
	f.g[from] = append(f.g[from], flowEdge{to: to, rev: len(f.g[to]), cap: cap, cost: cost})
	f.g[to] = append(f.g[to], flowEdge{to: from, rev: len(f.g[from]) - 1, cap: 0, cost: -cost})
}

type pqItem struct {
	node int
	dist int64
}

type pq []pqItem

func (p pq) Len() int            { return len(p) }
func (p pq) Less(i, j int) bool  { return p[i].dist < p[j].dist }
func (p pq) Swap(i, j int)       { p[i], p[j] = p[j], p[i] }
func (p *pq) Push(x interface{}) { *p = append(*p, x.(pqItem)) }
func (p *pq) Pop() interface{} {
	old := *p
	item := old[len(old)-1]
	*p = old[:len(old)-1]
	return item
}

const inf = int64(1) << 62

// minCostFlow 沿最短路反复增广，直到送出 amount 单位流量，返回最小总费用。
func (f *flowGraph) minCostFlow(src, sink int, amount int64) int64 {
	n := len(f.g)
	pot := make([]int64, n)
	prevV := make([]int, n)
	prevE := make([]int, n)
	dist := make([]int64, n)

	var totalCost int64
	flow := int64(0)
	for flow < amount {
		for i := range dist {
			dist[i] = inf
		}
		dist[src] = 0
		queue := &pq{{node: src, dist: 0}}
		for queue.Len() > 0 {
			item := heap.Pop(queue).(pqItem)
			v := item.node
			if dist[v] < item.dist {
				continue
			}
			for ei, e := range f.g[v] {
				if e.cap <= 0 {
					continue
				}
				nd := dist[v] + e.cost + pot[v] - pot[e.to]
				if dist[e.to] > nd {
					dist[e.to] = nd
					prevV[e.to] = v
					prevE[e.to] = ei
					heap.Push(queue, pqItem{node: e.to, dist: nd})
				}
			}
		}
		if dist[sink] == inf {
			break
		}
		for v := 0; v < n; v++ {
			if dist[v] < inf {
				pot[v] += dist[v]
			}
		}
		d := amount - flow
		for v := sink; v != src; v = prevV[v] {
			if c := f.g[prevV[v]][prevE[v]].cap; c < d {
				d = c
			}
		}
		flow += d
		totalCost += d * pot[sink]
		for v := sink; v != src; v = prevV[v] {
			e := &f.g[prevV[v]][prevE[v]]
			e.cap -= d
			f.g[v][e.rev].cap += d
		}
	}
	return totalCost
}
