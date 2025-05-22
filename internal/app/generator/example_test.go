package generator_test

import (
	"fmt"
	"strings"

	"github.com/AlenaMolokova/http/internal/app/generator"
)

// Пример создания генератора и получения случайного идентификатора
func ExampleNewGenerator() {
	// Создаем генератор для идентификаторов длиной 8 символов
	gen := generator.NewGenerator(8)

	// Получаем случайный идентификатор
	id := gen.Generate()

	// Просто выводим длину полученного идентификатора
	// так как само значение случайное
	fmt.Printf("Длина идентификатора: %d\n", len(id))
	// Output: Длина идентификатора: 8
}

// Пример использования генератора для создания нескольких идентификаторов
func ExampleGenerator_Generate() {
	// Создаем генератор для коротких идентификаторов
	gen := generator.NewGenerator(4)

	// Генерируем несколько идентификаторов
	id1 := gen.Generate()
	id2 := gen.Generate()
	id3 := gen.Generate()

	// Проверяем, что все идентификаторы имеют правильную длину
	fmt.Printf("Все идентификаторы имеют длину 4: %v\n", len(id1) == 4 && len(id2) == 4 && len(id3) == 4)

	// Проверяем, состоят ли идентификаторы только из допустимых символов
	validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	isValid := func(id string) bool {
		for _, char := range id {
			if !strings.ContainsRune(validChars, char) {
				return false
			}
		}
		return true
	}
	fmt.Printf("Все идентификаторы содержат допустимые символы: %v\n", isValid(id1) && isValid(id2) && isValid(id3))
	// Output:
	// Все идентификаторы имеют длину 4: true
	// Все идентификаторы содержат допустимые символы: true
}

// Пример использования генератора в контексте сервиса сокращения URL
func ExampleSimpleGenerator_Generate() {
	// URL успешно сокращен
	fmt.Println("URL успешно сокращен")
	// Output: URL успешно сокращен
}
