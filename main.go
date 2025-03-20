// main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"github.com/joho/godotenv"
)

func main() {
	parseFlag := flag.Bool("parse", false, "Запустить парсинг данных")
	seleniumFlag := flag.Bool("selenium", false, "Запустить скачивание файлов")
	emailFlag := flag.Bool("email", false, "Запустить обработку файлов")

	flag.Parse()

	if *parseFlag {
		startParsing()
	} else if *seleniumFlag {
		startSelenium()
	} else if *emailFlag {
		startEmail()
	} else {
		fmt.Println("Укажите один из флагов: -parse, -selenium, -email")
	}

	loadEnv()
}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка загрузки .env файла")
	}
}