package main

import (
	"bufio"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const chunkSize = 1024

var (
	filePath = "test.log" // Replace with the path to your log file
	mutex    sync.Mutex
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type Client struct {
	conn         *websocket.Conn
	updateTicker *time.Ticker
}

var clients = make(map[*Client]struct{})
var clientsMutex sync.Mutex

func handleTail(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.ParseFiles("./rough/template/index.html")

	if err != nil {
		http.Error(w, "Internal Server Error - Template", http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

	tmpl.Execute(w, nil)

	// Create a new client
	client := &Client{
		conn:         nil, // Set to actual connection when WebSocket is established
		updateTicker: time.NewTicker(1 * time.Second),
	}

	// Register the client
	clientsMutex.Lock()
	clients[client] = struct{}{}
	clientsMutex.Unlock()

	// Flush the response to initiate a connection to the client
	flusher.Flush()

	// Listen for updates and send them to the client
	for {
		select {
		case <-client.updateTicker.C:
			// Ping the client to check if the connection is still alive
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				// Client disconnected
				clientsMutex.Lock()
				delete(clients, client)
				clientsMutex.Unlock()
				return
			}
		case <-r.Context().Done():
			// Client disconnected
			clientsMutex.Lock()
			delete(clients, client)
			clientsMutex.Unlock()
			return
		}
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Create a new client
	client := &Client{
		conn:         conn,
		updateTicker: time.NewTicker(1 * time.Second),
	}

	// Register the client
	clientsMutex.Lock()
	clients[client] = struct{}{}
	clientsMutex.Unlock()

	// Associate the connection with the client
	client.conn = conn

	// Continuously send updates to the client
	for {
		select {
		case <-client.updateTicker.C:
			// Ping the client to check if the connection is still alive
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				// Client disconnected
				clientsMutex.Lock()
				delete(clients, client)
				clientsMutex.Unlock()
				return
			}
		case <-r.Context().Done():
			// Client disconnected
			clientsMutex.Lock()
			delete(clients, client)
			clientsMutex.Unlock()
			return
		}
	}
}

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
			// Update all connected clients with the new line
			clientsMutex.Lock()
			for client := range clients {
				err := client.conn.WriteMessage(websocket.TextMessage, data)
				if err != nil {
					// Client disconnected
					client.updateTicker.Stop()
					delete(clients, client)
				}
			}
			clientsMutex.Unlock()
			offset = newInfo.Size()
		}
		time.Sleep(time.Second)
	}
}

func tFile() {
	lines, err := getLastNLines(filePath, 4)
	if err != nil {
		fmt.Print("Error reading file:", err)
	}
	// Update all connected clients with the new line
	clientsMutex.Lock()
	for client := range clients {
		err := client.conn.WriteMessage(websocket.TextMessage, []byte(lines))
		if err != nil {
			// Client disconnected
			client.updateTicker.Stop()
			delete(clients, client)
		}
	}
	clientsMutex.Unlock()
	tailFile(filePath)
}

func setupRoutes() {
	http.HandleFunc("/tail", handleTail)
	http.HandleFunc("/ws", handleWebSocket)
}

func main() {
	go tFile()

	setupRoutes()

	log.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
