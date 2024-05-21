package chat

import (
	"context"
	"fmt"
	"strconv"
	"yt/chatbot/lib/workermanager"
	"yt/chatbot/server/chat/model"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

const WELCOME_MESSAGE_FORMAT = "%s joined %s"

type Channel struct {
	model.IChannel
	Id      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	Private bool      `json:"private"`

	sessions map[*Session]bool

	registerSession   chan *Session
	unregisterSession chan *Session
	broadcast         chan *Message
	rds               *redis.Client
	ctx               context.Context
	ctxCancel         context.CancelFunc
	stopping          bool
	stopped           bool
}

// Get existing channel - if previously  created. Otherwise, create one.
func NewChannel(
	rds *redis.Client,
	channelDs model.IChannelDS,
	name string,
	private bool,
) (*Channel, error) {

	ctx, cancel := context.WithCancel(context.Background())

	channel := &Channel{
		Name:    name,
		Private: private,

		sessions:          make(map[*Session]bool),
		registerSession:   make(chan *Session),
		unregisterSession: make(chan *Session),
		broadcast:         make(chan *Message),
		rds:               rds,
		ctx:               ctx,
		ctxCancel:         cancel,
		stopping:          false,
		stopped:           false,
	}

	// Load from Data source
	chDs, err := channelDs.Get(name)
	if err != nil {
		logger.Error("Channel not found in DS or error: " + err.Error())
		return nil, err
	}
	if chDs != nil {
		logger.Trace("Restored channel: " + chDs.GetName())
		channel.Private = chDs.IsPrivate()
	} else {
		channel.Id = uuid.New()
		err = channelDs.Add(channel)
		if err != nil {
			logger.Error("Add channel to repo failed: " + err.Error())
			return nil, err
		}
	}

	channel.Start()
	return channel, nil
}

func (m *Channel) stop() {

	m.stopping = true
	close(m.registerSession)
	close(m.unregisterSession)
	close(m.broadcast)

	logger.Trace(fmt.Sprintf("Channel stop. sessions: %d", len(m.sessions)))
	if len(m.sessions) == 0 {
		logger.Trace("Sending shutdown request")
		// Trigger when no session is active in channel
		m.ctxCancel()
	}

	logger.Trace(fmt.Sprintf("Num. sessions left on shutdown: %d", len(m.sessions)))
	m.stopped = true

}

func (m *Channel) Start() {

	wm := workermanager.GetInstance()
	ctx := context.Background()

	done := make(chan struct{})

	// Start subscriber worker
	//

	wm.StartWorker(func() {

		logger.Trace("Listening to channel messages...")

		// Subscribe to channel
		//

		pubsub := m.rds.Subscribe(ctx, m.Name)
		defer pubsub.Close()

		logger.Debug("Setup pubsub service")

		_, err := pubsub.Receive(ctx) // Wait until active
		if err != nil {
			logger.Error("Start channel error. Pubsub service error: " + err.Error())
			return
		}

		logger.Trace("Setup pubsub channels")

		ch := pubsub.Channel()
		done <- struct{}{} // Ready. Unblock parent.

		// Broadcast new subscriber to members of channel
		//
		logger.Trace("Monitor channel messages to other members in the channel")

		terminate := false
		for !terminate {
			select {
			case <-m.ctx.Done():
				logger.Debug("Received a shutdown request. Winding down.")
				err := pubsub.Unsubscribe(ctx)
				if err != nil {
					logger.Error("Unsubscribe pubsub failed: " + err.Error())
				}
				terminate = true
			case msg := <-ch:
				message := &Message{}
				err := message.Decode(&msg.Payload)
				if err != nil {
					logger.Error(err.Error())
					return
				}
				for sess := range m.sessions {
					// Do not broadcast back to sender
					if message.Session.Subscriber.Name != sess.Subscriber.Name {
						logger.Debug("Send message to session: " + sess.Subscriber.Name)
						sess.Msg <- []byte(msg.Payload)
					}
				}
				message = nil
			}
		}

		logger.Trace("Listening to channel messages stopped.")

	}, "channelMessageprocessor")

	// Start session worker
	//

	wm.StartWorker(func() {

		logger.Trace("Listening to channel requests...")

		// Polling for requests. Process by request type
		terminate := false

		for !terminate {
			select {
			case <-m.ctx.Done():
				logger.Trace("Received a shutdown request. Winding down.")
				terminate = true
			// Join channel request
			case session, ok := <-m.registerSession:
				if ok {
					logger.Trace("Register session: " + session.Subscriber.Name + " in channel: " + m.Name)
					// Send a welcome message to non-private channel
					if !m.IsPrivate() {

						message := NewMessage(MSGTYPE_BCAST)
						message.RequestType = ACTION_SEND_MESSAGE
						message.Session = session
						message.ChannelName = m.Name
						message.Message = fmt.Sprintf(WELCOME_MESSAGE_FORMAT, session.Subscriber.Name, m.Name)
						encoded, _ := message.Encode()
						message = nil

						logger.Debug("Send message: " + string(*encoded))

						err := m.rds.Publish(ctx, m.GetName(), *encoded).Err()
						if err != nil {
							logger.Error(err.Error())
							terminate = true
						}
					}
					if !terminate {
						m.sessions[session] = true
					}
				}
			// Leave channel
			case session, ok := <-m.unregisterSession:
				if ok {
					// Session leaves channel
					delete(m.sessions, session)
					logger.Trace(
						fmt.Sprintf(
							"Unregister session. sessions: %d, stopping: %s",
							len(m.sessions),
							strconv.FormatBool(m.stopping),
						),
					)
					// Check for shutdown request.
					if m.stopping && len(m.sessions) == 0 {
						// Main process is waiting for workers to cleanup and exit.
						// Process all pending requests then stop workers.
						m.ctxCancel()
					}
				}
			// Send request
			case message, ok := <-m.broadcast:
				if ok {
					logger.Trace("broadcast message to sessions")
					encoded, err := message.Encode()
					if err != nil {
						logger.Warn(err.Error())
					} else {
						err := m.rds.Publish(ctx, m.GetName(), *encoded).Err()
						if err != nil {
							logger.Error(err.Error())
							terminate = true
						}
					}
				}
			}
		}

		logger.Trace("going away. Bye!")

	}, "ChannelRequesProcessor")

	<-done // Block until pubsub service is ready to shutdown
	close(done)

}

func (m *Channel) GetId() string {
	return m.Id.String()
}

func (m *Channel) GetName() string {
	return m.Name
}

func (m *Channel) IsPrivate() bool {
	return m.Private
}
