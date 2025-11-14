package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
)

func clearText(data []byte) string {
	text := string(data)

	text = strings.ToLower(text)

	text = strings.Map(func(r rune) rune {
		if r == ' ' || (r >= 'а' && r <= 'я') || r == 'ё' {
			return r
		}
		if r == '\n' || r == '-' {
			return ' '
		}
		return -1
	}, text)

	return text
}

func makingMap(words []string) map[string]int {
	m := make(map[string]int, len(words))

	for _, v := range words {
		m[v] = len([]rune(v))
	}

	return m
}

func writeJSON(m map[string]int) {
	jsonData, err := json.MarshalIndent(m, "", " ")
	if err != nil {
		log.Println(err)
	}

	file, err := os.Create("words.json")
	if err != nil {
		log.Println(err)
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		log.Println(err)
	}
}

func main() {
	// Считываю файл целиком
	data, err := os.ReadFile("words.txt")
	if err != nil {
		log.Println(err)
	}

	// Строка из всего файла
	text := clearText(data)

	// Разбиваем строку на массив из слов
	words := strings.Fields(text)

	// Map из всех слов в формате слово: его длина
	m := makingMap(words)

	// Создание готового файла
	writeJSON(m)
}
