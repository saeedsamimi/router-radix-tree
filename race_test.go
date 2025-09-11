package radix_test

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

	radix "github.com/saeedsamimi/router-radix-tree"
)

func TestRaceHeavy(t *testing.T) {
	t.Parallel()

	tree := radix.NewRadixTree()

	// Preload some routes
	base := [][]string{
		{"api"},
		{"api", "v1"},
		{"api", "v1", "users"},
		{"files", "*filepath"},
		{"admin", "*path"},
		{"users", ":id"},
	}
	for _, p := range base {
		_, _ = tree.Add(p, "handler")
	}

	workers := runtime.GOMAXPROCS(0) * 4
	iters := 2000
	var wg sync.WaitGroup
	wg.Add(workers)

	stop := make(chan struct{})

	// Readers
	for i := 0; i < workers/2; i++ {
		go func(id int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)))
			paths := [][]string{
				{"api", "v1", "users"},
				{"files", "docs", "readme.md"},
				{"admin", "dashboard"},
				{"users", fmt.Sprintf("%d", id)},
			}
			for j := 0; j < iters; j++ {
				select {
				case <-stop:
					return
				default:
				}
				_ = tree.Get(paths[r.Intn(len(paths))])
			}
		}(i)
	}

	// Writers (adds on random paths)
	for i := 0; i < workers/2; i++ {
		go func(id int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)<<16))
			for j := 0; j < iters; j++ {
				select {
				case <-stop:
					return
				default:
				}
				s := fmt.Sprintf("%d", r.Intn(100))
				switch r.Intn(3) {
				case 0:
					_, _ = tree.Add([]string{"api", "dyn", s}, "h")
				case 1:
					_, _ = tree.Add([]string{"profile", ":user", s}, "h")
				case 2:
					_, _ = tree.Add([]string{"files", s}, "h")
				}
			}
		}(i)
	}

	wg.Wait()
	close(stop)
}
