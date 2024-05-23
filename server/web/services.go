package web

import (
	"encoding/json"
	"net/http"
	"yt/chat/lib/auth"
	"yt/chat/lib/utils/log"
	"yt/chat/server/chat"
	"yt/chat/server/chat/datasource"
	"yt/chat/server/chat/model"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
)

var logger = log.GetLogger()

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
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

	ctxValue := req.Context().Value(auth.CONTEXT_KEY)
	if ctxValue == nil {
		log.GetLogger().Error("Not authenticated")
		http.Error(resp, "Not authenticated", http.StatusUnauthorized)
		return
	}

	subscr := ctxValue.(*datasource.Subscriber)

	// Get websocket connection
	conn, err := upgrader.Upgrade(resp, req, nil)
	if err != nil {
		logger.Error("Connection request failed: " + err.Error())
		http.Error(resp, "Something went wrong", http.StatusInternalServerError)
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

	var subscr datasource.Subscriber

	// Try to decode the JSON request to a LoginUser
	err := json.NewDecoder(req.Body).Decode(&subscr)
	if err != nil {
		log.GetLogger().Error("Decode failed: " + err.Error())
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	log.GetLogger().Debug("Find subscriber")

	// Find the user in the database by username
	subs, err := subscriberDs.Get(&subscr)
	if err != nil {
		http.Error(resp, "Something went wrong", http.StatusInternalServerError)
		return
	}
	if subs == nil {
		// User not found or not registered
		http.Error(resp, "Invalid user/password", http.StatusUnauthorized)
		return
	}

	log.GetLogger().Debug("validate subscriber")

	// Check if the passwords match
	ok, err := auth.Validate(
		subscr.Password,
		subs.(*datasource.Subscriber).Password,
	)

	if !ok || err != nil {
		errorResponse(resp)
		return
	}

	log.GetLogger().Debug("create token")

	// Create a JWT
	token, err := auth.NewToken(subs)

	if err != nil {
		errorResponse(resp)
		return
	}

	resp.Write([]byte(token))

}

func onSubscriber(resp http.ResponseWriter, req *http.Request, rds *redis.Client) {

}

func onAllSubscribers(resp http.ResponseWriter, req *http.Request, rds *redis.Client) {

}

func errorResponse(w http.ResponseWriter) {

	log.GetLogger().Debug("ErrorResponse")

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "error"}`))
}
