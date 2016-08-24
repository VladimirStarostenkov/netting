package main

import "os"
import "fmt"
import (
	"bufio"
	"strconv"
	"github.com/gonum/graph/simple"
	"github.com/gonum/graph/topo"
	"math"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func calcH(g *simple.DirectedGraph) ([]float64) {
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

func calcL1(g *simple.DirectedGraph) (float64) {
	N := len(g.Nodes())
	cAbsSum := 0.0
	for i := 0; i < N; i++ {
		for j := i + 1; j < N; j++ {
			w, _ := g.Weight(g.Node(i), g.Node(j))
			cAbsSum += math.Abs(w)
		}
	}
	L1 := cAbsSum / float64(N * (N - 1)) * 2.0
	return L1
}

func calcL2(g *simple.DirectedGraph) (float64) {
	N := len(g.Nodes())
	cQuadSum := 0.0
	for i := 0; i < N; i++ {
		for j := i + 1; j < N; j++ {
			w, _ := g.Weight(g.Node(i), g.Node(j))
			cQuadSum += math.Pow(w, 2)
		}
	}
	L2 := math.Sqrt(cQuadSum / float64(N * (N - 1)) * 2.0)
	return L2
}

func addNegativeEdges(g *simple.DirectedGraph) {
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

func printGraph(g *simple.DirectedGraph) {
	fmt.Println("")
	N := len(g.Nodes())
	h := calcH(g)
	for j := 0; j < N; j++ {
		for i := 0; i < N; i++ {
			w, _ := g.Weight(g.Node(j), g.Node(i))
			fmt.Printf("%9.f ", w)
		}
		fmt.Printf(" | %9.f \n", h[j])
	}
	fmt.Printf("L1 norm: %9.2f, L2 norm: %9.2f \n\n", calcL1(g), calcL2(g))
}

func main() {
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

	// Create graph object
	N := int(math.Sqrt(float64(len(data))));
	fmt.Printf("Input graph has %d nodes.\n\n", N)
	graph := simple.NewDirectedGraph(0, 0)

	// Add graph nodes
	for i := 0; i < N; i++ {
		newNodeId := graph.NewNodeID()
		//fmt.Println("Adding new node with ID: " + strconv.Itoa(newNodeId))
		graph.AddNode(simple.Node(newNodeId))
	}

	// Add graph edges (only positive)
	for j := 0; j < N; j++ {
		for i := 0; i < N; i++ {
			weight := data[i + j*N]
			if weight > 0.0 {
				edge := simple.Edge{F: graph.Node(j), T: graph.Node(i), W: weight}
				graph.SetEdge(edge)
			}
		}
	}

	// Copy the graph and add negative edges to print graph and its h
	graphWithNegatives := simple.NewDirectedGraph(0, 0)
	for _, _ = range graph.Nodes() {
		graphWithNegatives.AddNode(simple.Node(graphWithNegatives.NewNodeID()))
	}
	for _, edge := range graph.Edges() {
		graphWithNegatives.SetEdge(edge)
	}
	addNegativeEdges(graphWithNegatives)
	printGraph(graphWithNegatives)

	// Find all cycles in graph
	cycles := topo.CyclesIn(graph)
	fmt.Printf("Number of cycles in graph: %d \n", len(cycles))
	counter := 0
	for _, cycle := range cycles {
		//fmt.Printf("%+v\n", cycle)

		// find min weight in cycle
		minWeight := math.MaxFloat64
		for i := 0; i < len(cycle) - 1; i++ {
			weight := graph.Edge(cycle[i], cycle[i + 1]).Weight()
			if weight < minWeight {
				minWeight = weight
			}
		}
		if minWeight == 0.0 {
			counter++
			continue
		}

		// subtract
		for i := 0; i < len(cycle) - 1; i++ {
			oldEdge := graph.Edge(cycle[i], cycle[i + 1])
			newEdge := simple.Edge{
				F: oldEdge.From(),
				T: oldEdge.To(),
				W: oldEdge.Weight() - minWeight}
			graph.RemoveEdge(oldEdge)
			graph.SetEdge(newEdge)
		}
	}
	fmt.Printf("%d cycles were skipped.\n", counter)

	// Add negative edges and print the results
	addNegativeEdges(graph)
	printGraph(graph)
}




