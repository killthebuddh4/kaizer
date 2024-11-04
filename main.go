package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
)

// Define an upgrader to upgrade HTTP connections to WebSocket connections.
var upgrader = websocket.Upgrader{
	// Allow all origins for simplicity. You may want to restrict this in production.
	CheckOrigin: func(r *http.Request) bool { return true },
}

type StartMessage struct {
	Event string `json:"event"`
	Start struct {
		AccountSid  string `json:"accountSid"`
		StreamSid   string `json:"streamSid"`
		CallSid     string `json:"callSid"`
		MediaFormat struct {
			Encoding   string `json:"encoding"`
			SampleRate int    `json:"sampleRate"`
			Channels   int    `json:"channels"`
		}
	}
}

type MediaMessage struct {
	Event string `json:"event"`
	Media struct {
		Track     string `json:"track"`
		Chunk     string `json:"chunk"`
		Timestamp string `json:"timestamp"`
		// This will be base64 encoded audio data
		Payload string `json:"payload"`
	}
}

type StopMessage struct {
	Event string `json:"event"`
	Stop  struct {
		AccountSid string `json:"accountSid"`
		CallSid    string `json:"callSid"`
	}
	StreamSid string `json:"streamSid"`
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
        <Stream name="Example Audio Stream" url="wss://7cfb-98-176-235-185.ngrok-free.app/stream" />
    </Connect>

    <Say>The stream has started.</Say>

</Response>`

		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(stream))
	})

	router.Get("/stream", func(w http.ResponseWriter, r *http.Request) {
		ofile, err := os.Create("output.wav")

		if err != nil {
			log.Println("Error creating output file:", err)
			return
		}

		enc := wav.NewEncoder(ofile, 8000, 16, 1, 7)
		defer enc.Close()

		// Upgrade the HTTP request to a WebSocket connection.
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}
		defer ws.Close() // Ensure the connection is closed when the function exits

		log.Println("Client connected")

		audioBuf := &audio.IntBuffer{
			Format: &audio.Format{
				SampleRate:  8000,
				NumChannels: 1,
			},
			Data: make([]int, 8000),
		}

		// Read messages in a loop and log them
		for {
			var raw map[string]interface{}

			_, bytes, err := ws.ReadMessage()
			if err != nil {
				log.Println("Read error:", err)
				break
			}

			err = json.Unmarshal(bytes, &raw)

			if err != nil {
				log.Println("Unmarshal error:", err)
				break
			}

			if raw["event"] == "start" {
				log.Println("Start messaging received, processing...")

				// var msg StartMessage
				// err := json.Unmarshal(bytes, &StartMessage{})

				// if err != nil {
				// 	log.Println("Unmarshal error:", err)
				// 	break
				// }

				log.Println("Done processing start message")
			} else if raw["event"] == "media" {
				log.Println("Media message received, processing...")

				var msg MediaMessage
				err := json.Unmarshal(bytes, &MediaMessage{})

				if err != nil {
					log.Println("Unmarshal error:", err)
					break
				}

				bs, err := base64.StdEncoding.DecodeString(msg.Media.Payload)

				if err != nil {
					log.Println("Base64 decode error:", err)
					break
				}

				ints := make([]int, len(bs))

				for i, b := range bs {
					ints[i] = int(b)
				}

				audioBuf.Data = append(audioBuf.Data, ints...)

				log.Println("Done processing media message")
			} else if raw["event"] == "stop" {
				log.Println("Stop message received")

				log.Println("Writing audio data to file")

				enc.Write(audioBuf)

				log.Println("Done processing stop message")
			}
		}

		log.Println("Client disconnected")
	})

	http.ListenAndServe(":8080", router)
}
