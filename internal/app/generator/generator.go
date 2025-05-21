package generator

import (
	"math/rand"
	"sync"
	"time"
)

type Generator interface {
	Generate() string
}

type SimpleGenerator struct {
	letters   string
	length    int
	rnd       *rand.Rand
	mu        sync.Mutex
	bufferLen int
	buffer    []byte
}

func NewGenerator(length int) Generator {
	return &SimpleGenerator{
		letters: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
		length:  length,
		rnd:     rand.New(rand.NewSource(time.Now().UnixNano())),
		buffer:  make([]byte, length),
	}
}

func (g *SimpleGenerator) Generate() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	for i := range g.buffer {
		g.buffer[i] = g.letters[g.rnd.Intn(len(g.letters))]
	}
	return string(g.buffer)
}
