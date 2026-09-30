package brandaudit

import "container/heap"

// minTotalCost 求解运输问题：把第 i 种色块的 weights[i] 权重分配给品牌色，
// 第 j 个品牌色必须恰好收到 demands[j] 的权重，单位代价为 cost[i][j]。
// 返回满足需求时的最小总代价。调用方保证 sum(weights) == sum(demands)。
func minTotalCost(weights []int64, demands []int64, cost [][]int64) int64 {
	n := len(weights)
	m := len(demands)
	src := n + m
	sink := n + m + 1
	g := newFlowGraph(sink + 1)

	const inf = int64(1) << 62
	for i := 0; i < n; i++ {
		g.addEdge(src, i, weights[i], 0)
		for j := 0; j < m; j++ {
			g.addEdge(i, n+j, inf, cost[i][j])
		}
	}
	for j := 0; j < m; j++ {
		g.addEdge(n+j, sink, demands[j], 0)
	}

	// 所有费用非负，使用势函数保证 Dijkstra 过程中约化费用非负。
	potential := make([]int64, sink+1)
	var total int64
	for {
		dist, prevNode, prevEdge := g.shortestPath(src, sink, potential)
		if dist[sink] == inf {
			break
		}
		for v := 0; v <= sink; v++ {
			if dist[v] < inf {
				potential[v] += dist[v]
			}
		}
		// 本次增广的流量受路径上最小残余容量限制。
		aug := inf
		for v := sink; v != src; {
			e := prevEdge[v]
			if g.edges[e].cap < aug {
				aug = g.edges[e].cap
			}
			v = prevNode[v]
		}
		for v := sink; v != src; {
			e := prevEdge[v]
			g.edges[e].cap -= aug
			g.edges[e^1].cap += aug
			total += aug * g.edges[e].cost
			v = prevNode[v]
		}
	}
	return total
}

type flowEdge struct {
	to   int
	cap  int64
	cost int64
	rev  int
}

type flowGraph struct {
	edges []flowEdge
	head  [][]int
}

func newFlowGraph(nodes int) *flowGraph {
	return &flowGraph{head: make([][]int, nodes)}
}

func (g *flowGraph) addEdge(from, to int, cap, cost int64) {
	fwd := len(g.edges)
	g.edges = append(g.edges, flowEdge{to: to, cap: cap, cost: cost, rev: fwd + 1})
	g.edges = append(g.edges, flowEdge{to: from, cap: 0, cost: -cost, rev: fwd})
	g.head[from] = append(g.head[from], fwd)
	g.head[to] = append(g.head[to], fwd+1)
}

type distItem struct {
	node int
	dist int64
}

type distHeap []distItem

func (h distHeap) Len() int            { return len(h) }
func (h distHeap) Less(i, j int) bool  { return h[i].dist < h[j].dist }
func (h distHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *distHeap) Push(x interface{}) { *h = append(*h, x.(distItem)) }
func (h *distHeap) Pop() interface{} {
	old := *h
	item := old[len(old)-1]
	*h = old[:len(old)-1]
	return item
}

func (g *flowGraph) shortestPath(src, sink int, potential []int64) (dist []int64, prevNode, prevEdge []int) {
	const inf = int64(1) << 62
	dist = make([]int64, len(g.head))
	prevNode = make([]int, len(g.head))
	prevEdge = make([]int, len(g.head))
	for i := range dist {
		dist[i] = inf
	}
	dist[src] = 0
	pq := &distHeap{{node: src, dist: 0}}
	for pq.Len() > 0 {
		item := heap.Pop(pq).(distItem)
		if item.dist > dist[item.node] {
			continue
		}
		for _, ei := range g.head[item.node] {
			e := g.edges[ei]
			if e.cap <= 0 {
				continue
			}
			nd := item.dist + e.cost + potential[item.node] - potential[e.to]
			if nd < dist[e.to] {
				dist[e.to] = nd
				prevNode[e.to] = item.node
				prevEdge[e.to] = ei
				heap.Push(pq, distItem{node: e.to, dist: nd})
			}
		}
	}
	return dist, prevNode, prevEdge
}
