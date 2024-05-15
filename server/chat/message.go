package chat

import (
	"encoding/json"

	"github.com/google/uuid"
)

const (
	ACTION_SEND_MESSAGE          = "send-msg"
	ACTION_JOIN_CHANNEL          = "join-channel"
	ACTION_LEAVE_CHANNEL         = "leave-channel"
	ACTION_LEFT_CHANNEL          = "left-channel"
	ACTION_JOINED_CHANNEL        = "joined-channel"
	ACTION_NOTSUBSCRIBED_CHANNEL = "not-joined-channel"
	ACTION_PRIVATE_CHANNEL       = "join-private-channel"

	ACTION_SUBSCRIBER_JOINED = "subscriber-joined"
	ACTION_SUBSCRIBER_LEFT   = "subscriber-left"
)

type MessageType int

const (
	MSGTYPE_REQ   MessageType = iota // Request message - requires an ack pair from other side
	MSGTYPE_ACK                      // Acknowledge message for every request message
	MSGTYPE_BCAST                    // Broadcast message - no acknowledge response needed
)

// Use in Req./Ack. messaging.
//
// Id will determine which ack. for each req. message
// MessageType will detect the source and direction of the message (client <-> server).
type Message struct {
	Id          uuid.UUID   `json:"id"`
	MessageType MessageType `json:"messagetype"`
	RequestType string      `json:"requesttype"`
	Message     string      `json:"message"`
	ChannelName string      `json:"channel"`
	Session     *Session    `json:"session"`
}

func NewMessage(messageType MessageType) *Message {
	return &Message{
		Id:          uuid.New(),
		MessageType: messageType,
	}
}

func (m *Message) Encode() (*[]byte, error) {

	jsonMsg, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return &jsonMsg, nil
}

func (m *Message) Decode(data *string) error {
	return json.Unmarshal([]byte(*data), m)
}
