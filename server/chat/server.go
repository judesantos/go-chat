package chat

import (
	"context"
	"yt/chatbot/lib/utils/log"
	"yt/chatbot/lib/workermanager"
	"yt/chatbot/server/chat/datasource"
	"yt/chatbot/server/chat/model"

	"github.com/go-redis/redis/v8"
)

var logger = log.GetLogger()

const MAIN_CHANNEL = "main-channel"

type Server struct {
	sessions          map[*Session]bool
	registerSession   chan *Session
	unregisterSession chan *Session
	channels          map[model.IChannel]bool
	subscribers       []model.ISubscriber
	channelDs         model.IChannelDS
	subsciberDs       model.ISubscriberDS
	rds               *redis.Client
	ctx               context.Context
	ctxCancel         context.CancelFunc
}

func NewServer(
	rds *redis.Client,
	channelDS model.IChannelDS,
	subscriberDS model.ISubscriberDS,
) *Server {

	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		sessions:          make(map[*Session]bool),
		registerSession:   make(chan *Session),
		unregisterSession: make(chan *Session),
		channels:          make(map[model.IChannel]bool),
		subsciberDs:       subscriberDS,
		channelDs:         channelDS,
		rds:               rds,
		ctx:               ctx,
		ctxCancel:         cancel,
	}
}

func (m *Server) Start() {
	m.listen()
}

func (m *Server) Stop() {

	logger.Trace("Stopping server")

	close(m.registerSession)
	close(m.unregisterSession)

	// channels may still be online with no sessions/subscribers
	// Shut down as well
	for ch := range m.channels {
		logger.Trace("Closing channel: " + ch.GetName())
		if !ch.(*Channel).stopped {
			ch.(*Channel).stop()
		}
	}
	m.channels = nil

	m.ctxCancel()
	logger.Trace("Stop success!")
}

func (m *Server) listen() {

	logger.Info("Listen for requests.")

	wm := workermanager.GetInstance()

	wm.StartWorker(func() { m.acceptSubscriberRequest() }, "acceptSubscriberRequest")
	wm.StartWorker(func() { m.acceptSessionRequest() }, "acceptSessionRequest")

}

func (m *Server) acceptSubscriberRequest() {

	logger.Info("Listen for subscriber requests...")

	ctx := context.Background()
	pubsub := m.rds.Subscribe(ctx, MAIN_CHANNEL)
	defer pubsub.Close()

	ch := pubsub.Channel()
	terminate := false

	for !terminate {
		select {
		case msg := <-ch:

			var message Message
			logger.Trace(msg.Payload)

			err := message.Decode(&msg.Payload)
			if err != nil {
				logger.Error(err.Error())
				terminate = true
			}

			if !terminate {

				logger.Trace("Subscriber request: " + message.Session.Subscriber.Name)
				logger.Trace("Subscriber requestType: " + message.RequestType)

				switch message.RequestType {
				case ACTION_JOINED_CHANNEL:
					m.joinedChannelRequest(message)
				case ACTION_LEAVE_CHANNEL:
					m.leftChannelRequest(message)
				case ACTION_PRIVATE_CHANNEL:
					m.joinPrivateChannel(message)
				}
			}
		case <-m.ctx.Done():
			logger.Trace("Got a cancellation event. Winding down...")
			err := pubsub.Unsubscribe(ctx)
			if err != nil {
				logger.Error("Unsubscribe pubsub failed: " + err.Error())
			}
			terminate = true
		}
	}

	logger.Trace("going away. Bye!")
}

func (m *Server) acceptSessionRequest() {

	logger.Trace("listen for session requests...")

	stop := false

	for !stop {
		select {
		case session, ok := <-m.registerSession:
			if ok {
				logger.Trace("Session register request: " + session.Subscriber.Name)
				m.registerSessionRequest(session)
			}
		case session, ok := <-m.unregisterSession:
			if ok {
				logger.Trace("Session unregister request: " + session.Subscriber.Name)
				m.unregisterSessionRequest(session)
			}
		case <-m.ctx.Done():
			logger.Trace("Got a cancellation event. Winding down...")
			stop = true
		}
	}

	logger.Trace("going away. Bye!")
}

func (m *Server) registerSessionRequest(session *Session) {

	logger.Trace("Register session: " + session.Subscriber.Name)

	subscr, err := m.subsciberDs.Get(session.Subscriber.Id)
	if err != nil {
		logger.Error(err.Error())
		panic(err)
		return
	}
	if subscr == nil {
		err = m.subsciberDs.Add(session.Subscriber)
		if err != nil {
			logger.Error(err.Error())
			panic(err)
			return
		}
	} else {
		session.Subscriber = subscr.(*datasource.Subscriber)
	}

	// Publish user in PubSub
	message := &Message{
		RequestType: ACTION_SUBSCRIBER_JOINED,
		Session:     session,
	}
	ctx := context.Background()
	encoded, _ := message.Encode()
	message = nil

	// Publish to all session in main channel?
	if err := m.rds.Publish(ctx, MAIN_CHANNEL, *encoded).Err(); err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// List online sessions
	var uniqueSubs = make(map[string]bool)
	for _, sub := range m.subscribers {
		if ok := uniqueSubs[sub.GetId()]; !ok {
			message := &Message{
				RequestType: ACTION_SUBSCRIBER_JOINED,
				Session:     session,
			}
			uniqueSubs[sub.GetId()] = true

			encoded, _ = message.Encode()
			message = nil
			session.Msg <- *encoded
		}
	}
	uniqueSubs = nil

	m.sessions[session] = true
	session.registered = true

	logger.Trace("End register session")
}

func (m *Server) unregisterSessionRequest(session *Session) {

	if _, ok := m.sessions[session]; ok {

		logger.Trace("Unregister session: " + session.Subscriber.Name)
		delete(m.sessions, session)
		// Publish user left in PubSub
		//message := Message{
		//	RequestType: ACTION_SUBSCRIBER_LEFT,
		//	Session:     session,
		//}
		//encoded, _ := message.Encode()

		//ctx := context.Background()
		//if err := m.rds.Publish(ctx, MAIN_CHANNEL, *encoded).Err(); err != nil {
		//	logger.Error(err.Error())
		//}
	}
}

func (m *Server) joinedChannelRequest(message Message) {

	// Add subscriber
	m.subscribers = append(m.subscribers, message.Session.Subscriber)
	// broadcast to all sessions?
	m.notifySessions(message)
}

func (m *Server) leftChannelRequest(message Message) {

	for i, subs := range m.subscribers {
		if subs.GetId() == message.Session.Subscriber.GetId() {
			m.subscribers[i] = m.subscribers[len(m.subscribers)-1]
			m.subscribers = m.subscribers[:len(m.subscribers)-1]
			break
		}
	}
	// broadcast to all sessions?
	m.notifySessions(message)
}

func (m *Server) joinPrivateChannel(message Message) {

	// Find relevant session
	for sess := range m.sessions {
		if sess.GetSubscriber().GetId() == message.Session.Subscriber.GetId() {
			sess.joinChannel(message.ChannelName, sess.Subscriber)
		}
	}

}

func (m *Server) notifySessions(msg Message) {

	bytes, err := msg.Encode()
	if err != nil {
		logger.Error(err.Error())
	}
	for sess := range m.sessions {
		sess.Msg <- *bytes
	}
}

// Create new channel, or reload from data source.
func (m *Server) GetChannel(channelName string) (*Channel, error) {

	var channel *Channel = nil
	// Find channel if previously created and is online
	for ch := range m.channels {
		if ch.GetName() == channelName {
			channel = ch.(*Channel)
			break
		}
	}

	if channel != nil {
		// Channel exists. Return this instance.
		return channel, nil
	}

	// No such channel found, create one.
	channel, err := NewChannel(m.rds, m.channelDs, channelName, false)
	if err != nil {
		return nil, err
	}

	m.channels[channel] = true
	return channel, nil
}
