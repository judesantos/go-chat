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

	WRITE_DELAY = 10 * time.Millisecond
)

type Session struct {
	model.ISession `json:"-"`
	Subscriber     *datasource.Subscriber
	channels       map[*Channel]bool
	wsConn         *websocket.Conn
	wsSrvr         *Server
	Msg            chan []byte `json:"-"`

	done chan struct{}
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
		done:       make(chan struct{}),
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
			} else {
				logger.Error("WebSocket read error: " + err.Error())
			}
			m.done <- struct{}{} // Tell responseHandler to git
			break
		} else {
			// Process incoming message
			m.processRequest(msg)
		}
	}

	logger.Trace("going away. Bye!")
}

// Polling response handler - send response
func (m *Session) responseHandler() {

	logger.Trace("Listen for session response...")

	ticker := time.NewTicker(PING_INTERVAL)
	defer func() {
		m.wsConn.Close()
		ticker.Stop()
	}()

	terminate := false

	for !terminate {
		select {
		case <-m.done:
			terminate = true
		case message, ok := <-m.Msg:
			logger.Trace("Send message in socket: " + string(message))
			m.wsConn.SetWriteDeadline(time.Now().Add(WRITE_DELAY))
			if !ok {
				// The WsServer closed the channel.
				logger.Warn("Message channel closed. Bye!")
				m.wsConn.WriteMessage(websocket.CloseMessage, []byte{})
				terminate = true
			} else {
				w, err := m.wsConn.NextWriter(websocket.TextMessage)
				if err != nil {
					logger.Error(err.Error())
					terminate = true
				}
				if !terminate {
					w.Write(message)

					// Attach queued chat messages to the current websocket message.
					n := len(m.Msg)
					for i := 0; i < n; i++ {
						w.Write([]byte{CHAR_NEW_LINE})
						w.Write(<-m.Msg)
					}

					if err := w.Close(); err != nil {
						logger.Error(err.Error())
						terminate = true
					}
				}
			}
		case <-ticker.C:
			m.wsConn.SetWriteDeadline(time.Now().Add(WRITE_DELAY))
			if err := m.wsConn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Error("Send ping error: " + err.Error())
				terminate = true
			}
		}
	}
	logger.Trace("Going away. Bye!")
}

func (m *Session) Disconnect() {

	logger.Trace("Session disconnect: " + m.Subscriber.Name)

	for chn := range m.channels {
		select {
		case chn.unregisterSession <- m:
		default:
			logger.Error("unregister from channel: " + chn.Name + " failed. Channel is gone")
		}
	}

	// Send a friendly socket close message to other side
	err := m.wsConn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure,
			"Server closed session."),
	)

	if err != nil {
		logger.Error("WebSocket close error: " + err.Error())
	}

	// Close the session message channel
	close(m.Msg)
	close(m.done)
	m.channels = make(map[*Channel]bool)

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
		if notInChannel {
			// Session is not subscribed in the channel
			// Inform subscriber as so.
			resp := Message{
				RequestType: ACTION_NOTSUBSCRIBED_CHANNEL,
				ChannelName: message.ChannelName,
				Session:     m,
				Message:     "Please subscribe to " + message.ChannelName,
			}

			encoded, err := resp.Encode()
			if err != nil {
				logger.Error("Encoding failed: " + err.Error())
				return
			}
			m.Msg <- *encoded
		}

	case ACTION_JOIN_CHANNEL:
		m.joinChannel(message.ChannelName, message.Session.Subscriber)
	case ACTION_LEAVE_CHANNEL:
		m.leaveChannel(&message)
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

func (m *Session) leaveChannel(message *Message) {

	channel, err := m.wsSrvr.GetChannel(message.ChannelName)
	if err != nil || channel == nil {
		logger.Error(err.Error())
		return
	}

	// De-enlist session from the channel list
	channel.unregisterSession <- m
	delete(m.channels, channel)

	// Notify that we joined the room.
	resp := Message{
		RequestType: ACTION_LEFT_CHANNEL,
		ChannelName: message.ChannelName,
		Session:     m,
	}

	encoded, err := resp.Encode()
	if err != nil {
		logger.Error("Encoding failed: " + err.Error())
		return
	}

	// Send joined message to all subscribers on the channel
	m.Msg <- *encoded

}

func (m *Session) joinChannel(channelName string, subscriber model.ISubscriber) {

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
			logger.Error(err.Error())
			return
		}

		m.channels[channel] = true
	}

	if subscriber == nil && channel.IsPrivate() {
		return
	}

	exists := false
	for sess := range channel.sessions {
		if sess.Subscriber.GetName() == m.Subscriber.Name {
			exists = true
			break
		}
	}

	if !exists {
		// Do not register same session
		channel.registerSession <- m
		// Notify that we joined the room.
		message := Message{
			RequestType: ACTION_JOINED_CHANNEL,
			ChannelName: channelName,
			Session:     m,
		}

		encoded, err := message.Encode()
		if err != nil {
			logger.Error("Encoding failed: " + err.Error())
			return
		}

		// Send joined message to all subscribers on the channel
		m.Msg <- *encoded
	}

}
