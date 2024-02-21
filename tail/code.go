package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	filePath      = "test.log" // Replace with the path to your log file
	updateChannel = make(chan string, 100)
	mutex         sync.Mutex
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
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

			mutex.Lock()
			updateChannel <- string(data)
			mutex.Unlock()
			//fmt.Print(string(data))
			offset = newInfo.Size()
		}
		time.Sleep(time.Second)
	}
}
func handleTail(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	line, _ := getLastNLines(filePath, 4)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.New("tail").Parse(fmt.Sprintf("<!DOCTYPE html>"+
		"	<html>"+
		"	<head>"+
		"		<title>Log Tail</title>"+
		"		"+
		"	<script type=\"application/javascript\">"+
		"	window.onload = function () {"+
		"		var logElement = document.getElementById(\"log\");"+
		"	"+
		"		function updateLog(line) {"+
		"			logElement.innerHTML += line + \"<br>\";"+
		"			logElement.scrollTop = logElement.scrollHeight;"+
		"		}"+
		"	"+
		"		var socket = new WebSocket(\"ws://localhost:8080/ws\");"+
		"	"+
		"		socket.onmessage = function(event) {"+
		"			updateLog(event.data);"+
		"		};"+
		"	"+
		"		socket.onclose = function(event) {"+
		"			console.error(\"WebSocket closed unexpectedly:\", event);"+
		"			setTimeout(function() {"+
		"				var newSocket = new WebSocket(\"ws://localhost:8080/ws\");"+
		"				newSocket.onmessage = socket.onmessage;"+
		"				newSocket.onclose = socket.onclose;"+
		"				socket = newSocket;"+
		"			}, 1000);"+
		"		};"+
		"	};"+
		"	"+
		"	</script>"+
		"	</head>"+
		"	<body>"+
		"		<pre id=\"log\">%s</pre>"+
		"	</body>"+
		"	</html>", line))

	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, nil)

	// Flush the response to initiate a connection to the client
	flusher.Flush()
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	go tailFile(filePath)
	// Continuously send updates to the client
	for {
		select {
		case line := <-updateChannel:
			err := conn.WriteMessage(websocket.TextMessage, []byte(line))
			if err != nil {
				log.Println(err)
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}

func main() {

	http.HandleFunc("/tail", handleTail)
	http.HandleFunc("/ws", handleWebSocket)

	log.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

}
