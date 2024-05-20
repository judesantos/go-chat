package chat

import (
	"time"
	"yt/chatbot/lib/workermanager"
	"yt/chatbot/server/chat/datasource"
	"yt/chatbot/server/chat/model"

	"github.com/google/uuid"
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
	Subscriber     *datasource.Subscriber
	channels       map[*Channel]bool
	wsConn         *websocket.Conn
	wsSrvr         *Server
	Msg            chan []byte `json:"-"`

	stop       chan struct{}
	registered bool
}

func NewSession(
	server *Server,
	wsConn *websocket.Conn,
	subscriberName string,
) error {
	logger.Info("Session::NewSession() creating session for: " + subscriberName)

	subscriber := &datasource.Subscriber{
		Id:   uuid.New().String(),
		Name: subscriberName,
	}

	session := &Session{
		Subscriber: subscriber,
		wsConn:     wsConn,
		wsSrvr:     server,
		Msg:        make(chan []byte, 256),
		channels:   make(map[*Channel]bool),
		stop:       make(chan struct{}),
		registered: false,
	}

	mw := workermanager.GetInstance()

	mw.StartWorker(func() { session.responseHandler() }, "responseHandler")
	mw.StartWorker(func() { session.requestHandler() }, "requestHandler")

	server.registerSession <- session

	return nil
}

// Polling request handler - receive messages from client
func (m *Session) requestHandler() {

	logger.Trace("Listen for subscriber messages...")

	m.wsConn.SetReadLimit(MAX_MESSAGE_BUFFER_SIZE)
	m.wsConn.SetReadDeadline(time.Now().Add(WRITE_DELAY))
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
			} else {
				logger.Error("WebSocket read error: " + err.Error())
			}
			m.disconnect()
			//m.done <- struct{}{} // Tell responseHandler to git
			break
		} else {
			// Process incoming message
			m.processRequest(msg)
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
		ticker.Stop()
	}()

	stop := false

	for !stop {
		select {
		case <-m.stop:
			stop = true
		case message, ok := <-m.Msg:
			logger.Trace("Send message: " + string(message))
			m.wsConn.SetWriteDeadline(time.Now().Add(WRITE_DELAY))
			if !ok {
				// The WsServer closed the channel.
				logger.Warn("Message channel closed. Bye!")
				m.wsConn.WriteMessage(websocket.CloseMessage, []byte{})
				stop = true
			} else {
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
			m.wsConn.SetWriteDeadline(time.Now().Add(WRITE_DELAY))
			if err := m.wsConn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Error("Send ping error: " + err.Error())
				stop = true
			}
		}
	}
	logger.Trace("Going away. Bye!")
}

func (m *Session) disconnect() {

	logger.Trace("Session disconnect: " + m.Subscriber.Name)

	for !m.registered {
		logger.Trace("Wait on registered : " + m.Subscriber.Name)
		// A premature websocket disconnection happened before sesssion is fully establised.
		// Wait for server to finish the registration process.
		timer := time.NewTimer(5 * time.Millisecond)
		<-timer.C
	}

	for chn := range m.channels {
		select {
		case chn.unregisterSession <- m:
		default:
			logger.Error("unregister from channel: " + chn.Name + " failed. Channel is gone")
		}
	}
	m.channels = nil

	// Send a friendly socket close message to other side
	err := m.wsConn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure,
			"Server closed session."),
	)
	if err != nil {
		logger.Error("WebSocket close error: " + err.Error())
	}

	// Tell server we quit
	m.wsSrvr.unregisterSession <- m

	// Close the session message channel
	m.stop <- struct{}{}

	close(m.stop)
	close(m.Msg)

	logger.Trace("Session disconnect done.")
}

func (m *Session) processRequest(msg []byte) {

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
	case ACTION_SEND_MESSAGE:

		notInChannel := true
		for ch := range m.channels {
			if ch.Name == message.ChannelName {
				ch.broadcast <- &message
				notInChannel = false
			}
		}

		message.MessageType = MSGTYPE_ACK

		if notInChannel {
			// Session is not subscribed in the channel
			// Inform subscriber as so.
			message.Message = "Please subscribe to " + message.ChannelName
			message.Status = STATUS_FAILED

			encoded, err := message.Encode()
			if err != nil {
				logger.Error("Encoding failed: " + err.Error())
			} else {
				m.Msg <- *encoded
			}
		} else {

			message.Message = "Message sent to " + message.ChannelName
			message.Status = STATUS_SUCCESS

			encoded, err := message.Encode()
			if err != nil {
				logger.Error("Encoding failed: " + err.Error())
			} else {
				m.Msg <- *encoded
			}
		}

	case ACTION_JOIN_CHANNEL:

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
		} else if ok {
			message.Message = "Welcome to " + message.ChannelName
			message.RequestSubType = ACTION_JOINED_CHANNEL
		} else {
			message.Message = "Already joined " + message.ChannelName
			message.RequestSubType = ACTION_JOINED_CHANNEL
		}

		message.Status = STATUS_SUCCESS
		encoded, err := message.Encode()
		if err != nil {
			logger.Error("Encoding failed: " + err.Error())
		} else {
			m.Msg <- *encoded
		}

	case ACTION_LEAVE_CHANNEL:

		message.MessageType = MSGTYPE_ACK
		err := m.leaveChannel(message.ChannelName)

		if err != nil {
			message.Message = "Failed to leave " + message.ChannelName
			message.Status = STATUS_FAILED

			encoded, err := message.Encode()
			if err != nil {
				logger.Error("Encoding failed: " + err.Error())
			} else {
				m.Msg <- *encoded
			}

		} else {
			message.Message = "Bye!"
			message.RequestSubType = ACTION_LEFT_CHANNEL
			message.Status = STATUS_SUCCESS

			encoded, err := message.Encode()
			if err != nil {
				logger.Error("Encoding failed: " + err.Error())
			} else {
				m.Msg <- *encoded
			}
		}

	case ACTION_PRIVATE_CHANNEL:
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
