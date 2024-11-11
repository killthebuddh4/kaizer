package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/websocket"
	"github.com/twilio/twilio-go/client"
)

// Define an upgrader to upgrade HTTP connections to WebSocket connections.
var upgrader = websocket.Upgrader{
	// Allow all origins for simplicity. You may want to restrict this in production.
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	kaizerEnv, err := getKaizerEnv()

	if err != nil {
		slog.Debug("Error from getKaizerEnv:" + err.Error())
		return
	}

	if kaizerEnv.Env != "production" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	sess := session.Must(session.NewSession(&aws.Config{
		Region: &kaizerEnv.BucketRegion,
		Credentials: credentials.NewStaticCredentials(
			kaizerEnv.BucketAccessKeyId,
			kaizerEnv.BucketAccessKey,
			"",
		),
	}))

	router := chi.NewRouter()

	router.Use(middleware.Logger)

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/index.html")
	})

	router.Get("/version", func(w http.ResponseWriter, r *http.Request) {
		message := "Thump. Thump thump. Thump. Version " + kaizerEnv.Version
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(message))
	})

	router.Post("/answer", func(w http.ResponseWriter, r *http.Request) {
		if !isRequestFromAdmin(r) {
			slog.Debug("isRequestFromAdmin returned false")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}

		if !isRequestFromTwilio("https://", r) {
			slog.Debug("isRequestFromTwilio returned false")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}

		command := getStartStreamCommand(kaizerEnv.WebsocketUrl)

		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(command))
	})

	router.Get("/stream", func(w http.ResponseWriter, r *http.Request) {
		if !isRequestFromTwilio("wss://", r) {
			slog.Debug("isRequestFromTwilio returned false")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}

		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Debug("Error from upgrader.Upgrade:" + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal server error"))
			return
		}
		defer ws.Close()

		reader, writer := io.Pipe()

		defer writer.Close()

		go func() {
			defer reader.Close()

			// TODO WE HAVE TO CHANGE THIS FOR MULTI-USER
			key := fmt.Sprintf("%s/%s", kaizerEnv.AdminPhoneNumber, getTimestamp())

			uploader := s3manager.NewUploader(sess)

			_, err = uploader.Upload(&s3manager.UploadInput{
				Body:   reader,
				Bucket: &kaizerEnv.BucketName,
				Key:    &key,
			})

			if err != nil {
				slog.Error("Error from s3 uploader:" + err.Error())
				return
			}
		}()

		for {
			var raw map[string]interface{}

			_, bytes, err := ws.ReadMessage()
			if err != nil {
				slog.Debug("Error from ws.ReadMessage:" + err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal server error"))
				return
			}

			err = json.Unmarshal(bytes, &raw)

			if err != nil {
				slog.Debug("Error unmarshalling message json:" + err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal server error"))
				return
			}

			if raw["event"] == "start" {
				slog.Debug("Start message received.")
			} else if raw["event"] == "stop" {
				slog.Debug("Stop message received.")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Processing complete"))
				return
			} else if raw["event"] == "media" {
				slog.Debug("Media message received.")

				var msg MediaMessage
				err := json.Unmarshal(bytes, &msg)

				if err != nil {
					slog.Debug("Error unmarshalling media message json:" + err.Error())
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Internal server error"))
					return
				}

				data, err := base64.StdEncoding.DecodeString(msg.Media.Payload)

				if err != nil {
					slog.Debug("Error decoding base64:" + err.Error())
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Internal server error"))
					return
				}

				_, err = writer.Write(data)

				if err != nil {
					slog.Debug("Error writing to pipe:" + err.Error())
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Internal server error"))
					return
				}
			} else {
				slog.Debug("Unknown message type.")
			}
		}
	})

	http.ListenAndServe(":8080", router)
}

func isRequestFromTwilio(scheme string, r *http.Request) bool {
	if scheme != "https://" && scheme != "wss://" {
		return false
	}

	kaizerEnv, err := getKaizerEnv()

	if err != nil {
		return false
	}

	validator := client.NewRequestValidator(kaizerEnv.TwilioAuthToken)

	err = r.ParseForm()

	if err != nil {
		return false
	}

	params := make(map[string]string)

	for key, values := range r.Form {
		if len(values) != 1 {
			return false
		}

		params[key] = values[0]
	}

	url := fmt.Sprintf("%s%s%s", scheme, r.Host, r.URL.Path)
	signature := r.Header.Get("X-Twilio-Signature")

	if signature == "" {
		return false
	}

	return validator.Validate(url, params, signature)
}

func isRequestFromAdmin(r *http.Request) bool {
	kaizerEnv, err := getKaizerEnv()

	if err != nil {
		return false
	}

	err = r.ParseForm()

	if err != nil {
		return false
	}

	from := r.Form.Get("From")

	slog.Debug("Request From:" + from)

	return from == kaizerEnv.AdminPhoneNumber
}

func getTimestamp() string {
	now := time.Now()
	return now.Format("2006-01-02-15-04-05-000")
}

func getStartStreamCommand(websocketUrl string) string {
	return fmt.Sprintf(`
<Response>
    <Say>Thanks for calling, we will now start recording your audio.</Say>
    <Connect>
        <Stream name="kaizer-stream" url="%s" />
    </Connect>
</Response>`, websocketUrl)
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

type KaizerEnv struct {
	Env               string
	AdminPhoneNumber  string
	TwilioAccountSid  string
	TwilioAuthToken   string
	Version           string
	BucketAccessKey   string
	BucketAccessKeyId string
	BucketName        string
	BucketRegion      string
	WebsocketUrl      string
}

func getKaizerEnv() (KaizerEnv, error) {
	adminPhoneNumber := os.Getenv("KAIZER_ADMIN_PHONE_NUMBER")

	if adminPhoneNumber == "" {
		log.Fatal("KAIZER_ADMIN_PHONE_NUMBER environment variable is required")
		return KaizerEnv{}, fmt.Errorf("KAIZER_ADMIN_PHONE_NUMBER environment variable is required")
	}

	twilioAccountSid := os.Getenv("KAIZER_TWILIO_ACCOUNT_SID")

	if twilioAccountSid == "" {
		log.Fatal("KAIZER_TWILIO_ACCOUNT_SID environment variable is required")
		return KaizerEnv{}, fmt.Errorf("KAIZER_TWILIO_ACCOUNT_SID environment variable is required")
	}

	twilioAuthToken := os.Getenv("KAIZER_TWILIO_AUTH_TOKEN")

	if twilioAuthToken == "" {
		log.Fatal("KAIZER_TWILIO_AUTH_TOKEN environment variable is required")
		return KaizerEnv{}, fmt.Errorf("KAIZER_TWILIO_AUTH_TOKEN environment variable is required")
	}

	version := os.Getenv("KAIZER_GIT_COMMIT_SHA")

	if version == "" {
		log.Fatal("KAIZER_GIT_COMMIT_SHA environment variable is required")
		return KaizerEnv{}, fmt.Errorf("KAIZER_GIT_COMMIT_SHA environment variable is required")
	}

	bucketAccessKeyId := os.Getenv("KAIZER_BUCKET_ACCESS_KEY_ID")

	if bucketAccessKeyId == "" {
		log.Fatal("KAIZER_BUCKET_ACCESS_KEY_ID environment variable is required")
		return KaizerEnv{}, fmt.Errorf("KAIZER_BUCKET_ACCESS_KEY_ID environment variable is required")
	}

	kaizerBucketAccessKey := os.Getenv("KAIZER_BUCKET_ACCESS_KEY")

	if kaizerBucketAccessKey == "" {
		log.Fatal("KAIZER_BUCKET_ACCESS_KEY environment variable is required")
		return KaizerEnv{}, fmt.Errorf("KAIZER_BUCKET_ACCESS_KEY environment variable is required")
	}

	bucketName := os.Getenv("KAIZER_BUCKET_NAME")

	if bucketName == "" {
		log.Fatal("KAIZER_BUCKET_NAME environment variable is required")
		return KaizerEnv{}, fmt.Errorf("KAIZER_BUCKET_NAME environment variable is required")
	}

	bucketRegion := os.Getenv("KAIZER_BUCKET_REGION")

	if bucketRegion == "" {
		log.Fatal("KAIZER_BUCKET_REGION environment variable is required")
		return KaizerEnv{}, fmt.Errorf("KAIZER_BUCKET_REGION environment variable is required")
	}

	websocketUrl := os.Getenv("KAIZER_WEBSOCKET_URL")

	if websocketUrl == "" {
		log.Fatal("KAIZER_WEBSOCKET_URL environment variable is required")
		return KaizerEnv{}, fmt.Errorf("KAIZER_WEBSOCKET_URL environment variable is required")
	}

	env := os.Getenv("KAIZER_GO_ENV")

	if env == "" {
		log.Fatal("KAIZER_GO_ENV environment variable is required")
		return KaizerEnv{}, fmt.Errorf("KAIZER_GO_ENV environment variable is required")
	}

	return KaizerEnv{
		Env:               env,
		AdminPhoneNumber:  adminPhoneNumber,
		TwilioAccountSid:  twilioAccountSid,
		TwilioAuthToken:   twilioAuthToken,
		Version:           version,
		BucketAccessKeyId: bucketAccessKeyId,
		BucketAccessKey:   kaizerBucketAccessKey,
		BucketName:        bucketName,
		BucketRegion:      bucketRegion,
		WebsocketUrl:      websocketUrl,
	}, nil
}
