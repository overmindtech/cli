package graph

import (
	"fmt"
	"testing"

	"github.com/overmindtech/cli/sdp-go"
	"gonum.org/v1/gonum/graph/network"
	"google.golang.org/protobuf/types/known/structpb"
)

func makeTestItem(name string) *sdp.Item {
	return &sdp.Item{
		Type:            "test",
		UniqueAttribute: "name",
		Scope:           "test",
		Attributes: &sdp.ItemAttributes{
			AttrStruct: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"name": {
						Kind: &structpb.Value_StringValue{
							StringValue: name,
						},
					},
				},
			},
		},
	}
}

func TestNode(t *testing.T) {
	node := Node{
		Item:   makeTestItem("test"),
		Weight: 1.5,
		Id:     1,
	}

	if node.ID() != 1 {
		t.Errorf("expected ID to be 1, got %v", node.ID())
	}
}

func TestNodes(t *testing.T) {
	nodes := Nodes{}

	nodes.Append(&Node{
		Item:   makeTestItem("a"),
		Weight: 1.5,
		Id:     1,
	})
	nodes.Append(&Node{
		Item:   makeTestItem("b"),
		Weight: 1.5,
		Id:     2,
	})

	if nodes.Len() != 2 {
		t.Errorf("expected length to be 2, got %v", nodes.Len())
	}

	// Call node before next should return nil
	if nodes.Node() != nil {
		t.Errorf("expected Node to be nil")
	}

	// Call next
	if nodes.Next() != true {
		t.Errorf("expected Next to be true")
	}

	// A
	if nodes.Node().ID() != 1 {
		t.Errorf("expected ID to be 1, got %v", nodes.Node().ID())
	}

	if nodes.Len() != 1 {
		t.Errorf("expected length to be 1, got %v", nodes.Len())
	}

	if nodes.Next() != true {
		t.Errorf("expected Next to be true")
	}

	// B
	if nodes.Node().ID() != 2 {
		t.Errorf("expected ID to be 2, got %v", nodes.Node().ID())
	}

	if nodes.Len() != 0 {
		t.Errorf("expected length to be 0, got %v", nodes.Len())
	}

	if nodes.Next() != false {
		t.Errorf("expected Next to be false")
	}

	if nodes.Node() != nil {
		t.Errorf("expected Node to be nil")

	}

	nodes.Reset()

	if nodes.Len() != 2 {
		t.Errorf("expected length to be 2, got %v", nodes.Len())
	}
}

func TestGraph(t *testing.T) {
	// A list of items that form the following graph:
	//
	//       ┌────┐
	//    ┌──┤ A  ├──┐
	//    │  └────┘  │
	//    │          │
	// ┌──▼───┐   ┌──▼─┐
	// │  B   ├──►│ C  │
	// └──┬───┘   └────┘
	//    │
	//    │
	// ┌──▼───┐
	// │  D   │
	// └──────┘
	//
	a := makeTestItem("a")
	b := makeTestItem("b")
	c := makeTestItem("c")
	d := makeTestItem("d")

	// TODO(LIQs): https://github.com/overmindtech/workspace/issues/1228
	a.LinkedItems = []*sdp.LinkedItem{
		{
			Item: b.Reference(),
		},
		{
			Item: c.Reference(),
		},
	}

	b.LinkedItems = []*sdp.LinkedItem{
		{
			Item: c.Reference(),
		},
		{
			Item: d.Reference(),
		},
	}

	graph := NewSDPGraph(false)

	aID := graph.AddItem(a, 1)
	bID := graph.AddItem(b, 1)
	cID := graph.AddItem(c, 1)
	dID := graph.AddItem(d, 1)

	fmt.Sprintln(aID, bID, cID, dID)

	t.Run("To", func(t *testing.T) {
		nodes := graph.To(cID)

		if nodes.Len() != 2 {
			t.Errorf("expected length to be 2, got %v", nodes.Len())
		}
	})

	t.Run("WeightedEdge", func(t *testing.T) {
		t.Run("with a real edge", func(t *testing.T) {
			e := graph.WeightedEdge(aID, cID)

			if e == nil {
				t.Fatal("expected edge to be non-nil")
			}

			if e.Weight() != 2 {
				t.Errorf("expected weight to be 2, got %v", e.Weight())
			}
		})

		t.Run("with a non-existent edge", func(t *testing.T) {
			e := graph.WeightedEdge(aID, dID)

			if e != nil {
				t.Errorf("expected edge to be nil")
			}
		})
	})

	t.Run("Weight", func(t *testing.T) {
		t.Run("with a real edge", func(t *testing.T) {
			w, ok := graph.Weight(aID, cID)

			if !ok {
				t.Fatal("expected edge to be non-nil")
			}

			if w != 2 {
				t.Errorf("expected weight to be 2, got %v", w)
			}
		})

		t.Run("with a non-existent edge", func(t *testing.T) {
			w, ok := graph.Weight(aID, dID)

			if ok {
				t.Errorf("expected edge to be nil")
			}

			if w != 0 {
				t.Errorf("expected weight to be 0, got %v", w)
			}
		})
	})

	t.Run("Node", func(t *testing.T) {
		t.Run("with a node that exists", func(t *testing.T) {
			n := graph.Node(aID)

			if n == nil {
				t.Fatal("expected node to be non-nil")
			}

			if n.ID() != aID {
				t.Errorf("expected ID to be %v, got %v", aID, n.ID())
			}
		})

		t.Run("with a node that doesn't exist", func(t *testing.T) {
			n := graph.Node(999)

			if n != nil {
				t.Errorf("expected node to be nil")
			}
		})
	})

	t.Run("Nodes", func(t *testing.T) {
		nodes := graph.Nodes()

		if nodes.Len() != 4 {
			t.Errorf("expected length to be 4, got %v", nodes.Len())
		}
	})

	t.Run("From", func(t *testing.T) {
		nodes := graph.From(bID)

		if nodes.Len() != 2 {
			t.Errorf("expected length to be 2, got %v", nodes.Len())
		}
	})

	t.Run("HasEdgeBetween", func(t *testing.T) {
		t.Run("with a real edge", func(t *testing.T) {
			ok := graph.HasEdgeBetween(aID, cID)

			if !ok {
				t.Fatal("expected edge to be non-nil")
			}
		})

		t.Run("with a non-existent edge", func(t *testing.T) {
			ok := graph.HasEdgeBetween(aID, dID)

			if ok {
				t.Errorf("expected edge to be nil")
			}
		})
	})

	t.Run("Edge", func(t *testing.T) {
		e := graph.Edge(aID, cID)

		if e == nil {
			t.Fatal("expected edge to be non-nil")
		}
	})

	t.Run("PageRank", func(t *testing.T) {
		ranks := network.PageRank(graph, 0.85, 0.0001)

		if len(ranks) != 4 {
			t.Errorf("expected length to be 4, got %v", len(ranks))
		}
	})

	t.Run("Undirected", func(t *testing.T) {
		directed := NewSDPGraph(false)
		undirected := NewSDPGraph(true)

		directed.AddItem(a, 1)
		directed.AddItem(b, 1)
		directed.AddItem(c, 1)
		directed.AddItem(d, 1)
		undirected.AddItem(a, 1)
		undirected.AddItem(b, 1)
		undirected.AddItem(c, 1)
		undirected.AddItem(d, 1)

		if len(undirected.edges) == 4 {
			t.Errorf("expected undirected graph to have > 4 edges, got %v", len(undirected.edges))
		}
	})
}
