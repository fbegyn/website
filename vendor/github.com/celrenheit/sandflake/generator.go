/*
Copyright 2017 Salim Alami

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

*/

package sandflake

import (
	securerandom "crypto/rand"
	"io"
	unsaferandom "math/rand"
	"sync"
	"time"
)

type Generator struct {
	mu       sync.Mutex
	workerID WorkerID
	lastTime time.Time
	sequence uint32
	once     sync.Once
	reader   io.Reader
	clock    clock
}

// Next returns the next id.
// It returns an error if New() fails.
// It is safe for concurrent use.
func (g *Generator) Next() ID {
	g.once.Do(func() {
		g.workerID = newWorkerID()
		g.reader = unsaferandom.New(unsaferandom.NewSource(time.Now().UnixNano()))
		if g.clock == nil {
			g.clock = stdClock{}
		}
	})

	g.mu.Lock()
	now := g.clock.Now().UTC()

	if now.UnixNano()/timeUnit == g.lastTime.UnixNano()/timeUnit {
		g.sequence++
	} else {
		g.lastTime = now
		g.sequence = 0
	}

	if g.sequence > maxSequence {
		// reset sequence
		g.sequence = 0
	}

	wid := g.workerID
	seq := g.sequence
	g.mu.Unlock()

	randomBytes := generateRandomBytes(g.reader)

	return NewID(now, wid, seq, randomBytes)
}

type FixedTimeGenerator struct {
	mu       sync.Mutex
	workerID WorkerID
	time     time.Time
	sequence uint32
	once     sync.Once
	reader   io.Reader
}

func NewFixedTimeGenerator(t time.Time) *FixedTimeGenerator {
	return &FixedTimeGenerator{
		time:     t,
		workerID: newWorkerID(),
		reader:   unsaferandom.New(unsaferandom.NewSource(time.Now().UnixNano())),
	}
}

// Next returns the next id.
// It returns an error if New() fails.
// It is safe for concurrent use.
func (g *FixedTimeGenerator) Next() ID {
	g.mu.Lock()
	g.sequence++

	if g.sequence > maxSequence {
		// reset sequence
		g.sequence = 0
	}

	wid := g.workerID
	seq := g.sequence
	g.mu.Unlock()

	randomBytes := generateRandomBytes(g.reader)

	return NewID(g.time, wid, seq, randomBytes)
}

type clock interface {
	Now() time.Time
}

type stdClock struct{}

func (c stdClock) Now() time.Time { return time.Now() }

type mockClock time.Time

func (t mockClock) Now() time.Time { return time.Time(t) }

func generateRandomBytes(unsafereader io.Reader) []byte {
	randomBytes := make([]byte, randomLength)
	// try crypto rand reader
	if _, err := io.ReadFull(securerandom.Reader, randomBytes); err != nil {
		// otherwise fallback to math/crypto reader
		io.ReadFull(unsafeReader, randomBytes)
	}

	return randomBytes
}

var unsafeReader = unsaferandomReader{}

type unsaferandomReader struct{}

func (_ unsaferandomReader) Read(p []byte) (int, error) {
	return unsaferandom.Read(p)
}
