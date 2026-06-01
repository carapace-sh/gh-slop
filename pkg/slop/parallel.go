package slop

import "sync"

// parallelMap runs fn on each item concurrently (max concurrency), collecting results and errors.
func parallelMap[T any, R any](items []T, concurrency int, fn func(T) (R, error)) ([]R, error) {
	type indexedResult struct {
		result R
		err    error
		index  int
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	results := make(chan indexedResult, len(items))

	for i, item := range items {
		wg.Add(1)
		go func(i int, item T) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			r, err := fn(item)
			results <- indexedResult{r, err, i}
		}(i, item)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	rs := make([]R, len(items))
	for res := range results {
		if res.err != nil {
			return nil, res.err
		}
		rs[res.index] = res.result
	}
	return rs, nil
}
