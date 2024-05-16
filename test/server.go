package main

import (
	"fmt"
	"net/url"
	"time"
	"yt/chatbot/lib/utils/log"
	"yt/chatbot/server/chat"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var (
	logger = log.GetLogger()

	wsTarget = "ws://localhost:8080/ws"
	conn     *websocket.Conn

	EXPECTED_PASSED_TESTS = 0

	passedTests = 0

	MSG_JOIN_CHANNEL_FMT  = `{"id":"%s", "messagetype": 0, "requesttype":"join-channel", "channel":"%s", "message":"hello %s", "subscriber":{"name":"%s"}}`
	MSG_SEND_CHANNEL_FMT  = `{"id":"%s", "messagetype": 0, "requesttype":"send-msg", "channel":"%s", "message":"hello %s, how are you doing?", "subscriber":{"name":"%s"}}`
	MSG_LEAVE_CHANNEL_FMT = `{"id":"%s", "messagetype": 0, "requesttype":"leave-channel", "channel":"%s", "message":"goodbye, %s!", "subscriber":{"name":"%s"}}`

	msgCh = make(chan *chat.Message)
)

const EXPECTED_TEST_RESPONSES = 6

func getConnection(serverURL string) *websocket.Conn {

	var err error

	subscriber := "santzky"
	wsURL := fmt.Sprintf("%s?name=%s", serverURL, url.QueryEscape(subscriber))

	// Upgrade the HTTP connection to a WebSocket connection
	conn, _, err = websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		panic("Websocket connect failed! Please check" + wsTarget + " exists, or is online.")
	}

	return conn
}

func processResponseMessage(conn *websocket.Conn) {

	for {
		var message chat.Message
		err := conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				logger.Error("WebSocket close error: " + err.Error())
			}
			// Socket closed from parent
			return
		}

		msgBytes, _ := message.Encode()
		msg := string(*msgBytes)
		logger.Info("Received message from server: " + msg)

		msgCh <- &message
	}
}

func validateResponse(send *string, outMsg *chat.Message) bool {

	var inMsg chat.Message

	inMsg.Decode(send)

	if inMsg.Id == outMsg.Id &&
		inMsg.RequestType == outMsg.RequestType &&
		outMsg.MessageType == chat.MSGTYPE_ACK &&
		outMsg.Status == "success" {

		return true
	}

	return false
}

func sendMessageWaitForResponse(msg string) (*chat.Message, error) {

	err := conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		return nil, err
	}

	err = conn.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		return nil, err
	}

	// Wait for response.
	// Block until response received.
	resp := <-msgCh

	return resp, nil
}

func getJoinChannelMessage(channel string, user string) string {
	id := uuid.New().String()
	return fmt.Sprintf(MSG_JOIN_CHANNEL_FMT, id, channel, channel, user)
}

func getSendChannelMessage(channel string, user string) string {
	id := uuid.New().String()
	return fmt.Sprintf(MSG_SEND_CHANNEL_FMT, id, channel, channel, user)
}

func getLeaveChannelMessage(channel string, user string) string {
	id := uuid.New().String()
	return fmt.Sprintf(MSG_LEAVE_CHANNEL_FMT, id, channel, channel, user)
}

// Create subscriber session
// Join the channel
// Send message to channel
// Leave channel
func joinChannelForSubscriber(channel string, user string) {

	// Send join-channel 'channel2' request
	//

	msg := getJoinChannelMessage(channel, user)
	//logger.Info("Send message: " + msg)
	resp, err := sendMessageWaitForResponse(msg)
	if err != nil {
		logger.Error("Failed to send 'join-channel' message: " + err.Error())
	}

	if !validateResponse(&msg, resp) {
		logger.Error("Join channel request failed.")
		return
	}

	// Send message 'send-msg' to channel 'channel1'
	//

	msg = getSendChannelMessage(channel, user)
	//logger.Info("Send message: " + msg)
	resp, err = sendMessageWaitForResponse(msg)
	if err != nil {
		logger.Error("Failed to send 'send-msg' message: " + err.Error())
	}

	if !validateResponse(&msg, resp) {
		logger.Error("Send message to channel failed.")
		return
	}

	// TODO: server closes connection too early. Check.
	//
	timer := time.NewTimer(5 * time.Millisecond)
	<-timer.C

	// Send leave-channel 'channel1' request
	//

	msg = getLeaveChannelMessage(channel, user)
	//logger.Info("Send message: " + msg)
	resp, err = sendMessageWaitForResponse(msg)
	if err != nil {
		logger.Error("Failed to send 'leave-channel' message: " + err.Error())
	}

	if !validateResponse(&msg, resp) {
		logger.Error("Leave channel request failed.")
	}

	passedTests++
}

func runTest(channel string, user string) {

	EXPECTED_PASSED_TESTS++

	joinChannelForSubscriber(channel+"1", user)
	joinChannelForSubscriber(channel+"2", user)

}

func runRegressionTest(channel string, user string, repeat int) {

	for loops := 0; loops < repeat; loops++ {

		runTest(channel, user)

		timer := time.NewTimer(5 * time.Millisecond)
		<-timer.C
	}

}

func main() {

	// Send subscriber 'santzky' join request
	//
	logger := log.GetLogger()

	defer func() {
		logger.Close()
	}()
	conn := getConnection(wsTarget)

	go processResponseMessage(conn)

	EXPECTED_PASSED_TESTS = 0
	passedTests = 0

	channel := "channel"
	user := "santzky"

	run := 1

	if run == 0 {
		runTest(channel, user)
	} else {
		runRegressionTest(channel, user, 700)
	}

	if EXPECTED_PASSED_TESTS == passedTests-1 {
		logger.Info(fmt.Sprintf("All %d tests passed!", EXPECTED_PASSED_TESTS))
	} else {
		logger.Warn(fmt.Sprintf("%d tests passed out of %d", passedTests-1, EXPECTED_PASSED_TESTS))
	}

	logger.Warn("Closing socket connection.")
	conn.Close()
}
