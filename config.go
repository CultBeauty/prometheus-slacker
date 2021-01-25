package main

import (
	"encoding/json"
	"log"
)

type Config struct {
	Version            string                  `json:"version"`
	Port               string                  `json:"port"`
	SlackWebhooks      map[string]SlackWebhook `json:"slack_webhooks"`
	PrometheusURL      string                  `json:"prometheus_url"`
	ScrapperMinutes    int                     `json:"scrapper_minutes"`
	NotificationLevels []NotificationLevel     `json:"notification_levels"`
}

func (c *Config) SetFromJSON(b []byte) {
	err := json.Unmarshal(b, c)
	if err != nil {
		log.Fatal("Error setting config from JSON:", err.Error())
	}
}

type NotificationLevel struct {
	Color         string   `json:"color"`
	Emoji         string   `json:"emoji"`
	SlackWebhooks []string `json:"slack_webhooks"`
	Metrics       []Metric `json:"metrics"`
}

type Metric struct {
	DisplayName string `json:"display_name"`
	Query       string `json:"query"`
	Threshold   string `json:"threshold"`
}
