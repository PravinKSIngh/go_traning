package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

const chunkSize = 1000

func getLastNLines(filePath string, n int) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var lines []string
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := fileInfo.Size()
	seakPosition := fileSize - 1

	var newLinepos int
	var line string
	for i := 0; i < n; {
		startProsition := seakPosition - int64(chunkSize) - 1
		if startProsition < 0 {
			startProsition = 0
		}
		_, err := file.Seek(startProsition, 0)
		if err != nil {
			return nil, err
		}

		data := make([]byte, chunkSize)
		_, err = file.Read(data)
		if err != nil {
			return nil, err
		}
		newLinepos = len(data)
		for j := len(data) - 1; j >= 0; j-- {
			if data[j] == '\n' {
				line = string(data[j+1:newLinepos]) + line
				lines = append([]string{line}, lines...)
				fmt.Println(line, "LINEEND")
				line = ""
				newLinepos = j
				i++
			}
		}
		line = string(data[:newLinepos])

		fmt.Println(line, "LINEx")
		seakPosition = startProsition - 1
		if seakPosition < 0 {
			break // Reached the beginning of the file
		}
	}
	return lines, nil
}

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
	lines, err := getLastNLines(filePath, 4)
	//fmt.Print(lines)
	if err != nil {
		fmt.Print("Error reading file:", err)
	}
	for _, line := range lines {
		fmt.Print(line)
	}
	tailFile(filePath)
}
