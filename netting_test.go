package netting

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"testing"
)

func TestNetting(t *testing.T) {
	check := func(e error) {
		if e != nil {
			fmt.Println(e.Error())
			t.Fail()
		}
	}

	fileName := "example01.txt"
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
		table.AddCounterParty()
		//fmt.Println("Adding new Counter Party with ID: " + strconv.Itoa(cPID))
	}

	// Add CP claims
	for j := 0; j < N; j++ {
		for i := 0; i < N; i++ {
			weight := data[i+j*N]
			table.AddClaim(j, i, weight)
		}
	}

	// Print the table
	fmt.Print(table.toText())

	// Run Netting Optimization
	table.Optimize()

	// Print the results
	fmt.Print(table.toText())

	// Print bytes
	fmt.Println("\n\n")
	bytes, _ := table.ToBytes()
	fmt.Println(string(bytes))
}
