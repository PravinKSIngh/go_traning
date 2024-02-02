package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

func tailFile(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}
	offset := fileInfo.Size()
	for {
		newInfo, err := file.Stat()
		if err != nil {
			log.Fatal("Error reading file:", err)
			time.Sleep(time.Second)
			continue
		}

		if newInfo.Size() > offset {
			data := make([]byte, newInfo.Size()-offset)
			_, err := file.ReadAt(data, offset)
			if err != nil {
				log.Fatal("Error reading file:", err)
				time.Sleep(time.Second)
				continue
			}
			fmt.Print(string(data))
			offset = newInfo.Size()
		}
		time.Sleep(time.Second)
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run tailf.go [filepath]")
		os.Exit(0)
	}

	filePath := os.Args[1]

	tailFile(filePath)
}
