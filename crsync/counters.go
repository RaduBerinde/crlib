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
	"sync/atomic"
	"unsafe"
)

type Counter struct {
	c Counters
}

func MakeCounter() Counter {
	return Counter{
		c: MakeCounters(1),
	}
}

func (c *Counter) Add(delta int64) {
	c.c.Add(0, delta)
}

func (c *Counter) Get() int64 {
	return c.c.Get(0)
}

// Counters is a sharded set of counters that can be incremented concurrently.
type Counters struct {
	numShards uint32
	// shardSize is the number of counters per shard.
	shardSize   uint32
	counters    []atomic.Int64
	numCounters int
}

// Number of counters per cacheline. We assume the typical 64-byte cacheline.
// Must be a power of 2.
const countersPerCacheline = 8

// MakeCounters creates a new Counters with the specified number of counters.
func MakeCounters(numCounters int) Counters {
	return makeCounters(NumShards(), numCounters)
}

func makeCounters(numShards, numCounters int) Counters {
	// Round up to the nearest cacheline size, to avoid false sharing.
	shardSize := (numCounters + countersPerCacheline - 1) &^ (countersPerCacheline - 1)
	counters := make([]atomic.Int64, shardSize*numShards+countersPerCacheline)
	// Align the slice to a cache line.
	if r := (uintptr(unsafe.Pointer(&counters[0])) / 8) & (countersPerCacheline - 1); r != 0 {
		counters = counters[countersPerCacheline-r:]
	}
	return Counters{
		numShards:   uint32(numShards),
		shardSize:   uint32(shardSize),
		counters:    counters,
		numCounters: numCounters,
	}
}

// Add a delta to the specified counter.
func (c *Counters) Add(counter int, delta int64) {
	shard := uint32(CPUBiasedInt()) % c.numShards
	c.counters[shard*c.shardSize+uint32(counter)].Add(delta)
}

// Get the current value of the specified counter.
func (c *Counters) Get(counter int) int64 {
	var res int64
	for shard := range c.numShards {
		res += c.counters[shard*c.shardSize+uint32(counter)].Load()
	}
	return res
}

// All returns the current values of all counters. There are no ordering
// guarantees with respect to concurrent updates.
func (c *Counters) All() []int64 {
	res := make([]int64, c.numCounters)
	for i := range c.numShards {
		start := int(i * c.shardSize)
		for j := range res {
			res[j] += c.counters[start+j].Load()
		}
	}
	return res
}
