package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"yt/chat/lib/utils/log"
	"yt/chat/server/chat"
	"yt/chat/server/chat/auth"
	"yt/chat/server/chat/datasource"
	"yt/chat/server/chat/model"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
)

var logger = log.GetLogger()

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins during development
		return true
	},
}

// Handle websocket connection request
func onSocketConnect(
	resp http.ResponseWriter,
	req *http.Request,
	wsServer *chat.Server,
	rds *redis.Client,
	channelDs model.IChannelDS,
	subscriberDs model.ISubscriberDS,
) {
	logger.Info("Start OnSocketConnect()")

	if wsServer.Stopping {
		sendErrorResponse(resp, "Connection refused.", http.StatusGone)
		return
	}

	ctxValue := req.Context().Value(auth.CONTEXT_KEY)
	if ctxValue == nil {
		log.GetLogger().Error("Not authorized")
		sendErrorResponse(resp, "Not authorized", http.StatusUnauthorized)
		return
	}

	subscr := ctxValue.(*datasource.Subscriber)

	// Get websocket connection
	conn, err := upgrader.Upgrade(resp, req, nil)
	if err != nil {
		logger.Error("Connection request failed: " + err.Error())
		sendErrorResponse(resp, err.Error(), http.StatusForbidden)
		return
	}

	logger.Debug("Creating new session for: " + subscr.Name)

	chat.NewSession(wsServer, conn, subscr)
}

// Handle subscriber login request
func onLogin(
	resp http.ResponseWriter,
	req *http.Request,
	wsServer *chat.Server,
	rds *redis.Client,
	channelDs model.IChannelDS,
	subscriberDs model.ISubscriberDS,
) {
	log.GetLogger().Debug("onLogin")

	if wsServer.Stopping {
		sendErrorResponse(resp, "Connection refused.", http.StatusGone)
		return
	}

	var subscr datasource.Subscriber

	// Try to decode the JSON request to a LoginUser
	err := json.NewDecoder(req.Body).Decode(&subscr)
	if err != nil {
		log.GetLogger().Error("Decode failed: " + err.Error())
		sendErrorResponse(resp, err.Error(), http.StatusBadRequest)
		return
	}
	// Find the user in the database by username
	subscr.Type = datasource.SUBSCRIBER_TYPE_LOGIN
	subs, err := subscriberDs.(*datasource.SubscriberPgsql).GetLoginInfo(&subscr)

	if err != nil {
		log.GetLogger().Error(err.Error())
		sendErrorResponse(resp, "Something went wrong", http.StatusInternalServerError)
		return
	}
	if subs == nil {
		// User not found or not registered
		log.GetLogger().Debug("Subscriber not found: " + subscr.Name)
		sendErrorResponse(resp, "Invalid user/password", http.StatusUnauthorized)
		return
	}

	recSubs := subs.(*datasource.Subscriber)

	// Check if the passwords match
	if !auth.Validate(
		subscr.Password,  // clear text
		recSubs.Password, // stored hash
	) {
		log.GetLogger().Debug("Invalid password: " + subscr.Name)
		sendErrorResponse(resp, "Invalid user/password", http.StatusUnauthorized)
		return
	}

	log.GetLogger().Debug("create token")

	// Create a JWT
	token, err := auth.NewToken(recSubs)

	if err != nil {
		sendErrorResponse(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	// Remove properties from response message

	jsonResp := chat.AppResponse{
		Token:  token,
		Name:   recSubs.Name,
		Email:  recSubs.Email,
		Status: chat.STATUS_SUCCESS,
	}

	respString, err := json.Marshal(jsonResp)
	if err != nil {
		sendErrorResponse(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Write(respString)

}

func onSubscriber(resp http.ResponseWriter, req *http.Request, rds *redis.Client) {

}

func onAllSubscribers(resp http.ResponseWriter, req *http.Request, rds *redis.Client) {

}

func sendErrorResponse(resp http.ResponseWriter, msg string, errCode int) {

	resp.Header().Set("Content-Type", "application/json")
	_msg := fmt.Sprintf(`{"status":"failed", "message": "%s"}`, msg)

	http.Error(resp, _msg, errCode)
}
