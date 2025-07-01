// Package main содержит пример корректного кода без прямых вызовов os.Exit в функции main.
package main

import (
	"log"
	"os"
)

// main выполняет основную логику программы, используя log.Fatal для обработки ошибок.
func main() {
	if err := run(); err != nil {
		log.Fatal(err) // OK: log.Fatal вместо os.Exit
	}
}

// run имитирует выполнение программы.
// Может содержать os.Exit, так как это не функция main.
func run() error {
	os.Exit(1) // OK: os.Exit разрешен вне функции main
	return nil
}

// helper демонстрирует использование os.Exit в некритических функциях.
func helper() {
	os.Exit(1) // OK: os.Exit разрешен в других функциях
}
