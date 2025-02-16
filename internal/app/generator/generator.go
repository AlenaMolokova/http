package generator

import (
	"math/rand"
	"time"
)

type Generator interface {
	Generate() string
}

type SimpleGenerator struct {
	letters string
	length  int
	rnd     *rand.Rand
}

func NewGenerator(length int) Generator {
	return &SimpleGenerator{
		letters: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
		length:  length,
		rnd:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (g *SimpleGenerator) Generate() string {
	id := make([]byte, g.length)
	for i := range id {
		id[i] = g.letters[g.rnd.Intn(len(g.letters))]
	}
	return string(id)
}
