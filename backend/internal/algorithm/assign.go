package algorithm

import "math"

// Candidate 参与分配的面试者（SlotIDs 按心仪顺序，index 0 为第一心仪）
type Candidate struct {
	Username  string
	StudentID string
	SlotIDs   []uint
}

// Result 分配结果
type Result struct {
	// SlotOf 用户名 -> 分到的时段 ID（不在表中表示未分配）
	SlotOf map[string]uint
	// RankOf 用户名 -> 被满足的心仪序（1/2/3，0 表示未分配）
	RankOf map[string]int
	// First/Second/Third/Unassigned 各类达成人数
	First      int
	Second     int
	Third      int
	Unassigned int
	// Participants 参与匹配的人数
	Participants int
}

// rankCost 心仪序对应费用：优先满足第一心仪，梯度拉开
var rankCost = [4]int{0, 1, 6, 20}

const inf = math.MaxInt32

type edge struct {
	to   int
	rev  int
	cap  int
	cost int
}

type graph struct {
	n     int
	edges [][]edge
}

func newGraph(n int) *graph {
	return &graph{n: n, edges: make([][]edge, n)}
}

func (g *graph) addEdge(u, v, cap, cost int) {
	g.edges[u] = append(g.edges[u], edge{to: v, rev: len(g.edges[v]), cap: cap, cost: cost})
	g.edges[v] = append(g.edges[v], edge{to: u, rev: len(g.edges[u]) - 1, cap: 0, cost: -cost})
}

// minCostMaxFlow SPFA 增广，返回 (流量, 费用, 前驱边)
func (g *graph) spfa(s, t int) (int, []int, []int) {
	dist := make([]int, g.n)
	inQueue := make([]bool, g.n)
	prevNode := make([]int, g.n)
	prevEdge := make([]int, g.n)
	for i := range dist {
		dist[i] = inf
		prevNode[i] = -1
		prevEdge[i] = -1
	}
	dist[s] = 0
	queue := []int{s}
	inQueue[s] = true
	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		inQueue[u] = false
		for i := range g.edges[u] {
			e := g.edges[u][i]
			if e.cap > 0 && dist[u]+e.cost < dist[e.to] {
				dist[e.to] = dist[u] + e.cost
				prevNode[e.to] = u
				prevEdge[e.to] = i
				if !inQueue[e.to] {
					queue = append(queue, e.to)
					inQueue[e.to] = true
				}
			}
		}
	}
	if dist[t] == inf {
		return 0, nil, nil
	}
	// 沿最短路找瓶颈
	f := inf
	for v := t; v != s; v = prevNode[v] {
		e := g.edges[prevNode[v]][prevEdge[v]]
		if e.cap < f {
			f = e.cap
		}
	}
	return f, prevNode, prevEdge
}

func (g *graph) minCostMaxFlow(s, t int) (int, int) {
	flow, cost := 0, 0
	for {
		f, prevNode, prevEdge := g.spfa(s, t)
		if f == 0 {
			break
		}
		for v := t; v != s; v = prevNode[v] {
			e := &g.edges[prevNode[v]][prevEdge[v]]
			e.cap -= f
			g.edges[e.to][e.rev].cap += f
		}
		flow += f
		cost += f * g.pathCost(prevNode, prevEdge, s, t)
	}
	return flow, cost
}

// pathCost 重新计算增广路上的单位费用
func (g *graph) pathCost(prevNode, prevEdge []int, s, t int) int {
	c := 0
	for v := t; v != s; v = prevNode[v] {
		c += g.edges[prevNode[v]][prevEdge[v]].cost
	}
	return c
}

// Assign 在「每位面试者一个时段、每时段一人」约束下完成分配（容量默认 1 的便捷入口）。
func Assign(cands []Candidate, slotIDs []uint) Result {
	caps := make(map[uint]int, len(slotIDs))
	for _, id := range slotIDs {
		caps[id] = 1
	}
	return AssignWithCapacity(cands, slotIDs, caps)
}

// AssignWithCapacity 每位面试者分到一个时段；每个时段可容纳 caps[slotID] 人同时面试。
// 费用梯度（1/6/20）保证：优先让更多人拿到第一心仪，其次第二、第三心仪。
func AssignWithCapacity(cands []Candidate, slotIDs []uint, caps map[uint]int) Result {
	res := Result{
		SlotOf:        make(map[string]uint, len(cands)),
		RankOf:        make(map[string]int, len(cands)),
		Participants:  len(cands),
		Unassigned:    len(cands),
	}
	slotIndex := make(map[uint]int, len(slotIDs))
	for i, id := range slotIDs {
		slotIndex[id] = i
	}

	src, sink := 0, 1+len(cands)+len(slotIDs)
	g := newGraph(sink + 1)
	// 源点 -> 候选人
	for i := range cands {
		g.addEdge(src, 1+i, 1, 0)
	}
	// 候选人 -> 心仪时段
	for i, cand := range cands {
		for rank, slotID := range cand.SlotIDs {
			if rank >= 3 {
				break
			}
			si, ok := slotIndex[slotID]
			if !ok || slotID == 0 {
				continue
			}
			g.addEdge(1+i, 1+len(cands)+si, 1, rankCost[rank+1])
		}
	}
	// 时段 -> 汇点：容量为管理员配置的「同时面试人数」，默认 1
	for i, id := range slotIDs {
		capacity := caps[id]
		if capacity < 1 {
			capacity = 1
		}
		g.addEdge(1+len(cands)+i, sink, capacity, 0)
	}

	g.minCostMaxFlow(src, sink)

	// 提取结果：候选人节点上 cap 归零的出边即被选中的时段
	for i, cand := range cands {
		node := 1 + i
		for _, e := range g.edges[node] {
			slotNode := e.to
			if slotNode >= 1+len(cands) && slotNode < sink && e.cap == 0 {
				slotID := slotIDs[slotNode-1-len(cands)]
				res.SlotOf[cand.Username] = slotID
				res.Unassigned--
				// 反查满足的心仪序
				for rank, sid := range cand.SlotIDs {
					if rank < 3 && sid == slotID {
						res.RankOf[cand.Username] = rank + 1
						switch rank + 1 {
						case 1:
							res.First++
						case 2:
							res.Second++
						case 3:
							res.Third++
						}
						break
					}
				}
				break
			}
		}
	}
	return res
}
