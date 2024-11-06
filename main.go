package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
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
	kaizerAdminPhoneNmber := os.Getenv("KAIZER_ADMIN_PHONE_NUMBER")

	if kaizerAdminPhoneNmber == "" {
		log.Fatal("KAIZER_ADMIN_PHONE_NUMBER environment variable is required")
		return
	}

	twilioAccountSid := os.Getenv("TWILIO_ACCOUNT_SID")

	if twilioAccountSid == "" {
		log.Fatal("TWILIO_ACCOUNT_SID environment variable is required")
		return
	}

	twilioAuthToken := os.Getenv("TWILIO_AUTH_TOKEN")

	if twilioAuthToken == "" {
		log.Fatal("TWILIO_AUTH_TOKEN environment variable is required")
		return
	}

	kaizerVersion := os.Getenv("KAIZER_VERSION")

	if kaizerVersion == "" {
		log.Fatal("KAIZER_VERSION environment variable is required")
		return
	}

	kaizerBucketAccessKeyId := os.Getenv("KAIZER_BUCKET_ACCESS_KEY_ID")

	if kaizerBucketAccessKeyId == "" {
		log.Fatal("KAIZER_BUCKET_ACCESS_KEY_ID environment variable is required")
		return
	}

	kaizerBucketAccessKey := os.Getenv("KAIZER_BUCKET_ACCESS_KEY")

	if kaizerBucketAccessKey == "" {
		log.Fatal("KAIZER_BUCKET_ACCESS_KEY environment variable is required")
		return
	}

	kaizerBucket := os.Getenv("KAIZER_BUCKET_NAME")

	if kaizerBucket == "" {
		log.Fatal("KAIZER_BUCKET_NAME environment variable is required")
		return
	}

	kaizerBucketRegion := os.Getenv("KAIZER_BUCKET_REGION")

	if kaizerBucketRegion == "" {
		log.Fatal("KAIZER_BUCKET_REGION environment variable is required")
		return
	}

	websocketUrl := os.Getenv("KAIZER_WEBSOCKET_URL")

	if websocketUrl == "" {
		log.Fatal("KAIZER_WEBSOCKET_URL environment variable is required")
		return
	}

	sess := session.Must(session.NewSession(&aws.Config{
		Region: &kaizerBucketRegion,
		Credentials: credentials.NewStaticCredentials(
			kaizerBucketAccessKeyId,
			kaizerBucketAccessKey,
			"",
		),
	}))

	router := chi.NewRouter()

	router.Get("/version", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Handling /version")
		message := "Thump. Thump thump. Thump. Version " + os.Getenv("KAIZER_VERSION")
		log.Println(message)
		w.Write([]byte(message))
	})

	router.Post("/answer", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Handling /answer, redirecting to %s", websocketUrl)

		err := r.ParseForm()

		if err != nil {
			log.Fatal("Error parsing form data:", err)
		}

		from := r.Form.Get("From")

		if from != kaizerAdminPhoneNmber {
			log.Printf("Unauthorized call from %s", from)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}

		stream := fmt.Sprintf(`
<Response>
    <Say>Thanks for calling, everything you say here will be recorded.</Say>
    <Connect>
        <Stream name="Example Audio Stream" url="%s" />
    </Connect>
    <Say>Thanks for calling, your log has been saved.</Say>
</Response>`, websocketUrl)

		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(stream))
	})

	router.Get("/stream", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Handling /stream")

		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}
		defer ws.Close()

		log.Println("Connection upgraded to WebSocket")

		reader, writer := io.Pipe()

		go func() {
			defer reader.Close()

			key := "test"

			log.Println("Creating S3 uploader")

			uploader := s3manager.NewUploader(sess)

			_, err = uploader.Upload(&s3manager.UploadInput{
				Body:   reader,
				Bucket: &kaizerBucket,
				Key:    &key,
			})

			if err != nil {
				log.Println("Error uploading to S3:", err)
				return
			}
		}()

		for {
			var raw map[string]interface{}

			_, bytes, err := ws.ReadMessage()
			if err != nil {
				log.Println("Read message error:", err)
				break
			}

			err = json.Unmarshal(bytes, &raw)

			if err != nil {
				log.Println("Unmarshal error:", err)
				break
			}

			if raw["event"] == "start" {
				log.Println("Start message received.")
			} else if raw["event"] == "stop" {
				log.Println("Stop message received, closing writer...")

				writer.Close()

				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Processing complete"))
			} else if raw["event"] == "media" {
				log.Println("Media message received, processing...")

				var msg MediaMessage
				err := json.Unmarshal(bytes, &msg)

				if err != nil {
					log.Println("Unmarshal error:", err)
					break
				}

				data, err := base64.StdEncoding.DecodeString(msg.Media.Payload)

				if err != nil {
					log.Println("Base64 decode error:", err)
					break
				}

				_, err = writer.Write(data)

				if err != nil {
					log.Println("Write error:", err)
					break
				}

				log.Println("Done processing media message")
			} else {
				log.Println("Unknown message type, received:", raw["event"])
			}
		}

		log.Println("Client disconnected")
	})

	http.ListenAndServe(":8080", router)
}
