// Copyright 2025 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package crsync

import (
	"fmt"
	"math/rand/v2"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

func runCountersBenchmark(
	b *testing.B, numCounters, parallelism int, incCounter func(counter int),
) {
	// Each element of ch corresponds to a batch of operations to be performed.
	ch := make(chan int, 1000)

	var wg sync.WaitGroup
	for range parallelism {
		wg.Add(1)
		go func() {
			defer wg.Done()

			rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
			for numOps := range ch {
				for range numOps {
					incCounter(rng.IntN(numCounters))
				}
			}
		}()
	}

	const batchSize = 1000
	numOps := int64(b.N) * int64(parallelism)
	for i := int64(0); i < numOps; i += batchSize {
		ch <- int(min(batchSize, numOps-i))
	}
	close(ch)
	wg.Wait()
}

func BenchmarkCounters(b *testing.B) {
	forEach := func(b *testing.B, fn func(b *testing.B, c, p int)) {
		for _, c := range []int{1, 10, 100} {
			for _, p := range []int{1, 4, runtime.GOMAXPROCS(0), 4 * runtime.GOMAXPROCS(0)} {
				b.Run(fmt.Sprintf("c=%d/p=%d", c, p), func(b *testing.B) {
					fn(b, c, p)
				})
			}
		}
	}

	// simple uses non-sharded atomic counters.
	b.Run("simple", func(b *testing.B) {
		forEach(b, func(b *testing.B, c, p int) {
			counters := make([]atomic.Int64, c)
			incCounter := func(counter int) {
				counters[counter].Add(1)
			}
			runCountersBenchmark(b, c, p, incCounter)
		})
	})

	// randshards uses a 4*N shards with random shard choice.
	b.Run("randshards", func(b *testing.B) {
		forEach(b, func(b *testing.B, c, p int) {
			counters := makeCounters(runtime.GOMAXPROCS(0)*4, c)
			incCounter := func(counter int) {
				shard := rand.Uint32N(counters.numShards)
				counters.counters[shard*counters.shardSize+uint32(counter)].Add(1)
			}
			runCountersBenchmark(b, c, p, incCounter)
		})
	})

	b.Run("crsync", func(b *testing.B) {
		forEach(b, func(b *testing.B, c, p int) {
			counters := MakeCounters(c)
			incCounter := func(counter int) {
				counters.Add(counter, 1)
			}
			runCountersBenchmark(b, c, p, incCounter)
		})
	})
}
