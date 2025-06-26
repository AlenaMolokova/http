// Package generator содержит логику генерации сокращённых URL и вспомогательные функции.
package generator

import (
	"math/rand"
	"sync"
	"time"
)

// SimpleGenerator реализует генерацию случайных строк
// с использованием алфавитно-цифровых символов.
// Структура потокобезопасна благодаря использованию мьютекса.
type SimpleGenerator struct {
	letters string     // Набор символов для генерации
	length  int        // Длина генерируемых идентификаторов
	rnd     *rand.Rand // Генератор случайных чисел
	mu      sync.Mutex // Мьютекс для синхронизации доступа к генератору
	buffer  []byte     // Буфер для формирования случайной строки
}

// New создает и инициализирует новый экземпляр генератора с заданной длиной идентификатора.
// Функция переименована с NewGenerator на New для соответствию Go convention.
//
// Параметры:
//   - length: желаемая длина генерируемых идентификаторов
//
// Возвращает:
//   - *SimpleGenerator: новый экземпляр генератора
func New(length int) *SimpleGenerator {
	return &SimpleGenerator{
		letters: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
		length:  length,
		rnd:     rand.New(rand.NewSource(time.Now().UnixNano())),
		buffer:  make([]byte, length),
	}
}

// Generate создает случайную строку из алфавитно-цифровых символов.
// Метод потокобезопасен благодаря использованию мьютекса.
//
// Возвращает:
//   - string: случайная строка заданной длины
func (g *SimpleGenerator) Generate() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	for i := range g.buffer {
		g.buffer[i] = g.letters[g.rnd.Intn(len(g.letters))]
	}
	return string(g.buffer)
}
