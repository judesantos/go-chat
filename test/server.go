package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"
	"yt/chatbot/lib/utils/log"

	"github.com/gorilla/websocket"
)

var (
	logger = log.GetLogger()

	wsTarget = "ws://localhost:8080/ws"
	conn     *websocket.Conn

	passedTests = 0
)

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

func processResponseMessage(conn *websocket.Conn, done chan struct{}, cb func(status int)) {

	const EXPECTED_RESPONSE_COUNT = 6
	responseCount := 0

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			logger.Error("Error reading WebSocket message: " + err.Error())
			done <- struct{}{}
			cb(-1)
		} else {
			logger.Info("Received message from server: " + string(message))
		}

		msg := string(message)

		// Process expected responses - exit when done
		//

		if strings.Contains(msg, "joined-channel") {
			responseCount++
		}
		if strings.Contains(msg, "hello channel") {
			responseCount++
		}
		if strings.Contains(msg, "how are you doing") {
			responseCount++
		}
		if strings.Contains(msg, "left-channel") {
			responseCount++
		}

		if responseCount >= EXPECTED_RESPONSE_COUNT {
			logger.Info("All expected messages received. Quitting...")
			passedTests++
			done <- struct{}{}
			cb(0)
		}

	}
}

func createChannel1Subscriber1() {

	// Send join-channel 'channel1' request
	//
	msg := `{"requesttype":"join-channel", "channel":"channel1", "message":"hello channel1", "subscriber":{"name":"santzky"}}`
	err := conn.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		logger.Error("Failed to send 'join-channel' message: " + err.Error())
	}

	// Send message 'send-msg' to channel 'channel1'
	//
	msg = `{"requesttype":"send-msg", "channel":"channel1", "message":"hello channel1, how are you doing?", "subscriber":{"name":"santzky"}}`
	err = conn.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		logger.Error("Failed to send 'send-msg' message: " + err.Error())
	}

	// TODO: server closes connection too early. Check.
	//
	timer := time.NewTimer(5 * time.Millisecond)
	<-timer.C

	// Send leave-channel 'channel1' request
	//
	msg = `{"requesttype":"leave-channel", "channel":"channel1", "message":"goodbye, channel1!", "subscriber":{"name":"santzky"}}`
	err = conn.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		logger.Error("Failed to send 'leave-channel' message: " + err.Error())
	}
}

func createChannel2Subscriber1() {

	// Send join-channel 'channel2' request
	//
	msg := `{"requesttype":"join-channel", "channel":"channel2", "message":"hello channel1", "subscriber":{"name":"santzky"}}`
	err := conn.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		logger.Error("Failed to send 'join-channel' message: " + err.Error())
	}

	// Send message 'send-msg' to channel 'channel2'
	//
	msg = `{"requesttype":"send-msg", "channel":"channel2", "message":"hello channel2, how are you doing?", "subscriber":{"name":"santzky"}}`
	err = conn.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		logger.Error("Failed to send 'send-msg' message: " + err.Error())
	}

	// TODO: server closes connection too early. Check.
	//
	timer := time.NewTimer(5 * time.Millisecond)
	<-timer.C

	// Send leave-channel 'channel2' request
	//
	msg = `{"requesttype":"leave-channel", "channel":"channel2", "message":"goodbye, channel2!", "subscriber":{"name":"santzky"}}`
	err = conn.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		logger.Error("Failed to send 'leave-channel' message: " + err.Error())
	}
}

func runTest() {

	done := make(chan struct{})
	go processResponseMessage(conn, done, func(st int) {
		if st == 0 {
			logger.Info("All tests passed")
		} else {
			logger.Error("Test failed.")
		}
	})

	createChannel1Subscriber1()
	createChannel2Subscriber1()

	<-done

}

func runSingleTest() {

	conn := getConnection(wsTarget)
	if conn == nil {
		return
	}

	runTest()

	logger.Warn("Closing socket connection.")
	conn.Close()
}

func runTestOfTests() {

	conn := getConnection(wsTarget)
	if conn == nil {
		return
	}

	loops := 0

	for {
		if loops > 10 {
			break
		}
		loops++

		runTest()

		timer := time.NewTimer(5 * time.Millisecond)
		<-timer.C
	}

	logger.Warn("Closing socket connection.")
	conn.Close()
}

func runSuperTest() {

	const EXPECTEDPASSEDTESTS = 10 * 10
	loops := 0

	for {
		if loops > 10 {
			break
		}
		loops++

		runTestOfTests()

		timer := time.NewTimer(5 * time.Millisecond)
		<-timer.C
	}

	if EXPECTEDPASSEDTESTS == passedTests {
		logger.Info(fmt.Sprintf("All %d tests passed!", EXPECTEDPASSEDTESTS))
	} else {
		logger.Warn(fmt.Sprintf("%d tests passed out of %d", passedTests, EXPECTEDPASSEDTESTS))
	}
}

func main() {

	// Send subscriber 'santzky' join request
	//
	logger := log.GetLogger()
	defer func() {
		logger.Close()
	}()

	run := 1

	if run == 0 {
		runSingleTest()
	} else if run == 1 {
		runTestOfTests()
	} else {
		runSuperTest()
	}
}
