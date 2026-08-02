package palace

import (
	"reflect"
	"testing"
)

func TestAddDrawer(t *testing.T) {
	g := NewGraph()

	g.AddDrawer("alpha", "room1", "hall-a")
	g.AddDrawer("alpha", "room1", "hall-b")
	g.AddDrawer("beta", "room1", "hall-a")

	if len(g.nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(g.nodes))
	}

	node := g.nodes["room1"]
	if node.Count != 3 {
		t.Errorf("expected count 3, got %d", node.Count)
	}
	if !containsString(node.Wings, "alpha") || !containsString(node.Wings, "beta") {
		t.Errorf("expected wings alpha and beta, got %v", node.Wings)
	}
	if !containsString(node.Halls, "hall-a") || !containsString(node.Halls, "hall-b") {
		t.Errorf("expected halls hall-a and hall-b, got %v", node.Halls)
	}
}

func TestAddDrawerSkipsEmptyAndGeneral(t *testing.T) {
	g := NewGraph()

	g.AddDrawer("alpha", "", "hall")
	g.AddDrawer("alpha", "general", "hall")
	g.AddDrawer("", "room1", "hall")

	if len(g.nodes) != 0 {
		t.Errorf("expected 0 nodes for empty/general room or wing, got %d", len(g.nodes))
	}
}

func TestAddDrawerMultipleRooms(t *testing.T) {
	g := NewGraph()

	g.AddDrawer("alpha", "room1", "hall-a")
	g.AddDrawer("beta", "room2", "hall-b")

	if len(g.nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(g.nodes))
	}
}

func TestBuildEdges(t *testing.T) {
	g := NewGraph()

	g.AddDrawer("alpha", "room1", "hall-a")
	g.AddDrawer("beta", "room1", "hall-a")
	g.AddDrawer("alpha", "room2", "hall-b")
	g.AddDrawer("beta", "room2", "hall-b")

	g.BuildEdges()

	if len(g.edges) == 0 {
		t.Error("expected edges to be built")
	}
}

func TestBuildEdgesSingleWing(t *testing.T) {
	g := NewGraph()

	g.AddDrawer("alpha", "room1", "hall-a")

	g.BuildEdges()

	if len(g.edges) != 0 {
		t.Errorf("expected 0 edges for single wing, got %d", len(g.edges))
	}
}

func TestGetNodes(t *testing.T) {
	g := NewGraph()

	g.AddDrawer("alpha", "room1", "hall-a")

	nodes := g.GetNodes()
	if len(nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(nodes))
	}
}

func TestGetEdges(t *testing.T) {
	g := NewGraph()

	g.AddDrawer("alpha", "room1", "hall-a")
	g.AddDrawer("beta", "room1", "hall-a")

	g.BuildEdges()

	edges := g.GetEdges()
	if len(edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(edges))
	}
}

func TestGetRooms(t *testing.T) {
	g := NewGraph()

	g.AddDrawer("alpha", "room2", "hall-a")
	g.AddDrawer("alpha", "room1", "hall-a")

	rooms := g.GetRooms()
	expected := []string{"room1", "room2"}
	if !reflect.DeepEqual(rooms, expected) {
		t.Errorf("expected %v, got %v", expected, rooms)
	}
}

func TestFindTunnels(t *testing.T) {
	g := NewGraph()

	g.AddDrawer("alpha", "room1", "hall-a")
	g.AddDrawer("beta", "room1", "hall-a")
	g.AddDrawer("alpha", "room2", "hall-b")
	g.AddDrawer("gamma", "room3", "hall-c")

	tunnels := g.FindTunnels("alpha", "beta")
	if len(tunnels) != 1 || tunnels[0] != "room1" {
		t.Errorf("expected [room1], got %v", tunnels)
	}
}

func TestFindTunnelsSingleWingFilter(t *testing.T) {
	g := NewGraph()

	g.AddDrawer("alpha", "room1", "hall-a")
	g.AddDrawer("beta", "room1", "hall-a")

	tunnels := g.FindTunnels("alpha", "")
	if len(tunnels) != 1 || tunnels[0] != "room1" {
		t.Errorf("expected [room1], got %v", tunnels)
	}
}

func TestFindTunnelsNoMatch(t *testing.T) {
	g := NewGraph()

	g.AddDrawer("alpha", "room1", "hall-a")
	g.AddDrawer("beta", "room1", "hall-a")
	g.AddDrawer("gamma", "room2", "hall-b")

	tunnels := g.FindTunnels("alpha", "gamma")
	if len(tunnels) != 0 {
		t.Errorf("expected [], got %v", tunnels)
	}
}

func TestTraverse(t *testing.T) {
	g := NewGraph()

	g.AddDrawer("alpha", "room1", "hall-a")
	g.AddDrawer("beta", "room1", "hall-a")
	g.AddDrawer("beta", "room2", "hall-b")
	g.AddDrawer("gamma", "room3", "hall-c")

	results := g.Traverse("room1", 3)
	if results == nil {
		t.Fatal("expected non-nil results")
	}
	if len(results) < 2 {
		t.Errorf("expected at least 2 rooms visited, got %d", len(results))
	}
}

func TestTraverseStartRoomNotExists(t *testing.T) {
	g := NewGraph()

	g.AddDrawer("alpha", "room1", "hall-a")

	results := g.Traverse("nonexistent", 3)
	if results != nil {
		t.Errorf("expected nil for nonexistent room, got %v", results)
	}
}

func TestTraverseMaxHopsOne(t *testing.T) {
	g := NewGraph()

	g.AddDrawer("alpha", "room1", "hall-a")
	g.AddDrawer("beta", "room1", "hall-a")
	g.AddDrawer("beta", "room2", "hall-b")
	g.AddDrawer("gamma", "room3", "hall-c")

	results := g.Traverse("room1", 1)
	if len(results) != 2 {
		t.Errorf("expected 2 rooms with maxHops=1, got %d", len(results))
	}
}

func TestTraverseHopSorting(t *testing.T) {
	g := NewGraph()

	g.AddDrawer("alpha", "room1", "hall-a")
	g.AddDrawer("beta", "room1", "hall-a")
	g.AddDrawer("beta", "room2", "hall-b")
	g.AddDrawer("gamma", "room3", "hall-c")

	results := g.Traverse("room1", 3)

	for i := 1; i < len(results); i++ {
		hopI := results[i]["hop"].(int)
		hopJ := results[i-1]["hop"].(int)
		if hopI < hopJ {
			t.Errorf("results not sorted by hop: results[%d].hop=%d < results[%d].hop=%d", i, hopI, i-1, hopJ)
		}
	}
}

func TestTraverseCountLimit(t *testing.T) {
	g := NewGraph()

	for i := range 60 {
		g.AddDrawer("alpha", "room1", "hall-a")
		g.AddDrawer("beta", "room1", "hall-a")
		g.AddDrawer("beta", "room"+string(rune('A'+i)), "hall-b")
	}

	results := g.Traverse("room1", 3)
	if len(results) > 50 {
		t.Errorf("expected max 50 results, got %d", len(results))
	}
}

func TestFuzzyMatch(t *testing.T) {
	g := NewGraph()

	g.AddDrawer("alpha", "machine-learning", "hall-a")
	g.AddDrawer("beta", "deep-neural", "hall-b")
	g.AddDrawer("gamma", "reinforcement", "hall-c")

	matches := g.FuzzyMatch("machine", 5)
	if len(matches) == 0 {
		t.Error("expected fuzzy matches for 'machine'")
	}
	if len(matches) > 0 && matches[0] != "machine-learning" {
		t.Errorf("expected machine-learning first, got %v", matches)
	}
}

func TestFuzzyMatchExact(t *testing.T) {
	g := NewGraph()

	g.AddDrawer("alpha", "machine-learning", "hall-a")
	g.AddDrawer("beta", "deep-neural", "hall-b")

	matches := g.FuzzyMatch("deep-neural", 5)
	if len(matches) == 0 {
		t.Error("expected exact match for 'deep-neural'")
	}
}

func TestFuzzyMatchHyphenated(t *testing.T) {
	g := NewGraph()

	g.AddDrawer("alpha", "machine-learning", "hall-a")
	g.AddDrawer("beta", "deep-neural", "hall-b")

	matches := g.FuzzyMatch("deep", 5)
	if len(matches) == 0 {
		t.Error("expected match for hyphenated word 'deep'")
	}
}

func TestFuzzyMatchLimit(t *testing.T) {
	g := NewGraph()

	g.AddDrawer("alpha", "aaa-room", "hall-a")
	g.AddDrawer("beta", "bbb-room", "hall-b")
	g.AddDrawer("gamma", "ccc-room", "hall-c")
	g.AddDrawer("delta", "ddd-room", "hall-d")

	matches := g.FuzzyMatch("-room", 2)
	if len(matches) != 2 {
		t.Errorf("expected max 2 matches, got %d", len(matches))
	}
}

func TestFuzzyMatchNoResults(t *testing.T) {
	g := NewGraph()

	g.AddDrawer("alpha", "room1", "hall-a")

	matches := g.FuzzyMatch("nonexistent", 5)
	if len(matches) != 0 {
		t.Errorf("expected no matches, got %v", matches)
	}
}

func TestContainsString(t *testing.T) {
	slice := []string{"a", "b", "c"}

	if !containsString(slice, "b") {
		t.Error("expected containsString to find 'b'")
	}
	if containsString(slice, "d") {
		t.Error("expected containsString to not find 'd'")
	}
}
