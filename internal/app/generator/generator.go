package generator

import (
	"math/rand"
	"sync"
	"time"
)

// Generator определяет интерфейс для генерации случайных коротких идентификаторов.
// Он используется для создания уникальных идентификаторов для сокращенных URL.
type Generator interface {
	// Generate создает новый уникальный идентификатор.
	//
	// Возвращает:
	//   - string: сгенерированный идентификатор
	Generate() string
}

// SimpleGenerator реализует интерфейс Generator для создания случайных строк
// с использованием алфавитно-цифровых символов.
type SimpleGenerator struct {
	letters   string     // Набор символов для генерации
	length    int        // Длина генерируемых идентификаторов
	rnd       *rand.Rand // Генератор случайных чисел
	mu        sync.Mutex // Мьютекс для синхронизации доступа к генератору
	bufferLen int        // Длина внутреннего буфера
	buffer    []byte     // Буфер для создания строки
}

// NewGenerator создает и инициализирует новый экземпляр генератора с заданной длиной идентификатора.
//
// Параметры:
//   - length: желаемая длина генерируемых идентификаторов
//
// Возвращает:
//   - Generator: новый экземпляр генератора
func NewGenerator(length int) Generator {
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
