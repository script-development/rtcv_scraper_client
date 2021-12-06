package main

import (
	"encoding/json"
	"fmt"
)

// MessageType represends a response message type
type MessageType uint8

const (
	// MessageTypeReady tells the client is ready to receive messages
	MessageTypeReady MessageType = iota
	// MessageTypePong is the response of a ping input message
	MessageTypePong
	// MessageTypeOk is the response of a successful operation
	MessageTypeOk
	// MessageTypeError is the response of an error
	MessageTypeError
)

func (mt MessageType) String() string {
	switch mt {
	case MessageTypeReady:
		return "ready"
	case MessageTypePong:
		return "pong"
	case MessageTypeOk:
		return "ok"
	case MessageTypeError:
		return "error"
	default:
		return "unknown"
	}
}

// Print prints a json message to screen with the MessageType and messageContent
func (mt MessageType) Print(messageContent interface{}) {
	var content []byte

	if messageContent != nil {
		var err error
		content, err = json.Marshal(messageContent)
		if err != nil {
			content = nil
		}
	}

	jsonData, err := json.Marshal(struct {
		Type    string          `json:"type"`
		Content json.RawMessage `json:"content,omitempty"`
	}{
		Type:    mt.String(),
		Content: content,
	})
	if err != nil {
		errJSON, _ := json.Marshal(err.Error())
		fmt.Println(`{"type":"error","content":` + string(errJSON) + `}`)
	} else {
		fmt.Println(string(jsonData))
	}
}

// InMessage represents the json contents of a stdin line
type InMessage struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
}
