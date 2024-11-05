package main

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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
	sess := session.Must(session.NewSession())

	router := chi.NewRouter()

	router.Get("/version", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Handling /version")
		message := "Thump. Thump thump. Thump. Version " + os.Getenv("KAIZER_VERSION")
		log.Println(message)
		w.Write([]byte(message))
	})

	router.Post("/answer", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Handling /answer")
		stream := `
<Response>
    <Connect>
        <Stream name="Example Audio Stream" url="wss://talktome.cloud/stream" />
    </Connect>
    <Say>Thanks for calling, your log has been saved.</Say>
</Response>`

		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(stream))
	})

	router.Get("/stream", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Handling /stream")
		// Upgrade the HTTP request to a WebSocket connection.
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}
		defer ws.Close() // Ensure the connection is closed when the function exits

		log.Println("Connection upgraded to WebSocket")

		reader, writer := io.Pipe()
		// Do we need to close the reader and the writer?

		defer writer.Close()

		log.Println("Creating S3 uploader")

		uploader := s3manager.NewUploader(sess)

		bucket := "kaizer"
		key := "test"

		_, err = uploader.Upload(&s3manager.UploadInput{
			Body:   reader,
			Bucket: &bucket,
			Key:    &key,
		})

		if err != nil {
			log.Println("Error uploading to S3:", err)
			return
		}

		log.Println("Writing to S3")

		_, err = writer.Write([]byte("Hello, World!"))

		log.Println("Done writing to S3")

		if err != nil {
			log.Println("Error uploading to S3:", err)
			return
		}

		// log.Println("Client connected")

		// // Read messages in a loop and log them
		// for {
		// 	var raw map[string]interface{}

		// 	_, bytes, err := ws.ReadMessage()
		// 	if err != nil {
		// 		log.Println("Read error:", err)
		// 		break
		// 	}

		// 	err = json.Unmarshal(bytes, &raw)

		// 	if err != nil {
		// 		log.Println("Unmarshal error:", err)
		// 		break
		// 	}

		// 	if raw["event"] == "start" {
		// 		log.Println("Start messaging received, processing...")

		// 		log.Println("Done processing start message")
		// 	} else if raw["event"] == "media" {
		// 		log.Println("Media message received, processing...")

		// 		var msg MediaMessage
		// 		err := json.Unmarshal(bytes, &msg)

		// 		if err != nil {
		// 			log.Println("Unmarshal error:", err)
		// 			break
		// 		}

		// 		data, err := base64.StdEncoding.DecodeString(msg.Media.Payload)

		// 		if err != nil {
		// 			log.Println("Base64 decode error:", err)
		// 			break
		// 		}

		// 		_, err = writer.Write(data)

		// 		if err != nil {
		// 			log.Println("Write error:", err)
		// 			break
		// 		}

		// 		log.Println("Done processing media message")
		// 	} else if raw["event"] == "stop" {
		// 		log.Println("Stop message received")
		// 		log.Println("Done processing stop message")

		// 		w.WriteHeader(http.StatusOK)
		// 		w.Write([]byte("Goodbye, World!"))
		// 	}
		// }

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Goodbye, World!"))

		log.Println("Client disconnected")
	})

	http.ListenAndServe(":8080", router)
}
