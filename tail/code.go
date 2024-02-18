package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

const chunkSize = 100

func getLastNLinesPosition(filePath string, n int) (int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var lineCounter int64 = 0
	fileInfo, err := file.Stat()
	if err != nil {
		return 0, err
	}
	fileSize := fileInfo.Size()
	seekPosition := fileSize - 1

	// Read chunks backward until we have enough lines or reach the beginning
	for i := 0; i < n; {
		// Calculate the start position for the current chunk
		startPosition := seekPosition - int64(chunkSize) + 1
		if startPosition < 0 {
			startPosition = 0
		}

		// Set the seek position to the start of the current chunk
		_, err := file.Seek(startPosition, 0)
		if err != nil {
			return 0, err
		}

		// Read the current chunk
		chunk := make([]byte, chunkSize)
		_, err = file.Read(chunk)
		if err != nil {
			return 0, err
		}

		// Process the chunk backward to find line breaks
		for j := chunkSize - 1; j >= 0; j-- {
			if chunk[j] == '\n' {
				lineCounter++
				if lineCounter == int64(n) {
					return startPosition + int64(j), nil
				}
				i++
			}
		}

		seekPosition = startPosition - 1
		if seekPosition < 0 {
			break // Reached the beginning of the file
		}
	}

	return 0, nil
}
func readLastNLines(file *os.File, lineBreakPositions int64) (string, error) {
	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}
	lineBreakPositions = lineBreakPositions + 1
	_, err = file.Seek(lineBreakPositions, 0)
	if err != nil {
		return "", err
	}
	data := make([]byte, fileInfo.Size()-lineBreakPositions)
	_, err = file.ReadAt(data, lineBreakPositions)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
func getLastNLines(filePath string, n int) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Determine the positions of the line breaks in the file
	lineBreakPositions, err := getLastNLinesPosition(filePath, 4)
	if err != nil {
		return "", err
	}
	// Read the last N lines based on the line break positions
	lines, err := readLastNLines(file, lineBreakPositions)
	if err != nil {
		return "", err
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
		fmt.Println("Usage: go run main.go file.log")
		os.Exit(1)
	}

	filePath := os.Args[1]
	line, err := getLastNLines(filePath, 4)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(line)
	tailFile(filePath)
}
