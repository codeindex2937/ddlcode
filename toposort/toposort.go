// ref: github.com/stevenle/topsort
// ref: github.com/oko/toposort
package toposort

import (
	"container/heap"
	"errors"
)

var (
	ErrNodeExists       = errors.New("node already exists in topology")
	ErrNodeDoesNotExist = errors.New("node does not exist in topology")
	ErrRuntimeExceeded  = errors.New("sort runtime exceeded bound")
)

type ErrCycleInTopology[Key comparable] struct {
	OriginalEdges  map[Key]int
	RemainingEdges map[Key]int
}

func (e *ErrCycleInTopology[Key]) Error() string {
	return "cycle in topology"
}

type Graph[Key comparable] struct {
	nodes    map[Key]nodeimpl[Key]
	inDegree map[Key]map[Key]bool
}

type nodeimpl[Key comparable] struct {
	id Key
	es map[Key]bool
}

func (n nodeimpl[Key]) addEdge(key Key) {
	n.es[key] = true
}

func NewGraph[Key comparable]() *Graph[Key] {
	return &Graph[Key]{
		nodes:    make(map[Key]nodeimpl[Key]),
		inDegree: make(map[Key]map[Key]bool),
	}
}

func (g *Graph[Key]) AddNode(key Key) {
	if !g.ContainsNode(key) {
		g.nodes[key] = nodeimpl[Key]{
			id: key,
			es: map[Key]bool{},
		}
	}
}

func (g *Graph[Key]) getOrAddNode(node Key) nodeimpl[Key] {
	n, ok := g.nodes[node]
	if !ok {
		n = nodeimpl[Key]{
			id: node,
			es: map[Key]bool{},
		}
		g.nodes[node] = n
	}
	return n
}

func (g *Graph[Key]) AddEdge(from Key, to Key) error {
	f := g.getOrAddNode(from)
	g.AddNode(to)
	f.addEdge(to)
	if _, ok := g.inDegree[to]; !ok {
		g.inDegree[to] = make(map[Key]bool)
	}
	g.inDegree[to][from] = true
	return nil
}

func (g *Graph[Key]) ContainsNode(key Key) bool {
	_, ok := g.nodes[key]
	return ok
}

func (g *Graph[Key]) Neighbors(key Key) []Key {
	keys := []Key{}
	for k := range g.nodes[key].es {
		keys = append(keys, k)
	}
	return keys
}

func (g *Graph[Key]) InDegree(key Key) int {
	return len(g.inDegree[key])
}

func (g *Graph[Key]) OutDegree(key Key) int {
	return len(g.nodes[key].es)
}

// Sort returns a valid topological sorting of this topology's Nodes
func (t *Graph[Key]) Sort() ([]Key, error) {
	/*
		Implementation of Kahn's algorithm: Wikipedia pseudocode

			L ← Empty list that will contain the sorted elements
			S ← Set of all Nodes with no incoming edge
			while S is non-empty do
			    remove a node n from S
			    add n to tail of L
			    for each node m with an edge e from n to m do
			        remove edge e from the graph
			        if m has no other incoming Edges then
			            insert m into S
			if graph has Edges then
			    return error   (graph has at least one cycle)
			else
			    return L   (a topologically sorted order)
	*/
	L := make([]Key, 0, len(t.nodes))
	Sq := &binaryheap[string]{}
	heap.Init(Sq)
	for _, x := range t.starts() {
		heap.Push(Sq, x.id)
	}
	inDegree := map[Key]int{}
	for k, v := range t.inDegree {
		inDegree[k] = len(v)
	}

	i := 0
	for {
		if Sq.Len() == 0 {
			break
		}

		n := heap.Pop(Sq).(Key)
		L = append(L, n)

		eq := &binaryheap[string]{}
		heap.Init(eq)
		for id := range t.nodes[n].es {
			heap.Push(eq, id)
		}
		for eq.Len() != 0 {
			m := heap.Pop(eq).(Key)
			inDegree[m] -= 1
			if inDegree[m] == 0 {
				heap.Push(Sq, m)
			}
		}
		i++

		// in case of bugs...
		if i > 2*t.bound() {
			return nil, ErrRuntimeExceeded
		}
	}

	remainEdges := 0
	for _, v := range inDegree {
		remainEdges += v
	}
	if remainEdges > 0 {
		originInDegree := map[Key]int{}
		for k, v := range t.inDegree {
			originInDegree[k] = len(v)
		}
		return nil, &ErrCycleInTopology[Key]{OriginalEdges: originInDegree, RemainingEdges: copyMap(inDegree)}
	}
	return L, nil
}

func (t *Graph[Key]) bound() int {
	sum := len(t.nodes)
	for _, ne := range t.inDegree {
		sum += len(ne)
	}
	return sum
}

func (t *Graph[Key]) starts() []nodeimpl[Key] {
	ret := make([]nodeimpl[Key], 0)
	for k, n := range t.nodes {
		if len(t.inDegree[k]) > 0 {
			continue
		}
		ret = append(ret, n)
	}
	return ret
}

func copyMap[Key comparable, V any](src map[Key]V) map[Key]V {
	clone := map[Key]V{}
	for k, v := range src {
		clone[k] = v
	}
	return clone
}
