package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"time"
)

const chunkSize = 1024

func getLastNLines(filePath string, n int) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Determine the positions of the line breaks in the file
	lineBreakPositions, err := findLineBreakPositions(file)
	if err != nil {
		return "", err
	}
	// Read the last N lines based on the line break positions
	lines, err := readLastNLines(file, lineBreakPositions, n)
	if err != nil {
		return "", err
	}

	return lines, nil
}

func findLineBreakPositions(file *os.File) ([]int64, error) {
	var positions []int64

	scanner := bufio.NewScanner(file)
	position := int64(0)

	for scanner.Scan() {
		positions = append(positions, position)
		position += int64(len(scanner.Bytes())) + 1 // Account for the newline character
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return positions, nil
}

func readLastNLines(file *os.File, lineBreakPositions []int64, n int) (string, error) {
	startIndex := len(lineBreakPositions) - n
	if startIndex < 0 {
		startIndex = 0
	}
	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}
	_, err = file.Seek(lineBreakPositions[startIndex], 0)
	if err != nil {
		return "", err
	}
	data := make([]byte, fileInfo.Size()-lineBreakPositions[startIndex])
	_, err = file.ReadAt(data, lineBreakPositions[startIndex])
	if err != nil {
		return "", err
	}
	return string(data), nil
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
	if err != nil {
		fmt.Print("Error reading file:", err)
	}
	fmt.Println(lines)
	tailFile(filePath)
}
