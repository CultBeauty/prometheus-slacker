package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
	"net/url"
)

type PrometheusSlacker struct {
	config Config
}

func (ps *PrometheusSlacker) GetConfig() *Config {
	return &(ps.config)
}

func (ps *PrometheusSlacker) Init(p string) {
	c, err := ioutil.ReadFile(p)
	if err != nil {
		log.Fatal("Error reading config file")
	}

	var cfg Config
	cfg.SetFromJSON(c)
	ps.config = cfg
}

func (ps *PrometheusSlacker) Run() int {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Syntax: prometheusslacker <config.json path>\n")
		os.Exit(1)
	}
	ps.Init(os.Args[1])
	log.Print(ps.config)
	done := make(chan bool)
	go ps.startHttpd()
	<-done
	return 0
}

func (ps *PrometheusSlacker) startHttpd() {
	done := make(chan bool)
	go ps.startScrapper()
	go ps.startApi()
	<-done
}

func (ps *PrometheusSlacker) startScrapper() {
	delay := ps.config.ScrapperMinutes
	log.Print(fmt.Sprintf("Starting scrapper to check every %d minutes...", delay))
	if delay < 1 {
		delay = 1
	}
	for {
		currentLevel := ""
		for i, notificationLevel := range ps.config.NotificationLevels {
			color := notificationLevel.Color

			if len(notificationLevel.Metrics) == 0 && i == 0 {
				currentLevel = color
			}

			for _, metric := range notificationLevel.Metrics {
				c, err := ps.GetMetricValue(metric.Query)
				if err != nil {
					log.Print(fmt.Sprintf("Error getting metric value for %s", metric.Query))
					break
				}

				leverage, err := ps.IsValueBiggerThanThreshold(c, metric.Threshold)
				if err != nil {
					log.Print(err.Error())
					break
				}
				if leverage {
					log.Print(fmt.Sprintf("%f <= %f so changing color to %s", metric.Threshold, c, color))
					currentLevel = color
					break
				}
			}

			log.Print(fmt.Sprintf("Current color is: %s\n", currentLevel))
		}
		if currentLevel != "" {
			for _, notificationLevel := range ps.config.NotificationLevels {
				if notificationLevel.Color == currentLevel {
					log.Print(notificationLevel.SlackWebhooks)
					for _, w := range notificationLevel.SlackWebhooks {
						webhook := ps.config.SlackWebhooks[w]
						msg := SlackMessage{
							Attachments: []SlackAttachment{
								SlackAttachment{
									Color: currentLevel,
									Fields: []SlackField{
										SlackField{
											Title: fmt.Sprintf("Color: %s", currentLevel),
											Value: "",
											Short: false,
										},
									},
								},
							},
						}
						log.Print(msg)
						err := webhook.SendMessage(msg)
						if err != nil {
							log.Print("Error sending slack msg")
						}
					}
				}
			}
		}

		log.Print(fmt.Sprintf("Sleeping %d minutes...", delay))
		time.Sleep(time.Minute * time.Duration(delay))
	}
}

func (ps *PrometheusSlacker) startApi() {
	router := mux.NewRouter()
	router.HandleFunc("/", ps.getHandler()).Methods("POST")
	log.Print("Starting daemon listening on " + ps.config.Port + "...")
	log.Fatal(http.ListenAndServe(":"+ps.config.Port, router))
}

func (ps PrometheusSlacker) GetMetricValue(metric string) (string, error) {
	res, err := http.Get(ps.config.PrometheusURL + "/api/v1/query?query=" + url.QueryEscape(metric))
	if err != nil {
		return "", err
	}
	c, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	var j map[string]interface{}
	err = json.Unmarshal(c, &j)
	if err != nil {
		log.Print("Error unmarshalling metric response")
		return "", nil
	}
	// data.result.0.value.1
	data := j["data"].(map[string]interface{})
	result := data["result"].([]interface{})
	value := result[0].(map[string]interface{})["value"].([]interface{})
	currentValStr := value[1].(string)
	log.Print("Metric " + metric + " value " + currentValStr)

	return currentValStr, nil
}

func (ps *PrometheusSlacker) getHandler() func(http.ResponseWriter, *http.Request) {
	fn := func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		msg := SlackMessage{}
		err = json.Unmarshal(b, &msg)
		if err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		for _, s := range ps.config.SlackWebhooks {
			_ = s.SendMessage(msg)
		}

		w.WriteHeader(http.StatusOK)
		return
	}
	return http.HandlerFunc(fn)
}

func (ps PrometheusSlacker) IsValueBiggerThanThreshold(val string, threshold string) (bool, error) {
	currentVal, err := strconv.ParseFloat(val, 64)
	if err != nil {
		log.Print("Error getting float from string for current val")
	}

	compareVal, err := strconv.ParseFloat(threshold, 64)
	if err != nil {
		log.Print("Error getting float from string for threshold val")
	}

	log.Print(fmt.Sprintf("Comparing %f and %f ...", compareVal, currentVal))
	if compareVal <= currentVal {
		return true, nil
	}

	return false, nil
}

func NewPrometheusSlacker() *PrometheusSlacker {
	ps := &PrometheusSlacker{}
	return ps
}
