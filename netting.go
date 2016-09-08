package netting

import "os"
import "fmt"
import (
	"bufio"
	"github.com/gonum/graph/simple"
	"github.com/gonum/graph/topo"
	"math"
	"strconv"
)

type NettingTable struct {
	graph *simple.DirectedGraph
}

func (this *NettingTable) Init() {
	this.graph = simple.NewDirectedGraph(0, 0)
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
	graph := this.graph
	sourceNode := simple.Node(SrcCounterPartyID)
	destinationNode := simple.Node(DstCounterPartyID)
	if graph.Has(sourceNode) && graph.Has(destinationNode) && (Value > 0) {
		newEdge := simple.Edge{F: graph.Node(SrcCounterPartyID), T: graph.Node(DstCounterPartyID), W: Value}
		graph.SetEdge(newEdge)
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
	//log.Debugf("%d cycles were skipped.\n", counter)
}

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

func (this *NettingTable) print() {
	tableWithNegativeValues := this.makeACopy()
	tableWithNegativeValues.addNegativeEdges()
	g := tableWithNegativeValues.graph

	fmt.Println("")
	N := len(g.Nodes())
	h := this.CalcH()
	for j := 0; j < N; j++ {
		for i := 0; i < N; i++ {
			w, _ := g.Weight(g.Node(j), g.Node(i))
			fmt.Printf("%9.f ", w)
		}
		fmt.Printf(" | %9.f \n", h[j])
	}
	fmt.Printf("L1 norm: %9.2f, L2 norm: %9.2f \n\n", this.CalcL1(), this.CalcL2())
}

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

func main() {
	check := func(e error) {
		if e != nil {
			panic(e)
		}
	}

	if len(os.Args) != 2 {
		fmt.Println("Provide a file name as an command line argument.")
		return
	}

	fileName := os.Args[1]
	file, err := os.Open(fileName)
	check(err)

	// Create scanner
	scanner := bufio.NewScanner(bufio.NewReader(file))

	// Set the split function for the scanning operation
	scanner.Split(bufio.ScanWords)

	// Count the words
	count := 0
	data := []float64{}
	for scanner.Scan() {
		count++
		value, err := strconv.ParseFloat(scanner.Text(), 64)
		check(err)
		data = append(data, value)
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("reading input: " + err.Error())
	}

	N := int(math.Sqrt(float64(len(data))))
	fmt.Printf("Input table has %d CPs.\n\n", N)

	// Create Netting Table object
	table := NettingTable{}
	table.Init()

	// Add CPs
	for i := 0; i < N; i++ {
		cPID := table.AddCounterParty()
		fmt.Println("Adding new Counter Party with ID: " + strconv.Itoa(cPID))
	}

	// Add CP claims
	for j := 0; j < N; j++ {
		for i := 0; i < N; i++ {
			weight := data[i+j*N]
			table.AddClaim(j, i, weight)
		}
	}

	// Print the table
	table.print()

	// Run Netting Optimization
	table.Optimize()

	// Print the results
	table.print()
}
