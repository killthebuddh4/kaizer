package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
)

// Define an upgrader to upgrade HTTP connections to WebSocket connections.
var upgrader = websocket.Upgrader{
	// Allow all origins for simplicity. You may want to restrict this in production.
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	router := chi.NewRouter()

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Goodbye, World!"))
	})

	router.Post("/answer", func(w http.ResponseWriter, r *http.Request) {
		stream := `
<Response>

    <Connect>
        <Stream name="Example Audio Stream" url="wss://12eb-98-176-235-185.ngrok-free.app/stream" />
    </Connect>

    <Say>The stream has started.</Say>

</Response>`

		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(stream))
	})

	router.Get("/stream", func(w http.ResponseWriter, r *http.Request) {
		log.Println("/stream was called")
		// Upgrade the HTTP request to a WebSocket connection.
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}
		defer ws.Close() // Ensure the connection is closed when the function exits

		log.Println("Client connected")

		// Read messages in a loop and log them
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				log.Println("Read error:", err)
				break
			}
			log.Printf("Received a message of length", len(msg))
		}

		log.Println("Client disconnected")
	})

	http.ListenAndServe(":8080", router)
}
