package main

import (
	"encoding/json"
	"fmt"
)

type MessageType uint8

const (
	MessageTypeReady MessageType = iota
	MessageTypePong
	MessageTypeOk
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

func PrintMessage(messageType MessageType, messageContent interface{}) {
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
		Type:    messageType.String(),
		Content: content,
	})
	if err != nil {
		errJSON, _ := json.Marshal(err.Error())
		fmt.Println(`{"type":"error","content":` + string(errJSON) + `}`)
	} else {
		fmt.Println(string(jsonData))
	}
}

type InMessage struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
}
