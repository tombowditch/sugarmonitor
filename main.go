package main

import (
	"bytes"
	"embed"
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/tombowditch/sugarmonitor/nightscout"
)

func init() {
	godotenv.Load()
}

var (
	ackTime   *time.Time
	sleepTime = 1 * time.Minute
)

//go:embed sound.mp3
var f embed.FS

func main() {
	ns, err := nightscout.NewNightscout()
	if err != nil {
		logrus.WithError(err).Fatal("creating Nightscout instance")
	}

	go startWeb()

	for {
		logrus.Info("getting current blood sugar")
		glucoseReading, err := ns.GetCurrentBloodSugar()
		if err != nil {
			logrus.WithError(err).Error("getting current blood sugar")
			time.Sleep(15 * time.Second)
			continue
		}

		logrus.WithField("mmol", glucoseReading).Info("got blood sugar")
		go sendToTinybird(*glucoseReading)

		if glucoseReading.GlucoseMmol < 5.0 || glucoseReading.GlucoseMmol > 12.0 {
			logrus.WithField("mmol", glucoseReading).Info("alerting")

			if ackTime != nil {
				if time.Since(*ackTime) < 30*time.Minute {
					logrus.WithField("ack_time", *ackTime).Info("already acknowledged, not alerting")
					time.Sleep(sleepTime)
					continue
				}
			}

			// alert
			data, err := f.Open("sound.mp3")
			if err != nil {
				logrus.WithError(err).Error("opening sound.mp3")
				time.Sleep(sleepTime)
				continue
			}

			streamer, format, err := mp3.Decode(data)
			if err != nil {
				logrus.WithError(err).Error("decoding mp3")
				time.Sleep(sleepTime)
				continue
			}

			err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
			if err != nil {
				logrus.WithError(err).Error("initializing speaker")
				time.Sleep(sleepTime)
				continue
			}

			speaker.Play(streamer)
		}

		time.Sleep(sleepTime)
	}
}

func startWeb() {
	r := mux.NewRouter()
	r.HandleFunc("/ack", func(w http.ResponseWriter, r *http.Request) {
		logrus.Info("received ack request")
		now := time.Now()
		ackTime = &now
		w.Write([]byte("ok"))
	})

	s := &http.Server{
		Handler: r,
		Addr:    "0.0.0.0:4444",
	}

	logrus.Fatal(s.ListenAndServe())
}

func sendToTinybird(gr nightscout.GlucoseReading) {
	tinybirdToken := os.Getenv("TINYBIRD_TOKEN")
	if tinybirdToken == "" {
		return
	}

	jsonStr, err := json.Marshal(gr)
	if err != nil {
		logrus.WithError(err).Error("marshalling glucose reading")
		return
	}

	req, err := http.NewRequest("POST", "https://api.tinybird.co/v0/events?name=nightscout", bytes.NewBuffer(jsonStr))
	if err != nil {
		logrus.WithError(err).Error("creating request")
		return
	}

	req.Header.Set("Authorization", "Bearer "+os.Getenv("TINYBIRD_TOKEN"))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("sending to tinybird")
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.WithError(err).Error("reading response body")
		return
	}

	logrus.WithField("response", string(body)).Info("sent to tinybird")
}
