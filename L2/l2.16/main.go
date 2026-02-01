package main

import (
	"fmt"
	"mywget/pkg/downloader"
	"os"
	"strconv"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Использование: mywget <URL> [глубина]")
		os.Exit(1)
	}

	url := os.Args[1]
	depth := 1
	if len(os.Args) >= 3 {
		d, _ := strconv.Atoi(os.Args[2])
		depth = d
	}

	err := downloader.Load(url, depth)
	if err != nil {
		fmt.Printf("Загрузка %s завершилась с ошибкой: %v\n", url, err)
	} else {
		fmt.Printf("Загрузка %s завершилась успешно.\n", url)
	}
}
