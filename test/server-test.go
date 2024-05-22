package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"time"
	"yt/chat/lib/utils"
	"yt/chat/lib/utils/log"
	"yt/chat/lib/workermanager"
	"yt/chat/server/chat"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	wsTarget = "ws://localhost:8080/ws"

	MSG_JOIN_CHANNEL_FMT  = `{"id":"%s", "messagetype": 0, "requesttype":"join-channel", "channel":"%s", "message":"hello %s", "subscriber":{"name":"%s"}}`
	MSG_SEND_CHANNEL_FMT  = `{"id":"%s", "messagetype": 0, "requesttype":"send-msg", "channel":"%s", "message":"hello %s, how are you doing?", "subscriber":{"name":"%s"}}`
	MSG_LEAVE_CHANNEL_FMT = `{"id":"%s", "messagetype": 0, "requesttype":"leave-channel", "channel":"%s", "message":"goodbye, %s!", "subscriber":{"name":"%s"}}`
)

var logger = log.GetLogger()

type TestConfigData struct {
	conn    *websocket.Conn
	wsError chan error
	msgCh   chan *chat.Message

	expectedPassedTests int
	passedTests         int

	serverHost string
	channel    string
	user       string
}

const (
	STATUS_SUCCESS = iota
	STATUS_CONNECTION_ERROR
	STATUS_TEST_FAILED
)

type TestStatus int

func NewTestConfig(server string, channel string, user string) *TestConfigData {
	return &TestConfigData{
		expectedPassedTests: 0,
		passedTests:         0,
		serverHost:          server,
		channel:             channel,
		user:                user,
	}
}

func getConnection(serverURL string, user string) *websocket.Conn {

	subscriber := user //"santzky"
	wsURL := fmt.Sprintf("%s?name=%s", serverURL, url.QueryEscape(subscriber))

	// Upgrade the HTTP connection to a WebSocket connection
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		logger.Error("Websocket connect failed! Please check " + wsTarget + " exists, or is online.")
		return nil
	}

	return conn
}

func processResponseMessage(config *TestConfigData) {

	for {
		var message chat.Message
		err := config.conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				logger.Error("WebSocket closed error: " + err.Error())
				// Socket closed from parent, or server
				config.wsError <- err
			} else {
				logger.Error("Websocket error: " + err.Error())
			}
			break
		}

		config.msgCh <- &message
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

func sendMessageWaitForResponse(config *TestConfigData, msg string) (*chat.Message, error) {

	err := config.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		logger.Trace("Set read timeout failed: " + err.Error())
		return nil, err
	}

	err = config.conn.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		logger.Trace("Write failed: " + err.Error())
		return nil, err
	}

	// Wait for response.
	// Block until response received.
	select {
	case resp := <-config.msgCh:
		return resp, nil
	case <-config.wsError:
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
func joinChannelForSubscriber(config *TestConfigData, channel string, user string) bool {

	// Send join-channel 'channel2' request
	//

	msg := getJoinChannelMessage(channel, user)
	logger.Info("Send message: " + msg)
	resp, err := sendMessageWaitForResponse(config, msg)
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
	resp, err = sendMessageWaitForResponse(config, msg)
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
	resp, err = sendMessageWaitForResponse(config, msg)
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

func runTest(cfg *TestConfigData) TestStatus {

	timer := utils.PerfTimer{}
	timer.Start()

	cfg.conn = getConnection(cfg.serverHost, cfg.user)
	if cfg.conn == nil {
		return STATUS_CONNECTION_ERROR
	}

	cfg.msgCh = make(chan *chat.Message)
	cfg.wsError = make(chan error)

	defer func() {
		logger.Warn("Closing socket connection")
		cfg.conn.Close()

		close(cfg.msgCh)
		close(cfg.wsError)
	}()

	go processResponseMessage(cfg)

	if !joinChannelForSubscriber(cfg, cfg.channel+"1", cfg.user) {
		return STATUS_TEST_FAILED
	}
	if !joinChannelForSubscriber(cfg, cfg.channel+"2", cfg.user) {
		return STATUS_TEST_FAILED
	}

	timer.Stop()
	logger.Debug(fmt.Sprintf("Test elapsed(ms): %.03f", timer.ElapsedMs()))

	cfg.passedTests++
	return STATUS_SUCCESS
}

func runRegressionTest(cfg *TestConfigData) bool {

	origName := cfg.user
	for loop := 0; loop < cfg.expectedPassedTests; loop++ {

		cfg.user = fmt.Sprintf("%s%d", origName, loop)
		if runTest(cfg) == STATUS_CONNECTION_ERROR {
			return false
		}

	}

	return true
}

func createTearDownSessions(cfg *TestConfigData) bool {

	origName := cfg.user

	for loop := 0; loop < cfg.expectedPassedTests; loop++ {

		cfg.user = fmt.Sprintf("%s%d", origName, loop)
		cfg.conn = getConnection(cfg.serverHost, cfg.user)
		if cfg.conn == nil {
			return false
		}
		cfg.msgCh = make(chan *chat.Message)
		cfg.wsError = make(chan error)

		go processResponseMessage(cfg)
		cfg.passedTests++

		cfg.conn.Close()
		close(cfg.msgCh)
		close(cfg.wsError)

	}

	return true
}

func runRegressionWorkers(cfg *TestConfigData, numWorkers int) bool {

	expectedPassed := cfg.expectedPassedTests
	cfg.expectedPassedTests = cfg.expectedPassedTests * numWorkers

	rw := workermanager.GetInstance()

	for loop := 0; loop < numWorkers; loop++ {

		_ch := fmt.Sprintf("%s%d", cfg.channel, loop)
		_usr := fmt.Sprintf("%s%d", cfg.user, loop)

		_cfg := NewTestConfig(cfg.serverHost, _ch, _usr)
		_cfg.expectedPassedTests = expectedPassed

		rw.StartWorker(
			func() {
				if runRegressionTest(_cfg) {
					cfg.passedTests += _cfg.passedTests
				}
			},
			fmt.Sprintf("runReqressionTest%d", loop),
		)

	}

	rw.WaitAll()
	return true
}

func usage() {
	fmt.Println("Usage: test -c [test_case_to_run: 1,2,3,4] -r [repeat_test_case_count. e.g.: 2]")
	fmt.Println("Options:")
	flag.PrintDefaults()
}

func main() {

	var run, repeatCount, workerCount int

	flag.IntVar(&run, "c", 1, "Test case option: [1, 2]; Default=1")
	flag.IntVar(&repeatCount, "r", 1, "Repeat test case. Default=1")
	flag.IntVar(&workerCount, "w", 1, "Num. workers (Only available for test #4.). Default=1")

	flag.Parse()

	if run > 4 {
		fmt.Println("Invalid test-case option: ", run)
		usage()
		os.Exit(-1)
	}

	// Send subscriber 'santzky' join request
	//
	logger := log.GetLogger()
	defer func() {
		logger.Stop()
	}()

	timer := utils.PerfTimer{}
	timer.Start()

	channel := "channel"
	user := "santzky"

	cfg := NewTestConfig(wsTarget, channel, user)
	cfg.expectedPassedTests = repeatCount

	if run == 2 {

		// Create websocket session, teardown session
		// Create new user, connection in each case
		createTearDownSessions(cfg)

	} else if run == 1 {

		// Repeat runRegressionTest * workerCount of parallel workers.
		runRegressionWorkers(cfg, workerCount)
	}

	if cfg.expectedPassedTests == cfg.passedTests {
		logger.Info(fmt.Sprintf("All %d tests passed!", cfg.expectedPassedTests))
	} else {
		logger.Warn(fmt.Sprintf("%d tests passed out of %d", cfg.passedTests, cfg.expectedPassedTests))
	}

	timer.Stop()
	logger.Debug(fmt.Sprintf("Test duration(ms): %.03f", timer.ElapsedMs()))
}
