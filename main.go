package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

const (
	defaultDogStatsDHost = "127.0.0.1"
	defaultDogStatsDPort = "8125"
	defaultMetricPrefix  = "sendgrid.event."
	defaultServerPort    = "8080"
)

var (
	basicAuthPassword string
	basicAuthUsername string
	metricPrefix      string
	statsdClient      *statsd.Client
)

// SendGridEvents represents the scheme of Event Webhook body
// https://sendgrid.com/docs/for-developers/tracking-events/event/
type SendGridEvents []struct {
	Email       string   `json:"email"`
	Timestamp   int      `json:"timestamp"`
	SMTPID      string   `json:"smtp-id,omitempty"`
	Event       string   `json:"event"`
	Category    []string `json:"category,omitempty"`
	SGEventID   string   `json:"sg_event_id"`
	SGMessageID string   `json:"sg_message_id"`
	Useragent   string   `json:"useragent,omitempty"`
	URL         string   `json:"url,omitempty"`
	AsmGroupID  int      `json:"asm_group_id,omitempty"`
	Reason      string   `json:"reason,omitempty"`
	Status      string   `json:"status,omitempty"`
	Response    string   `json:"response,omitempty"`
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong")
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "https://github.com/dtan4/sendgrid2datadog")
}

func eventUnmarshal(b []byte, e *SendGridEvents) error {
	if err := json.Unmarshal(b, &e); err != nil {
		return err
	}
	return nil
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	if basicAuthUsername != "" && basicAuthPassword != "" {
		if !checkAuth(r) {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "Unauthorized")
			return
		}
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		fmt.Fprintf(w, "%s", err)
		return
	}

	var events SendGridEvents

	if err = eventUnmarshal(body, &events); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		fmt.Fprintf(w, "%s", err)
		return
	}

	if err := submitMetrics(events); err != nil {
		w.WriteHeader(http.StatusBadGateway)
		log.Println(err)
		fmt.Fprintf(w, "%s", err)
		return
	}
}

func submitMetrics(events SendGridEvents) error {
	for _, event := range events {
		var tags []string
		for _, c := range event.Category {
			tags = append(tags, fmt.Sprintf("category:%s", c))
		}

		if event.Reason != "" {
			tags = append(tags, fmt.Sprintf("reason:%s", event.Reason))
		}

		if event.Response != "" {
			tags = append(tags, fmt.Sprintf("reason:%s", event.Response))
		}

		if err := statsdClient.Incr(metricPrefix+event.Event, tags, 1); err != nil {
			return err
		}
	}
	return nil
}

func checkAuth(r *http.Request) bool {
	username, password, ok := r.BasicAuth()
	if !ok {
		return false
	}

	return username == basicAuthUsername && password == basicAuthPassword
}

func main() {
	var (
		dogStatsDHost, dogStatsDPort string
		serverPort                   string
	)

	basicAuthUsername = os.Getenv("BASIC_AUTH_USERNAME")
	basicAuthPassword = os.Getenv("BASIC_AUTH_PASSWORD")

	dogStatsDHost = os.Getenv("DOGSTATSD_HOST")
	if dogStatsDHost == "" {
		dogStatsDHost = defaultDogStatsDHost
	}

	dogStatsDPort = os.Getenv("DOGSTATSD_PORT")
	if dogStatsDPort == "" {
		dogStatsDPort = defaultDogStatsDPort
	}

	dogStatsDAddr := fmt.Sprintf("%s:%s", dogStatsDHost, dogStatsDPort)

	metricPrefix = os.Getenv("METRIC_PREFIX")
	if metricPrefix == "" {
		metricPrefix = defaultMetricPrefix
	}

	serverPort = os.Getenv("PORT")
	if serverPort == "" {
		serverPort = defaultServerPort
	}

	var err error

	statsdClient, err = statsd.New(dogStatsDAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	r := mux.NewRouter()

	r.Handle("/", handlers.LoggingHandler(os.Stdout, http.HandlerFunc(rootHandler))).Methods("GET")
	r.Handle("/ping", handlers.LoggingHandler(os.Stdout, http.HandlerFunc(pingHandler))).Methods("GET")
	r.Handle("/webhook", handlers.LoggingHandler(os.Stdout, http.HandlerFunc(webhookHandler))).Methods("POST")

	fmt.Println("Server started.")

	if err := http.ListenAndServe(fmt.Sprintf(":%s", serverPort), r); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
