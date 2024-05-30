package chat

import (
	"encoding/json"
	"yt/chat/server/chat/auth"

	"github.com/google/uuid"
)

//
// Websocket request, response messaging
//

const (
	REQ_SEND_MESSAGE = "message"

	REQ_JOIN_CHANNEL   = "join-channel"
	REQ_LEAVE_CHANNEL  = "leave-channel"
	REQ_JOINED_CHANNEL = "joined-channel"

	REQ_SUBSCRIBER_JOINED = "subscriber-joined"
	REQ_SUBSCRIBER_LEFT   = "subscriber-left"

	REQ_JOIN_PRIVATE_CHANNEL = "join-private-channel"

	STATUS_SUCCESS = "success"
	STATUS_FAILED  = "failed"
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
	Id             uuid.UUID   `json:"id"`
	MessageType    MessageType `json:"messagetype"`
	RequestType    string      `json:"requesttype"`
	RequestSubType string      `json:"requestsubtype"`
	Message        string      `json:"message"`
	ChannelName    string      `json:"channelname"`
	Session        *Session    `json:"session"`
	Status         string      `json:"status"`
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

//
// Http request, response messaging
//

type AppResponse struct {
	Token   *auth.TokenMeta `json:"token"`
	Name    string          `json:"name"`
	Email   string          `json:"email"`
	Status  string          `json:"status"`
	Message string          `json:"message"`
}
