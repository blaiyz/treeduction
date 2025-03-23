package treeduction

import (
	"context"
	"sync"
)

type tree[T any] struct {
	combiner   func(f T, s T) T
	roots      []<-chan T
	bufSize    int
	output     chan T
	stop       chan struct{}
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	waitForAll bool
	ordered    bool
}

type Tree[T any] interface {
	Add(out ...<-chan T)
	Output() <-chan T
	Finish() error
}

func New[T any](combiner func(f T, s T) T, bufferSize int, waitForAll bool, ordered bool) Tree[T] {
	ctx, cancel := context.WithCancel(context.Background())
	return &tree[T]{
		combiner:   combiner,
		roots:      make([]<-chan T, 20),
		bufSize:    bufferSize,
		output:     make(chan T, bufferSize),
		stop:       make(chan struct{}),
		ctx:        ctx,
		cancel:     cancel,
		waitForAll: waitForAll,
		ordered:    ordered,
	}
}

func (t *tree[T]) Add(out ...<-chan T) {
	for _, o := range out {
		c := make(chan T, t.bufSize)

		// Wraping <-o in a select which checks for ctx.Done()
		go func(o <-chan T) {
		loop:
			for {
				select {
				case v, ok := <-o:
					if !ok {
						break loop
					}
					c <- v
				case <-t.ctx.Done():
					break loop
				}
			}
			close(c)
		}(o)

		t.addOne(c, 0)
	}
	// Update the root receivers
	t.updateCollectors()
}

func (t *tree[T]) Output() <-chan T {
	return t.output
}

func (t *tree[T]) Finish() error {
	if !t.waitForAll {
		t.cancel()
		t.wg.Wait()
		close(t.output)
		return nil
	}

	// WaitForAll assumes that inputs should eventually stop (and channels closed)
	t.wg.Wait()
	t.cancel()

	select {
	case final := <-t.output:
	s:
		for {
			select {
			case v := <-t.output:
				final = t.combiner(final, v)
			default:
				break s
			}
		}
		t.output <- final
	default:
	}
	close(t.output)
	return nil
}

func (t *tree[T]) updateCollectors() {
	// Stop the previous select goroutings
	close(t.stop)
	t.stop = make(chan struct{})

	for _, ch := range t.roots {
		if ch == nil {
			continue
		}

		t.wg.Add(1)
		go func(c <-chan T) {
		Inner:
			for {
				select {
				case <-t.stop:
					break Inner
				case v, ok := <-c:
					if !ok {
						break Inner
					}
					t.output <- v
				}
			}
			t.wg.Done()
		}(ch)
	}
}

func (t *tree[T]) addOne(root <-chan T, level int) {
	// Extend the slice to the level
	for i := len(t.roots); i <= level; i++ {
		t.roots = append(t.roots, nil)
	}

	if t.roots[level] == nil {
		t.roots[level] = root
		return
	}

	prev := t.roots[level]
	t.roots[level] = nil
	var c <-chan T
	if t.ordered {
		c = t.orderedNode(prev, root)
	} else {
		c = t.unorderedNode(prev, root)
	}
	t.addOne(c, level+1)
}

func (t *tree[T]) unorderedNode(f <-chan T, s <-chan T) <-chan T {
	c := make(chan T, t.bufSize)
	go func() {
		fanIn := make(chan T, t.bufSize)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			for v := range f {
				fanIn <- v
			}
			wg.Done()
		}()

		go func() {
			for v := range s {
				fanIn <- v
			}
			wg.Done()
		}()

		go func() {
			wg.Wait()
			close(fanIn)
		}()

		for {
			v1, ok := <-fanIn
			if !ok {
				break
			}

			v2, ok := <-fanIn
			if !ok {
				c <- v1
				break
			}
			c <- t.combiner(v1, v2)
		}

		close(c)
	}()

	return c
}

func (t *tree[T]) orderedNode(f <-chan T, s <-chan T) <-chan T {
	c := make(chan T, t.bufSize)
	go func() {
		for {
			v1, ok := <-f
			if !ok {
				break
			}

			v2, ok := <-s
			if !ok {
				c <- v1
				break
			}

			c <- t.combiner(v1, v2)
		}
		close(c)
	}()

	return c
}
