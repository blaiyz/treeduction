Small library that implements a parallel reduction tree in Go.

## How to use
The tree is managed using the `Tree` object. The input of the tree (the leaves) is accepted as channels. The channels provide the output of some parallel tasks, which can be combined using a `combine` function. For example, one can create a sum reduction tree like so:
```go
// Create a tree with integer addition as combiner
tree := treeduction.New(func(a, b int) int {
    return a + b
}, 10, false, false)
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

result := <-tree.Output()
fmt.Println("Result: %d", result) // Should be 10
```

Now, the constructor accepts a few parameters:
```go
func New[T any](combiner func(f T, s T) T, bufferSize int, waitForAll bool, ordered bool) Tree[T];
```
Combiner is, as the name suggests, a function that performs the reduction (like the example above). bufferSize is the channel size to use for the channels created by the tree.

#### `waitForAll`
Set this to true if you want a single output out of `tree.Output()` instead of accepting multiple intermediary results. Use `tree.Finish()` to complete the reduction before reading `tree.Output()` when using this parameter. `tree.Finish()` closes all the channels created by the tree.

#### `ordered`
Each tree node combines results from its child nodes as soon as it has the 2 results.
If this is set to false, the node may combine 2 results from the **same** child node.
If this is set to true, the node would wait for a result from **both** children when combining.
Set this to true if you care about order of the results.
> [!WARNING]
> When this is set to true, make sure to add all channels in a single call to `tree.Add()` (it's variadic), otherwise you could run into deadlocks. Also, note that all channels should output the same number of results, otherwise the tree would wait for the other child node's nonexistent result (and that would cause a deadlock).

