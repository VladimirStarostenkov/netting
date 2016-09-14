package netting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gonum/graph/simple"
	"github.com/gonum/graph/topo"
	"math"
)

type NettingTable struct {
	graph *simple.DirectedGraph
}

type NettingTableStats struct {
	NumberOfCounterParties 	int	 `json:"number_of_counter_parties"`
	NumberOfClaims 		int	 `json:"number_of_claims"`
	MetricL1		float64	 `json:"metric_l1"`
	MetricL2		float64	 `json:"metric_l2"`
	SumH			float64	 `json:"sum_of_h"`
}

type graphBytes struct {
	Nodes []int
	Edges []edge
}

type edge struct {
	From   int     `json:"f"`
	To     int     `json:"t"`
	Weight float64 `json:"v"`
}

func (this *NettingTable) Init() {
	this.graph = simple.NewDirectedGraph(0, 0)
}

func (this *NettingTable) InitFromBytes(bytes []byte) error {
	// Pre-init
	this.Init()

	// From Bytes
	var nodesAndEdges graphBytes
	err := json.Unmarshal(bytes, &nodesAndEdges)
	if err != nil {
		return err
	}

	// Add Nodes
	for _ = range nodesAndEdges.Nodes {
		this.AddCounterParty()
	}

	// Add Edges
	for _, edge := range nodesAndEdges.Edges {
		this.AddClaim(edge.From, edge.To, edge.Weight)
	}

	return nil
}

func (this *NettingTable) ToBytes() ([]byte, error) {
	// Collect Nodes
	thisNodes := []int{}
	for _, node := range this.graph.Nodes() {
		thisNodes = append(thisNodes, node.ID())
	}

	// Collect Edges
	thisEdges := []edge{}
	for _, e := range this.graph.Edges() {
		thisEdges = append(thisEdges, edge{From: e.From().ID(), To: e.To().ID(), Weight: e.Weight()})
	}

	// To Bytes
	nodesAndEdges := graphBytes{Nodes: thisNodes, Edges: thisEdges}
	bytes, err := json.Marshal(nodesAndEdges)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (this *NettingTable) CalcH() []float64 {
	g := this.graph
	N := len(g.Nodes())
	h := make([]float64, N)
	for j := 0; j < N; j++ {
		for i := 0; i < N; i++ {
			w, _ := g.Weight(g.Node(j), g.Node(i))
			h[j] += w
		}
	}
	return h
}

func (this *NettingTable) CalcL1() float64 {
	g := this.graph
	N := len(g.Nodes())
	if N == 0 {
		return -1.0
	}
	cAbsSum := 0.0
	for i := 0; i < N; i++ {
		for j := i + 1; j < N; j++ {
			w, _ := g.Weight(g.Node(i), g.Node(j))
			cAbsSum += math.Abs(w)
		}
	}
	L1 := cAbsSum / float64(N*(N-1)) * 2.0
	return L1
}

func (this *NettingTable) CalcL2() float64 {
	g := this.graph
	N := len(g.Nodes())
	if N == 0 {
		return -1.0
	}
	cQuadSum := 0.0
	for i := 0; i < N; i++ {
		for j := i + 1; j < N; j++ {
			w, _ := g.Weight(g.Node(i), g.Node(j))
			cQuadSum += math.Pow(w, 2)
		}
	}
	L2 := math.Sqrt(cQuadSum / float64(N*(N-1)) * 2.0)
	return L2
}

func (this *NettingTable) AddCounterParty() (CounterPartyID int) {
	g := this.graph
	CounterPartyID = g.NewNodeID()
	g.AddNode(simple.Node(CounterPartyID))
	return
}

func (this *NettingTable) AddClaim(SrcCounterPartyID int, DstCounterPartyID int, Value float64) {
	if (SrcCounterPartyID == DstCounterPartyID) {
		return
	}
	graph := this.graph
	sourceNode := simple.Node(SrcCounterPartyID)
	destinationNode := simple.Node(DstCounterPartyID)
	if graph.Has(sourceNode) && graph.Has(destinationNode) && (Value > 0) {
		// 2 cases when an edge {source -> destination} or {destination -> source} already exists
		if existingEdge := graph.Edge(sourceNode, destinationNode); existingEdge != nil {
			Value += existingEdge.Weight()
			graph.RemoveEdge(existingEdge)
		} else if existingEdge := graph.Edge(destinationNode, sourceNode); existingEdge != nil {
			Value -= existingEdge.Weight()
			graph.RemoveEdge(existingEdge)
		}

		if Value > 0 {
			newEdge := simple.Edge{F: graph.Node(SrcCounterPartyID), T: graph.Node(DstCounterPartyID), W: Value}
			graph.SetEdge(newEdge)
		} else if Value < 0 {
			newEdge := simple.Edge{F: graph.Node(DstCounterPartyID), T: graph.Node(SrcCounterPartyID), W: -Value}
			graph.SetEdge(newEdge)
		}
	}
}

func (this *NettingTable) Optimize() {
	graph := this.graph

	// Find all cycles in graph
	cycles := topo.CyclesIn(graph)
	//log.Debugf("Number of cycles in graph: %d \n", len(cycles))

	// Loop optimize graph
	counter := 0
	for _, cycle := range cycles {
		// find min weight in cycle
		minWeight := math.MaxFloat64
		for i := 0; i < len(cycle)-1; i++ {
			weight := graph.Edge(cycle[i], cycle[i+1]).Weight()
			if weight < minWeight {
				minWeight = weight
			}
		}
		if minWeight == 0.0 {
			counter++
			continue
		}

		// subtract
		for i := 0; i < len(cycle)-1; i++ {
			oldEdge := graph.Edge(cycle[i], cycle[i+1])
			newEdge := simple.Edge{
				F: oldEdge.From(),
				T: oldEdge.To(),
				W: oldEdge.Weight() - minWeight}
			graph.RemoveEdge(oldEdge)
			graph.SetEdge(newEdge)
		}
	}

	// Remove 0.0 edges
	for _, edge := range graph.Edges() {
		if edge.Weight() == 0.0 {
			graph.RemoveEdge(edge)
		}
	}
	//log.Debugf("%d cycles were skipped.\n", counter)
}

func (this *NettingTable) GetClaims(CounterPartyID int) ([]byte) {
	tableWithNegativeValues := this.makeACopy()
	tableWithNegativeValues.addNegativeEdges()
	g := tableWithNegativeValues.graph

	claims := []edge{}
	counterPartyNode := g.Node(CounterPartyID)
	for _, destinationNode := range g.From(counterPartyNode) {
		from := counterPartyNode.ID()
		to := destinationNode.ID()
		value, _ := g.Weight(counterPartyNode, destinationNode)

		claims = append(claims, edge{From: from, To: to, Weight: value})
	}

	result, err := json.Marshal(claims)
	if err != nil {
		return []byte{}
	}

	return result
}

func (this *NettingTable) GetStats() ([]byte) {
	floatSum := func(vals []float64) (sum float64) {
		for _, val := range vals {
			sum += val
		}
		return
	}

	tableWithNegativeValues := this.makeACopy()
	tableWithNegativeValues.addNegativeEdges()
	g := tableWithNegativeValues.graph

	stats := NettingTableStats{
		NumberOfCounterParties: len(g.Nodes()),
		NumberOfClaims: len(g.Edges()) / 2,
		MetricL1: tableWithNegativeValues.CalcL1(),
		MetricL2: tableWithNegativeValues.CalcL2(),
		SumH: floatSum(tableWithNegativeValues.CalcH()),

	}

	result, err := json.Marshal(stats)
	//fmt.Printf("%s\n", string(result))
	if err != nil {
		fmt.Errorf("%s", err.Error())
		return []byte{}
	}

	return result
}

// internal
func (this *NettingTable) addNegativeEdges() {
	g := this.graph
	N := len(g.Nodes())
	for j := 0; j < N; j++ {
		J := g.Node(j)
		for i := 0; i < N; i++ {
			I := g.Node(i)
			w, exists := g.Weight(J, I)
			if exists && w > 0.0 {
				if negativeEdge := g.Edge(I, J); negativeEdge == nil {
					g.SetEdge(simple.Edge{F: I, T: J, W: -w})
				}
			}
		}
	}
}
// internal
func (this *NettingTable) toText() string {
	tableWithNegativeValues := this.makeACopy()
	tableWithNegativeValues.addNegativeEdges()
	g := tableWithNegativeValues.graph

	var buf bytes.Buffer
	buf.WriteString("\n")

	N := len(g.Nodes())
	h := tableWithNegativeValues.CalcH()
	for j := 0; j < N; j++ {
		for i := 0; i < N; i++ {
			w, _ := g.Weight(g.Node(j), g.Node(i))
			buf.WriteString(fmt.Sprintf("%9.f ", w))
		}
		buf.WriteString(fmt.Sprintf(" | %9.f \n", h[j]))
	}
	buf.WriteString(fmt.Sprintf("L1 norm: %9.2f, L2 norm: %9.2f \n\n",
		tableWithNegativeValues.CalcL1(), tableWithNegativeValues.CalcL2()))

	return buf.String()
}
// internal
func (this *NettingTable) makeACopy() (copy NettingTable) {
	copy.Init()
	graph := this.graph
	graphCopy := copy.graph
	for _, _ = range graph.Nodes() {
		copy.AddCounterParty()
	}
	for _, edge := range graph.Edges() {
		graphCopy.SetEdge(edge)
	}
	return
}
