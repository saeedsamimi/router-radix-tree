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

type wrapperPool struct {
	mu   sync.Mutex
	recs []*radix.NodeWrapper
}

func (p *wrapperPool) add(nw *radix.NodeWrapper) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if nw == nil {
		return
	}

	p.recs = append(p.recs, nw)
}

func TestRaceHeavy(t *testing.T) {
	t.Parallel()

	tree := radix.NewRadixTree()
	pool := &wrapperPool{}

	// Preload some routes
	base := [][]string{
		{"api"},
		{"api", "v1"},
		{"api", "v1", "users"},
		{"files", "*filepath"},
		{"admin", "*path"},
		{"users", ":id"},
	}
	for _, pth := range base {
		if nw, _ := tree.Add(pth, "handler"); nw != nil {
			pool.add(nw)
		}
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
			for range iters {
				select {
				case <-stop:
					return
				default:
				}
				_ = tree.Get(paths[r.Intn(len(paths))])
			}
		}(i)
	}

	// Writers (adds and deletes on random paths)
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
				switch r.Intn(4) {
				case 0:
					if nw, _ := tree.Add([]string{"api", "dyn", s}, "h"); nw != nil {
						pool.add(nw)
					}
				case 1:
					if nw, _ := tree.Add([]string{"profile", ":user", s}, "h"); nw != nil {
						pool.add(nw)
					}
				case 2:
					if nw, _ := tree.Add([]string{"files", s}, "h"); nw != nil {
						pool.add(nw)
					}
				case 3:
					pool.mu.Lock()
					nw := pool.recs[r.Intn(len(pool.recs))]
					if err := tree.Delete(nw.Path()); err != nil {
						// assert.Fail(t, "Delete failed: %v", err)
					}
					pool.mu.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()
	close(stop)
}
