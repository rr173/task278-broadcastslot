package conflict

import "task278-broadcastslot/internal/model"

// Edge 是引用图上的一条有向边。
type Edge struct {
	From string
	To   string
}

// Reachable 从 start 沿引用边 DFS 是否能走到 goal。start==goal 视为可达（用于自环）。
func Reachable(edges []Edge, start, goal string) bool {
	if start == "" || goal == "" {
		return false
	}
	if start == goal {
		return true
	}
	adj := adjacency(edges)
	seen := make(map[string]bool, len(adj))
	var dfs func(string) bool
	dfs = func(n string) bool {
		if n == goal {
			return true
		}
		if seen[n] {
			return false
		}
		seen[n] = true
		for _, nxt := range adj[n] {
			if dfs(nxt) {
				return true
			}
		}
		return false
	}
	return dfs(start)
}

// WouldCycle 若已存在 to→…→from，再加 from→to 即成环。
func WouldCycle(edges []Edge, from, to string) bool {
	if from == "" || to == "" {
		return false
	}
	return Reachable(edges, to, from)
}

// DetectCycle 检查新增边；成环返回 ErrSourceCycle。
func DetectCycle(existing []model.SourceCitation, from, to string) error {
	edges := make([]Edge, 0, len(existing))
	for _, c := range existing {
		edges = append(edges, Edge{From: c.FromRef, To: c.ToRef})
	}
	if WouldCycle(edges, from, to) {
		return model.ErrSourceCycle
	}
	return nil
}

func adjacency(edges []Edge) map[string][]string {
	adj := make(map[string][]string, len(edges))
	for _, e := range edges {
		adj[e.From] = append(adj[e.From], e.To)
	}
	return adj
}
