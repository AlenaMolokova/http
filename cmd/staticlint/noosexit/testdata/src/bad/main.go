package main

import "os"

func main() {
	if true {
		//lint:ignore noosexit Тестовый файл для проверки анализатора
		os.Exit(1) // want "прямой вызов os.Exit в функции main запрещен"
	}
}
