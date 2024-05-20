package main

import (
	"fmt"
	"net/url"
	"time"
	"yt/chatbot/lib/utils"
	"yt/chatbot/lib/utils/log"
	"yt/chatbot/server/chat"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	wsTarget = "ws://localhost:8080/ws"

	MSG_JOIN_CHANNEL_FMT  = `{"id":"%s", "messagetype": 0, "requesttype":"join-channel", "channel":"%s", "message":"hello %s", "subscriber":{"name":"%s"}}`
	MSG_SEND_CHANNEL_FMT  = `{"id":"%s", "messagetype": 0, "requesttype":"send-msg", "channel":"%s", "message":"hello %s, how are you doing?", "subscriber":{"name":"%s"}}`
	MSG_LEAVE_CHANNEL_FMT = `{"id":"%s", "messagetype": 0, "requesttype":"leave-channel", "channel":"%s", "message":"goodbye, %s!", "subscriber":{"name":"%s"}}`
)

var (
	logger = log.GetLogger()

	// Globals.
	// TODO: Unglobalize

	EXPECTED_PASSED_TESTS = 0
	passedTests           = 0
	conn                  *websocket.Conn
	msgCh                 chan *chat.Message
	wsError               chan error
)

const EXPECTED_TEST_RESPONSES = 6

func getConnection(serverURL string, user string) *websocket.Conn {

	var err error

	subscriber := user //"santzky"
	wsURL := fmt.Sprintf("%s?name=%s", serverURL, url.QueryEscape(subscriber))

	// Upgrade the HTTP connection to a WebSocket connection
	conn, _, err = websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		panic("Websocket connect failed! Please check" + wsTarget + " exists, or is online.")
	}

	return conn
}

func processResponseMessage(conn *websocket.Conn) {

	//logger.Info("Start processResponse")

	for {
		var message chat.Message
		err := conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				logger.Error("WebSocket closd error: " + err.Error())
				// Socket closed from parent, or server
				wsError <- err
			}
			break
		}

		//msgBytes, _ := message.Encode()
		//msg := string(*msgBytes)
		//logger.Info("Received message from server: " + msg)

		msgCh <- &message
	}

	//logger.Info("End processResponse")
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
		logger.Trace("Set read timeout failed: " + err.Error())
		return nil, err
	}

	err = conn.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		logger.Trace("Write failed: " + err.Error())
		return nil, err
	}

	// Wait for response.
	// Block until response received.
	//logger.Info("Wait response from server: " + msg)
	select {
	case resp := <-msgCh:
		//logger.Info("Received message from server: " + msg)
		return resp, nil
	case <-wsError:
		return nil, err
	}
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
func joinChannelForSubscriber(channel string, user string) bool {

	// Send join-channel 'channel2' request
	//

	msg := getJoinChannelMessage(channel, user)
	logger.Info("Send message: " + msg)
	resp, err := sendMessageWaitForResponse(msg)
	if err != nil {
		logger.Error("Failed to send 'join-channel' message: " + err.Error())
		return false
	}

	if !validateResponse(&msg, resp) {
		logger.Error("Join channel request failed.")
		return false
	}

	// Send message 'send-msg' to channel 'channel1'
	//

	msg = getSendChannelMessage(channel, user)
	logger.Info("Send message: " + msg)
	resp, err = sendMessageWaitForResponse(msg)
	if err != nil {
		logger.Error("Failed to send 'send-msg' message: " + err.Error())
		return false
	}

	if !validateResponse(&msg, resp) {
		logger.Error("Send message to channel failed.")
		return false
	}

	// TODO: server closes connection too early. Check.
	//
	timer := time.NewTimer(5 * time.Millisecond)
	<-timer.C

	// Send leave-channel 'channel1' request
	//

	msg = getLeaveChannelMessage(channel, user)
	logger.Info("Send message: " + msg)
	resp, err = sendMessageWaitForResponse(msg)
	if err != nil {
		logger.Error("Failed to send 'leave-channel' message: " + err.Error())
		return false
	}

	if !validateResponse(&msg, resp) {
		logger.Error("Leave channel request failed.")
		return false
	}

	return true
}

func runTest(channel string, user string) (int, int) {

	timer := utils.PerfTimer{}
	timer.Start()

	msgCh = make(chan *chat.Message)
	wsError = make(chan error)

	conn := getConnection(wsTarget, user)
	defer func() {

		logger.Warn("Closing socket connection.")
		conn.Close()
		conn = nil

		close(msgCh)
		close(wsError)
	}()

	go processResponseMessage(conn)

	if !joinChannelForSubscriber(channel+"1", user) {
		return EXPECTED_PASSED_TESTS, passedTests
	}
	if !joinChannelForSubscriber(channel+"2", user) {
		return EXPECTED_PASSED_TESTS, passedTests
	}

	timer.Stop()
	logger.Debug(fmt.Sprintf("Test elapsed(ms): %.03f", timer.ElapsedMs()))

	passedTests++
	return EXPECTED_PASSED_TESTS, passedTests
}

func runRegressionTest(channel string, user string, repeat int) (int, int) {

	EXPECTED_PASSED_TESTS = repeat
	passedTests = 0

	for loop := 0; loop < repeat; loop++ {

		runTest(channel, fmt.Sprintf("%s%d", user, loop))

	}

	return EXPECTED_PASSED_TESTS, passedTests
}

func createTearDownSessions(user string, repeat int) (int, int) {

	EXPECTED_PASSED_TESTS = repeat

	for loop := 0; loop < repeat; loop++ {

		_user := fmt.Sprintf("%s%d", user, loop)
		conn := getConnection(wsTarget, _user)
		defer func() {
			logger.Warn("Closing socket connection.")
			conn.Close()
			conn = nil
		}()

		go processResponseMessage(conn)
		passedTests++
	}

	return EXPECTED_PASSED_TESTS, passedTests + 1
}

func main() {

	// Send subscriber 'santzky' join request
	//
	logger := log.GetLogger()
	defer func() {
		logger.Stop()
	}()

	channel := "channel"
	user := "santzky"

	EXPECTED_PASSED_TESTS = 0
	passedTests = -1

	run := 2
	repeatCount := 3000

	expected := 0
	passed := 0

	timer := utils.PerfTimer{}
	timer.Start()

	if run == 0 {

		conn := getConnection(wsTarget, user)
		defer func() {
			logger.Warn("Closing socket connection.")
			conn.Close()
			conn = nil
		}()

		go processResponseMessage(conn)
		expected, passed = runTest(channel, user)

	} else if run == 1 {

		expected, passed = runRegressionTest(channel, user, repeatCount)

	} else if run == 2 {

		expected, passed = createTearDownSessions(user, repeatCount)
	}

	if expected == passed {
		logger.Info(fmt.Sprintf("All %d tests passed!", expected))
	} else {
		logger.Warn(fmt.Sprintf("%d tests passed out of %d", passed, expected))
	}

	timer.Stop()
	logger.Debug(fmt.Sprintf("Test duration(ms): %.03f", timer.ElapsedMs()))
}
