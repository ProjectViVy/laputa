// Package palace provides the memory palace graph model.
// It implements the drawer-wing-hall architecture for organizing memories.
package palace

import (
	"slices"
	"sort"
	"strings"
)

type GraphNode struct {
	Wings []string `json:"wings"`
	Halls []string `json:"halls"`
	Count int      `json:"count"`
	Dates []string `json:"dates,omitempty"`
}

type GraphEdge struct {
	Room  string `json:"room"`
	WingA string `json:"wing_a"`
	WingB string `json:"wing_b"`
	Hall  string `json:"hall,omitempty"`
	Count int    `json:"count"`
}

type Graph struct {
	nodes map[string]*GraphNode
	edges []GraphEdge
}

func NewGraph() *Graph {
	return &Graph{
		nodes: make(map[string]*GraphNode),
		edges: make([]GraphEdge, 0),
	}
}

func (g *Graph) AddDrawer(wing, room, hall string) {
	if room == "" || room == "general" || wing == "" {
		return
	}

	node, exists := g.nodes[room]
	if !exists {
		node = &GraphNode{
			Wings: []string{},
			Halls: []string{},
			Dates: []string{},
		}
		g.nodes[room] = node
	}

	node.Count++
	if !containsString(node.Wings, wing) {
		node.Wings = append(node.Wings, wing)
	}
	if hall != "" && !containsString(node.Halls, hall) {
		node.Halls = append(node.Halls, hall)
	}
}

func (g *Graph) BuildEdges() {
	for room, node := range g.nodes {
		if len(node.Wings) < 2 {
			continue
		}

		sortedWings := make([]string, len(node.Wings))
		copy(sortedWings, node.Wings)
		sort.Strings(sortedWings)

		for i, wa := range sortedWings {
			for _, wb := range sortedWings[i+1:] {
				for _, hall := range node.Halls {
					g.edges = append(g.edges, GraphEdge{
						Room:  room,
						WingA: wa,
						WingB: wb,
						Hall:  hall,
						Count: node.Count,
					})
				}
			}
		}
	}
}

func (g *Graph) GetNodes() map[string]*GraphNode {
	return g.nodes
}

func (g *Graph) GetEdges() []GraphEdge {
	return g.edges
}

func (g *Graph) GetRooms() []string {
	rooms := make([]string, 0, len(g.nodes))
	for room := range g.nodes {
		rooms = append(rooms, room)
	}
	sort.Strings(rooms)
	return rooms
}

func (g *Graph) FindTunnels(wingA, wingB string) []string {
	var tunnels []string
	for room, node := range g.nodes {
		if len(node.Wings) < 2 {
			continue
		}
		if wingA != "" && !containsString(node.Wings, wingA) {
			continue
		}
		if wingB != "" && !containsString(node.Wings, wingB) {
			continue
		}
		tunnels = append(tunnels, room)
	}
	sort.Strings(tunnels)
	return tunnels
}

func (g *Graph) Traverse(startRoom string, maxHops int) []map[string]any {
	startNode, exists := g.nodes[startRoom]
	if !exists {
		return nil
	}

	type visit struct {
		room string
		hop  int
	}

	var results []map[string]any
	visited := map[string]bool{startRoom: true}
	frontier := []visit{{startRoom, 0}}

	results = append(results, map[string]any{
		"room":  startRoom,
		"wings": startNode.Wings,
		"halls": startNode.Halls,
		"count": startNode.Count,
		"hop":   0,
	})

	for len(frontier) > 0 {
		current := frontier[0]
		frontier = frontier[1:]

		if current.hop >= maxHops {
			continue
		}

		currentNode := g.nodes[current.room]
		if currentNode == nil {
			continue
		}

		currentWings := make(map[string]bool)
		for _, w := range currentNode.Wings {
			currentWings[w] = true
		}

		for room, node := range g.nodes {
			if visited[room] {
				continue
			}

			sharedWings := 0
			for _, w := range node.Wings {
				if currentWings[w] {
					sharedWings++
				}
			}

			if sharedWings > 0 {
				visited[room] = true
				connectedVia := []string{}
				for _, w := range node.Wings {
					if currentWings[w] {
						connectedVia = append(connectedVia, w)
					}
				}
				results = append(results, map[string]any{
					"room":          room,
					"wings":         node.Wings,
					"halls":         node.Halls,
					"count":         node.Count,
					"hop":           current.hop + 1,
					"connected_via": connectedVia,
				})
				if current.hop+1 < maxHops {
					frontier = append(frontier, visit{room, current.hop + 1})
				}
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		hopI := results[i]["hop"].(int)
		hopJ := results[j]["hop"].(int)
		if hopI != hopJ {
			return hopI < hopJ
		}
		countI := results[i]["count"].(int)
		countJ := results[j]["count"].(int)
		return countI > countJ
	})

	if len(results) > 50 {
		results = results[:50]
	}

	return results
}

func (g *Graph) FuzzyMatch(query string, n int) []string {
	queryLower := strings.ToLower(query)
	var scored []struct {
		room  string
		score float64
	}

	for room := range g.nodes {
		roomLower := strings.ToLower(room)
		if strings.Contains(roomLower, queryLower) {
			scored = append(scored, struct {
				room  string
				score float64
			}{room, 1.0})
		} else {
			queryWords := strings.SplitSeq(queryLower, "-")
			for word := range queryWords {
				if strings.Contains(roomLower, word) {
					scored = append(scored, struct {
						room  string
						score float64
					}{room, 0.5})
					break
				}
			}
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	var results []string
	for i := 0; i < n && i < len(scored); i++ {
		results = append(results, scored[i].room)
	}
	return results
}

func containsString(slice []string, item string) bool {
	return slices.Contains(slice, item)
}
