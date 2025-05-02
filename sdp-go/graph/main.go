// This was written as part of an experiment That required the use of the
// pagerank algorithm on Overmind data. This satisfies the interfaces inside the
// gonum package, which means that we can use any of the code in
// [gonum.org/v1/gonum/graph](https://pkg.go.dev/gonum.org/v1/gonum/graph@v0.15.0)
// to analyse our data.
package graph

import (
	"github.com/overmindtech/cli/sdp-go"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/set/uid"
)

///////////
// Nodes //
///////////

var _ graph.Node = &Node{}

// A node is always an item
type Node struct {
	Item   *sdp.Item
	Weight float64
	Id     int64
}

// A graph-unique integer ID
func (n *Node) ID() int64 {
	return n.Id
}

var _ graph.Nodes = &Nodes{}

type Nodes struct {
	// The nodes in the iterator
	nodes []*Node

	// The current position in the iterator
	i int
}

// Adds a new node to the list
func (n *Nodes) Append(node *Node) {
	n.nodes = append(n.nodes, node)
}

// Next advances the iterator and returns whether the next call to the item
// method will return a non-nil item.
//
// Next should be called prior to any call to the iterator's item retrieval
// method after the iterator has been obtained or reset.
//
// The order of iteration is implementation dependent.
func (n *Nodes) Next() bool {
	n.i++
	return n.i-1 < len(n.nodes)
}

// Len returns the number of items remaining in the iterator.
//
// If the number of items in the iterator is unknown, too large to materialize
// or too costly to calculate then Len may return a negative value. In this case
// the consuming function must be able to operate on the items of the iterator
// directly without materializing the items into a slice. The magnitude of a
// negative length has implementation-dependent semantics.
func (n *Nodes) Len() int {
	return len(n.nodes) - n.i
}

// Reset returns the iterator to its start position.
func (n *Nodes) Reset() {
	n.i = 0
}

// Node returns the current Node from the iterator.
func (n *Nodes) Node() graph.Node {
	// The Next() function gets called *before* the first item is returned, so
	// we need to return the item at position i (e.g. 1 is 1st position) rather
	// than the actual index i. This allows us to start i at zero which makes a
	// lot more sense
	getIndex := n.i - 1

	if getIndex >= len(n.nodes) || getIndex < 0 {
		return nil
	}

	return n.nodes[getIndex]
}

///////////
// Edges //
///////////

var _ graph.WeightedEdge = &Edge{}

type Edge struct {
	from   *Node
	to     *Node
	weight float64
}

// Creates a new edge. The weight of an edge is the sum of the weights of the
// two nodes
func NewEdge(from, to *Node) *Edge {
	return &Edge{
		from:   from,
		to:     to,
		weight: from.Weight + to.Weight,
	}
}

// From returns the from node of the edge.
func (e *Edge) From() graph.Node {
	return e.from
}

// To returns the to node of the edge.
func (e *Edge) To() graph.Node {
	return e.to
}

// ReversedEdge returns the edge reversal of the receiver if a reversal is valid
// for the data type. When a reversal is valid an edge of the same type as the
// receiver with nodes of the receiver swapped should be returned, otherwise the
// receiver should be returned unaltered.
func (e *Edge) ReversedEdge() graph.Edge {
	return nil
}

func (e *Edge) Weight() float64 {
	return e.weight
}

///////////
// Graph //
///////////

// Assert that SDPGraph satisfies the graph.WeightedDirected interface
var _ graph.WeightedDirected = &SDPGraph{}

type SDPGraph struct {
	uidSet *uid.Set

	nodesByID  map[int64]*Node
	nodesByGUN map[string]*Node

	// A map of items that have not been seen yet. The key is the GUN of the
	// "To" end of the edge, and the value is a slice of nodes that are the
	// "From" edges
	unseenEdges map[string][]*Node

	edges []*Edge

	undirected bool
}

// NewSDPGraph creates a new SDPGraph. If undirected is true, the graph will be
// treated as undirected, meaning that all edges will be bidirectional
func NewSDPGraph(undirected bool) *SDPGraph {
	return &SDPGraph{
		uidSet:      uid.NewSet(),
		nodesByID:   make(map[int64]*Node),
		nodesByGUN:  make(map[string]*Node),
		unseenEdges: make(map[string][]*Node),
		edges:       make([]*Edge, 0),
		undirected:  undirected,
	}
}

// AddItem adds an item to the graph including processing of its edges, returns
// the ID the node was assigned.
func (g *SDPGraph) AddItem(item *sdp.Item, weight float64) int64 {
	id := g.uidSet.NewID()
	g.uidSet.Use(id)

	// Add the node to the storage
	node := Node{
		Item:   item,
		Weight: weight,
		Id:     id,
	}
	g.nodesByID[id] = &node
	g.nodesByGUN[item.GloballyUniqueName()] = &node

	// TODO(LIQs): https://github.com/overmindtech/workspace/issues/1228
	// Find all edges and add them
	for _, linkedItem := range item.GetLinkedItems() {
		// Check if the linked item node exists
		linkedItemNode, exists := g.nodesByGUN[linkedItem.GetItem().GloballyUniqueName()]

		if exists {
			// Add the edge
			g.edges = append(g.edges, NewEdge(&node, linkedItemNode))

			if g.undirected {
				// Also add the reverse edge
				g.edges = append(g.edges, NewEdge(linkedItemNode, &node))
			}
		} else {
			// If the target for the edge doesn't exist, add this to the list to
			// be created later
			if _, exists := g.unseenEdges[linkedItem.GetItem().GloballyUniqueName()]; !exists {
				g.unseenEdges[linkedItem.GetItem().GloballyUniqueName()] = []*Node{&node}
			} else {
				g.unseenEdges[linkedItem.GetItem().GloballyUniqueName()] = append(g.unseenEdges[linkedItem.GetItem().GloballyUniqueName()], &node)
			}
		}
	}

	// If there are any unseen edges that are now seen, add them
	if unseenEdges, exists := g.unseenEdges[item.GloballyUniqueName()]; exists {
		for _, unseenEdge := range unseenEdges {
			// Add the edge
			g.edges = append(g.edges, NewEdge(unseenEdge, &node))

			if g.undirected {
				// Also add the reverse edge
				g.edges = append(g.edges, NewEdge(&node, unseenEdge))
			}
		}
	}

	return id
}

// HasEdgeFromTo returns whether an edge exists in the graph from u to v with
// the IDs uid and vid.
func (g *SDPGraph) HasEdgeFromTo(uid, vid int64) bool {
	for _, edge := range g.edges {
		if edge.from.Id == uid && edge.to.Id == vid {
			return true
		}
	}

	return false
}

// To returns all nodes that can reach directly to the node with the given ID.
//
// To must not return nil.
func (g *SDPGraph) To(id int64) graph.Nodes {
	nodes := Nodes{}

	for _, edge := range g.edges {
		if edge.to.Id == id {
			nodes.Append(edge.to)
		}
	}

	return &nodes
}

// WeightedEdge returns the weighted edge from u to v with IDs uid and vid if
// such an edge exists and nil otherwise. The node v must be directly reachable
// from u as defined by the From method.
func (g *SDPGraph) WeightedEdge(uid, vid int64) graph.WeightedEdge {
	for _, edge := range g.edges {
		if edge.from.Id == uid && edge.to.Id == vid {
			return edge
		}
	}

	return nil
}

// Weight returns the weight for the edge between x and y with IDs xid and yid
// if Edge(xid, yid) returns a non-nil Edge. If x and y are the same node or
// there is no joining edge between the two nodes the weight value returned is
// implementation dependent. Weight returns true if an edge exists between x and
// y or if x and y have the same ID, false otherwise.
func (g *SDPGraph) Weight(xid, yid int64) (w float64, ok bool) {
	edge := g.WeightedEdge(xid, yid)

	if edge == nil {
		return 0, false
	}

	return edge.Weight(), true
}

// Node returns the node with the given ID if it exists in the graph, and nil
// otherwise.
func (g *SDPGraph) Node(id int64) graph.Node {
	node, exists := g.nodesByID[id]

	if !exists {
		return nil
	}

	return node
}

// Gets a node from the graph by it's globally unique name
func (g *SDPGraph) NodeByGloballyUniqueName(globallyUniqueName string) *Node {
	node, exists := g.nodesByGUN[globallyUniqueName]

	if !exists {
		return nil
	}

	return node
}

// Nodes returns all the nodes in the graph.
//
// Nodes must not return nil.
func (g *SDPGraph) Nodes() graph.Nodes {
	nodes := Nodes{}

	for _, node := range g.nodesByID {
		nodes.Append(node)
	}

	return &nodes
}

// From returns all nodes that can be reached directly from the node with the
// given ID.
//
// From must not return nil.
func (g *SDPGraph) From(id int64) graph.Nodes {
	nodes := Nodes{}

	for _, edge := range g.edges {
		if edge.From().ID() == id {
			nodes.Append(edge.to)
		}
	}

	return &nodes
}

// HasEdgeBetween returns whether an edge exists between nodes with IDs xid and
// yid without considering direction.
func (g *SDPGraph) HasEdgeBetween(xid, yid int64) bool {
	var fromID int64
	var toID int64

	for _, edge := range g.edges {
		fromID = edge.From().ID()
		toID = edge.To().ID()

		if (fromID == xid && toID == yid) || (fromID == yid && toID == xid) {
			return true
		}
	}

	return false
}

// Edge returns the edge from u to v, with IDs uid and vid, if such an edge
// exists and nil otherwise. The node v must be directly reachable from u as
// defined by the From method.
func (g *SDPGraph) Edge(uid, vid int64) graph.Edge {
	for _, edge := range g.edges {
		if (edge.From().ID() == uid) && (edge.To().ID() == vid) {
			return edge
		}
	}

	return nil
}
