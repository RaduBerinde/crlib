// Copyright 2024 The Cockroach Authors.
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

package fifo

import (
	"math/rand"
	"testing"
)

var pool = MakeQueueBackingPool[int]()

func TestQueue(t *testing.T) {
	q := MakeQueue[int](&pool)
	requireEqual(t, q.PeekFront(), nil)
	requireEqual(t, q.Len(), 0)
	q.PushBack(1)
	q.PushBack(2)
	q.PushBack(3)
	requireEqual(t, q.Len(), 3)
	requireEqual(t, *q.PeekFront(), 1)
	q.PopFront()
	requireEqual(t, *q.PeekFront(), 2)
	q.PopFront()
	requireEqual(t, *q.PeekFront(), 3)
	q.PopFront()
	requireEqual(t, q.PeekFront(), nil)

	for i := 1; i <= 1000; i++ {
		q.PushBack(i)
		requireEqual(t, q.Len(), i)
	}
	for i := 1; i <= 1000; i++ {
		requireEqual(t, *q.PeekFront(), i)
		q.PopFront()
		requireEqual(t, q.Len(), 1000-i)
	}
}

func TestQueueRand(t *testing.T) {
	q := MakeQueue[int](&pool)
	l, r := 0, 0
	for iteration := 0; iteration < 100; iteration++ {
		for n := rand.Intn(100); n > 0; n-- {
			r++
			q.PushBack(r)
			requireEqual(t, q.Len(), r-l)
		}
		for n := rand.Intn(q.Len() + 1); n > 0; n-- {
			l++
			requireEqual(t, *q.PeekFront(), l)
			q.PopFront()
			requireEqual(t, q.Len(), r-l)
		}
	}
}

func requireEqual[T comparable](t *testing.T, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Fatalf("expected %v, but found %v", expected, actual)
	}
}
