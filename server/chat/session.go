package chat

import (
	"time"
	"yt/chat/lib/workermanager"
	"yt/chat/server/chat/datasource"
	"yt/chat/server/chat/model"

	"github.com/gorilla/websocket"
)

const (
	CHAR_NEW_LINE = '\n'
	CHAR_SPACE    = " "

	MAX_MESSAGE_BUFFER_SIZE = 10000

	PONG_INTERVAL = 60 * time.Second
	PING_INTERVAL = (PONG_INTERVAL * 9) / 10

	WRITE_DELAY = 10 * time.Second
)

type Session struct {
	model.ISession `json:"-"`
	Subscriber     *datasource.Subscriber `json:"subscriber"`
	channels       map[*Channel]bool      `json:"-"`
	wsConn         *websocket.Conn        `json:"-"`
	wsSrvr         *Server                `json:"-"`
	Msg            chan []byte            `json:"-"`

	//stop chan struct{}
}

func NewSession(
	server *Server,
	wsConn *websocket.Conn,
	subscriber *datasource.Subscriber,
) error {
	logger.Info("Creating session for: " + subscriber.Name)

	session := &Session{
		Subscriber: subscriber,
		wsConn:     wsConn,
		wsSrvr:     server,
		Msg:        make(chan []byte),
		channels:   make(map[*Channel]bool),
		//stop:       make(chan struct{}),
	}

	// Let WS server know that we exist
	server.registerSession <- session

	mw := workermanager.GetInstance()
	mw.StartWorker(func() { session.responseHandler() }, "responseHandler")
	mw.StartWorker(func() { session.requestHandler() }, "requestHandler")

	logger.Info("Created session for: " + subscriber.Name)
	return nil
}

// Polling request handler - receive messages from client
func (m *Session) requestHandler() {

	logger.Trace("Listen for subscriber messages...")

	m.wsConn.SetReadLimit(MAX_MESSAGE_BUFFER_SIZE)
	m.wsConn.SetReadDeadline(time.Now().Add(PONG_INTERVAL))
	m.wsConn.SetPongHandler(func(string) error {
		m.wsConn.SetReadDeadline(time.Now().Add(PONG_INTERVAL))
		return nil
	})

	// Start WebSocket read routine

	for {
		// Unblocks on deadline hits
		_, msg, err := m.wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				logger.Error("WebSocket close error: " + err.Error())
				// Client closed connection
				m.disconnect()
			} else {
				logger.Error("WebSocket read error: " + err.Error())
			}
			break
		} else {
			// Process incoming message
			m.processSubscriberRequest(msg)
		}
	}

	logger.Trace("going away. Bye!")
}

// Polling response handler - send response through websocker writer
func (m *Session) responseHandler() {

	logger.Trace("Listen for session response...")

	ticker := time.NewTicker(PING_INTERVAL)
	defer func() {
		m.wsConn.Close()
		m.wsConn = nil
		ticker.Stop()
	}()

	stop := false

	for !stop {
		select {
		//case <-m.stop:
		//	stop = true
		case message, ok := <-m.Msg:
			if !ok {
				// The WsServer closed the channel.
				logger.Warn("Message channel closed. Bye!")
				m.wsConn.WriteMessage(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure,
						"Server closed session."),
				)
				stop = true
			} else {
				logger.Trace("Send message: " + string(message))
				m.wsConn.SetWriteDeadline(time.Now().Add(WRITE_DELAY))
				w, err := m.wsConn.NextWriter(websocket.TextMessage)
				if err != nil {
					logger.Error(err.Error())
					stop = true
				}
				if !stop {
					w.Write(message)

					// Attach queued chat messages to the current websocket message.
					n := len(m.Msg)
					for i := 0; i < n; i++ {
						w.Write([]byte{CHAR_NEW_LINE})
						w.Write(<-m.Msg)
					}

					if err := w.Close(); err != nil {
						logger.Error(err.Error())
						stop = true
					}
				}
			}
		case <-ticker.C:
			if !stop {
				m.wsConn.SetWriteDeadline(time.Now().Add(WRITE_DELAY))
				if err := m.wsConn.WriteMessage(websocket.PingMessage, nil); err != nil {
					logger.Error("Send ping error: " + err.Error())
					stop = true
				}
			}
		}
	}
	logger.Trace("Going away. Bye!")
}

func (m *Session) disconnect() {

	logger.Trace("Session disconnect: " + m.Subscriber.Name)
	close(m.Msg)
	m.Msg = nil

	// Tell server we quit
	m.wsSrvr.unregisterSession <- m
	for chn := range m.channels {
		select {
		case chn.unregisterSession <- m:
		default:
			logger.Error("unregister from channel: " + chn.Name + " failed. Channel is gone")
		}
	}
	m.channels = nil // Hasten GC

	// Close the session message channel
	//m.stop <- struct{}{}
	//close(m.stop)
	//m.stop = nil

	logger.Trace("Session disconnect done.")
}

func (m *Session) processSubscriberRequest(msg []byte) {

	var message Message
	logger.Trace("Received message: " + string(msg))

	strMsg := string(msg)
	err := message.Decode(&strMsg)
	if err != nil {
		logger.Error("Decode message failed for msg: " + string(msg) + ", error: " + err.Error())
		return
	}

	// Process subscriber request.
	// Send reply using same message id, requesttype
	//

	message.Session = m

	switch message.RequestType {
	case REQ_SEND_MESSAGE:

		notInChannel := true
		var ch *Channel

		for ch = range m.channels {
			if ch.Name == message.ChannelName {
				notInChannel = false
				break
			}
		}

		if notInChannel {
			// Session is not subscribed in the channel
			// Inform subscriber as so.
			message.MessageType = MSGTYPE_ACK
			message.Status = STATUS_FAILED
			message.Message = "Please subscribe to " + message.ChannelName

			encoded, err := message.Encode()
			if err != nil {
				logger.Error("Encoding failed: " + err.Error())
			} else {
				m.Msg <- *encoded
			}
		} else {
			// Send response to client
			message.MessageType = MSGTYPE_ACK
			message.Status = STATUS_SUCCESS

			encoded, err := message.Encode()
			if err != nil {
				logger.Error("Encoding failed: " + err.Error())
			} else {
				m.Msg <- *encoded
			}

			// broadcast to other subscribers.
			logger.Debug("Sending message to " + ch.Name)
			message.MessageType = MSGTYPE_BCAST
			ch.broadcast <- &message
		}

	case REQ_JOIN_CHANNEL:

		// Send response to subscriber
		//

		message.MessageType = MSGTYPE_ACK
		ok, err := m.joinChannel(message.ChannelName, message.Session.Subscriber)

		if err != nil {

			logger.Error("Failed to join channel: " + err.Error())

			message.Message = "Can not join channel " + message.ChannelName
			message.Status = STATUS_FAILED

			encoded, err := message.Encode()
			if err != nil {
				logger.Error("Encoding failed: " + err.Error())
			} else {
				m.Msg <- *encoded
			}
			return
		} else if ok {
			message.Message = "Welcome to " + message.ChannelName
		} else {
			message.Message = "Already joined " + message.ChannelName
		}

		message.RequestType = REQ_JOINED_CHANNEL
		message.Status = STATUS_SUCCESS

		encoded, err := message.Encode()
		if err != nil {
			logger.Error("Encoding failed: " + err.Error())
			return
		} else {
			m.Msg <- *encoded
		}

	case REQ_LEAVE_CHANNEL:

		// Send response to subscriber
		//

		message.MessageType = MSGTYPE_ACK
		message.Message = "Leave channel success"
		message.Status = STATUS_SUCCESS

		encoded, err := message.Encode()
		if err != nil {
			logger.Error("Encoding failed: " + err.Error())
			return
		} else {
			m.Msg <- *encoded
		}

		err = m.leaveChannel(message.ChannelName)
		if err != nil {
			logger.Error("Failed to leave " + message.ChannelName + ": " + err.Error())
		}

	case REQ_JOIN_PRIVATE_CHANNEL:
		m.joinPrivateChannel(&message)
	default:
		logger.Warn("Unknown request received. Ignored message: " + string(msg))
	}

}

func (m *Session) GetSubscriber() model.ISubscriber {
	return m.Subscriber
}

func (m *Session) GetChannels() *map[*Channel]bool {
	return &m.channels
}

func (m *Session) GetMessage() chan []byte {
	return m.Msg
}

func (m *Session) joinPrivateChannel(message *Message) {

}

func (m *Session) leaveChannel(channelName string) error {

	channel, err := m.wsSrvr.GetChannel(channelName)
	if err != nil || channel == nil {
		return err
	}

	// De-enlist session from the channel list
	channel.unregisterSession <- m
	delete(m.channels, channel)

	return nil
}

func (m *Session) joinChannel(channelName string, subscriber model.ISubscriber) (bool, error) {

	var channel *Channel
	var err error

	for ch := range m.channels {
		if ch.GetName() == channelName {
			logger.Trace("Get channel found: " + ch.GetName())
			channel = ch
			break
		}
	}

	if channel == nil {

		channel, err = m.wsSrvr.GetChannel(channelName)
		if err != nil {
			return false, err
		}

		m.channels[channel] = true
		channel.registerSession <- m
	}

	if subscriber == nil && channel.IsPrivate() {
		return true, nil
	}

	exists := false
	for sess := range channel.sessions {
		if sess.Subscriber.GetName() == m.Subscriber.Name {
			exists = true
			break
		}
	}

	if exists {
		return false, nil
	} else {
		return true, nil
	}
}
