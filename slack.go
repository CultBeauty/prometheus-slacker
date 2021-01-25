package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

const ColorInfo = "#00BFFF"    // Deep Sky Blue
const ColorSuccess = "#00FF00" // Lime
const ColorWarn = "#FFD700"    // Gold
const ColorError = "#DC143C"   // Crimson

type SlackMessage struct {
	Attachments []SlackAttachment `json:"attachments"`
}

type SlackAttachment struct {
	Fallback string       `json:"fallback"`
	Color    string       `json:"color"`
	Fields   []SlackField `json:"fields"`
}

type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

type SlackWebhook struct {
	Url         string          `json:"url"`
	ShowDetails map[string]bool `json:"show_details"`
}

func (n *SlackWebhook) SendMessage(msg SlackMessage) error {
	log.Print("Sending Slack message...")

	payload, err := json.Marshal(msg)
	if err != nil {
		return errors.New("Failed to marshal Slack message: " + err.Error())
	}

	res, err := http.Post(n.Url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return errors.New("Failed to send Slack message - got error: " + err.Error())
	}
	res.Body.Close()

	log.Print("Slack message sent")

	return nil
}
