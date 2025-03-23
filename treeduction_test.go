package treeduction_test

import (
	"fmt"
	"testing"
	"time"
	"treeduction"
)

// TestBasicReduction tests the basic functionality of the tree reduction.
func TestBasicReduction(t *testing.T) {
	// Create a tree with integer addition as combiner
	tree := treeduction.New(func(a, b int) int {
		return a + b
	}, 10, false, true)
	defer tree.Finish()

	// Create and add input channels
	ch1 := make(chan int, 5)
	ch2 := make(chan int, 5)
	ch3 := make(chan int, 5)
	ch4 := make(chan int, 5)

	tree.Add(ch1, ch2, ch3, ch4)

	// Send values
	ch1 <- 1
	ch2 <- 2
	ch3 <- 3
	ch4 <- 4

	// Close channels to signal completion
	close(ch1)
	close(ch2)
	close(ch3)
	close(ch4)

	// Read the result
	result := <-tree.Output()
	if result != 10 { // 1 + 2 + 3 + 4 = 10
		t.Errorf("Expected result to be 10, got %d", result)
	}
}

// TestLargeInputs tests reduction with a larger number of inputs.
func TestLargeInputs(t *testing.T) {
	tree := treeduction.New(func(a, b int) int {
		return a + b
	}, 10, true, true)

	numInputs := 100
	sum := 0
	var channels []chan int

	// Create channels and compute expected sum
	for i := 0; i < numInputs; i++ {
		ch := make(chan int, 1)
		channels = append(channels, ch)
		value := i + 1
		sum += value
		go func(c chan int, v int) {
			c <- v
			close(c)
		}(ch, value)
	}

	// Convert to read-only channels for the Add method
	readOnlyChannels := make([]<-chan int, numInputs)
	for i, ch := range channels {
		readOnlyChannels[i] = ch
	}

	tree.Add(readOnlyChannels...)

	// Read the result
	if tree.Finish() != nil {
		t.Errorf("Shouldn't happen")
	}

	result := <-tree.Output()
	if result != sum {
		t.Errorf("Expected result to be %d, got %d", sum, result)
	}
}

// TestMultipleOutputs tests that the tree correctly processes multiple outputs.
func TestMultipleOutputs(t *testing.T) {
	tree := treeduction.New(func(a, b string) string {
		return a + b
	}, 10, false, true)
	defer tree.Finish()

	ch1 := make(chan string, 3)
	ch1 <- "Hello, "
	ch1 <- "Goodbye, "
	close(ch1)

	ch2 := make(chan string, 3)
	ch2 <- "World!"
	ch2 <- "Everyone!"
	close(ch2)

	tree.Add(ch1, ch2)

	// Should get two combined results
	result1 := <-tree.Output()
	result2 := <-tree.Output()

	expected := map[string]bool{
		"Hello, World!":      true,
		"Goodbye, Everyone!": true,
	}

	if !expected[result1] || !expected[result2] || result1 == result2 {
		t.Errorf("Unexpected results: %q, %q", result1, result2)
	}
}

// TestWaitForAllBasic tests the basic functionality of waitForAll parameter.
func TestWaitForAllBasic(t *testing.T) {
	// Create a tree with integer addition as combiner and waitForAll=true
	tree := treeduction.New(func(a, b int) int {
		return a + b
	}, 10, true, true)

	// Create and add input channels
	ch1 := make(chan int, 5)
	ch2 := make(chan int, 5)
	ch3 := make(chan int, 5)
	ch4 := make(chan int, 5)

	tree.Add(ch1, ch2, ch3, ch4)

	// Send values
	ch1 <- 1
	ch2 <- 2
	ch3 <- 3
	ch4 <- 4

	// Close channels to signal completion
	close(ch1)
	close(ch2)
	close(ch3)
	close(ch4)

	// Close the tree to trigger the final reduction
	err := tree.Finish()
	if err != nil {
		t.Errorf("Unexpected error from.Finish(): %v", err)
	}

	// Read the result - should be a single output with the total sum
	result := <-tree.Output()
	if result != 10 { // 1 + 2 + 3 + 4 = 10
		t.Errorf("Expected result to be 10, got %d", result)
	}

	// Should be no more values in the output
	select {
	case val, ok := <-tree.Output():
		if ok {
			t.Errorf("Expected closed channel, got value: %v", val)
		}
	default:
		t.Error("Expected closed channel, but it's still open")
	}
}

// TestWaitForAllMultipleValues tests waitForAll with multiple values in different phases.
func TestWaitForAllMultipleValues(t *testing.T) {
	// Create a tree with integer addition as combiner and waitForAll=true
	tree := treeduction.New(func(a, b int) int {
		return a + b
	}, 10, true, false)

	// Phase 1: Initial set of channels
	ch1 := make(chan int, 5)
	ch2 := make(chan int, 5)

	tree.Add(ch1, ch2)

	ch1 <- 10
	ch2 <- 20
	close(ch1)
	close(ch2)

	// Phase 2: Add more channels
	ch3 := make(chan int, 5)
	ch4 := make(chan int, 5)

	tree.Add(ch3, ch4)

	ch3 <- 30
	ch4 <- 40
	close(ch3)
	close(ch4)

	// Close the tree to trigger final reduction
	err := tree.Finish()
	if err != nil {
		t.Errorf("Unexpected error from.Finish(): %v", err)
	}

	// Check result - should be the full sum (10+20+30+40 = 100)
	result := <-tree.Output()
	if result != 100 {
		t.Errorf("Expected result to be 100, got %d", result)
	}

	// Should be no more values
	_, ok := <-tree.Output()
	if ok {
		t.Error("Expected channel to be closed")
	}
}

// TestWaitForAllLargeDataset tests waitForAll with a large number of inputs.
func TestWaitForAllLargeDataset(t *testing.T) {
	tree := treeduction.New(func(a, b int) int {
		return a + b
	}, 10, true, true)

	numInputs := 1000
	sum := 0
	var channels []chan int

	// Create channels and compute expected sum
	for i := 0; i < numInputs; i++ {
		ch := make(chan int, 1)
		channels = append(channels, ch)
		value := i + 1
		sum += value
		go func(c chan int, v int) {
			c <- v
			close(c)
		}(ch, value)
	}

	// Convert to read-only channels for the Add method
	readOnlyChannels := make([]<-chan int, numInputs)
	for i, ch := range channels {
		readOnlyChannels[i] = ch
	}

	tree.Add(readOnlyChannels...)

	// Close to trigger final reduction
	tree.Finish()

	// Read the result - should be a single value with the total sum
	result := <-tree.Output()
	if result != sum {
		t.Errorf("Expected result to be %d, got %d", sum, result)
	}

	// Verify channel is closed
	_, ok := <-tree.Output()
	if ok {
		t.Error("Expected channel to be closed")
	}
}

// TestWaitForAllWithEmptyOutput tests waitForAll behavior when no values are in the output.
func TestWaitForAllWithEmptyOutput(t *testing.T) {
	tree := treeduction.New(func(a, b int) int {
		return a + b
	}, 10, true, true)

	// Create empty channels that will close immediately
	ch1 := make(chan int)
	ch2 := make(chan int)
	close(ch1)
	close(ch2)

	tree.Add(ch1, ch2)

	// Close the tree
	tree.Finish()

	// Channel should be closed, no values
	_, ok := <-tree.Output()
	if ok {
		t.Error("Expected channel to be closed with no values")
	}
}

// TestWaitForAllVsNonWaitForAll compares the behavior of waitForAll true vs false.
func TestWaitForAllVsNonWaitForAll(t *testing.T) {
	// Test with waitForAll=false
	treeNoWait := treeduction.New(func(a, b int) int {
		return a + b
	}, 10, false, false)

	// Test with waitForAll=true
	treeWithWait := treeduction.New(func(a, b int) int {
		return a + b
	}, 10, true, false)

	// Create identical channels for both trees
	for _, tree := range []treeduction.Tree[int]{treeNoWait, treeWithWait} {
		for range 4 {
			ch := make(chan int, 1)
			ch <- 5
			close(ch)
			tree.Add(ch)
		}
	}

	// Let some results accumulate in the output channels
	time.Sleep(100 * time.Millisecond)

	// For non-waitForAll tree, we collect results before closing
	var noWaitSum int
	var noWaitCount int

	// Close both trees
	treeNoWait.Finish()
	treeWithWait.Finish()

	// Read available values
	for val := range treeNoWait.Output() {
		noWaitSum += val
		noWaitCount++
	}

	// For waitForAll tree, we expect a single result with total sum
	waitForAllResult := <-treeWithWait.Output()

	// Both should equal 20 (4 channels with value 5 each)
	if noWaitSum != 20 {
		t.Errorf("Expected noWaitSum to be 20, got %d from %d results", noWaitSum, noWaitCount)
	}

	if waitForAllResult != 20 {
		t.Errorf("Expected waitForAllResult to be 20, got %d", waitForAllResult)
	}

	// Both channels should now be closed
	_, noWaitOk := <-treeNoWait.Output()
	_, waitOk := <-treeWithWait.Output()
	if noWaitOk || waitOk {
		t.Error("Expected both channels to be closed")
	}
}

// Example usage.
func ExampleNew() {
	// Create a tree reducer that concatenates strings
	tree := treeduction.New(func(a, b string) string {
		return a + b
	}, 5, false, true)
	defer tree.Finish()

	// Create input channels
	ch1 := make(chan string, 1)
	ch2 := make(chan string, 1)

	// Add them to the tree
	tree.Add(ch1, ch2)

	// Send values
	ch1 <- "Hello, "
	ch2 <- "World!"

	// Close channels to signal completion
	close(ch1)
	close(ch2)

	// Read and print the result
	fmt.Println(<-tree.Output())
	// Output: Hello, World!
}

// ExampleNewWithWaitForAll demonstrates using waitForAll for a complete reduction.
func ExampleNew_waitForAll() {
	// Create a tree reducer with waitForAll=true
	tree := treeduction.New(func(a, b int) int {
		return a + b
	}, 5, true, true)

	// Create input channels
	channels := make([]chan int, 5)
	for i := range channels {
		channels[i] = make(chan int, 1)
		channels[i] <- i + 1 // Values 1, 2, 3, 4, 5
		close(channels[i])
	}

	// Convert to read-only channels and add them
	readOnlyChannels := make([]<-chan int, 5)
	for i, ch := range channels {
		readOnlyChannels[i] = ch
	}
	tree.Add(readOnlyChannels...)

	// Close the tree to trigger final reduction
	tree.Finish()

	// Read and print the single output (1+2+3+4+5 = 15)
	fmt.Println(<-tree.Output())
	// Output: 15
}
