package web

import (
	"net/http"
	"yt/chat/lib/utils/log"
	"yt/chat/server/chat"
	"yt/chat/server/chat/model"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
)

var logger = log.GetLogger()

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func OnSocketConnect(
	resp http.ResponseWriter,
	req *http.Request,
	wsServer *chat.Server,
	rds *redis.Client,
	channelDs model.IChannelDS,
	subscriberDs model.ISubscriberDS,
) {
	logger.Info("Start OnSocketConnect()")

	// Get request url params
	_name, ok := req.URL.Query()["name"]

	if !ok || len(_name[0]) < 1 {
		logger.Error("Url Param 'name' is missing")
		return
	}
	name := _name[0]

	// Get websocket connection
	conn, err := upgrader.Upgrade(resp, req, nil)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	logger.Info("Creating new session for: " + name)

	chat.NewSession(wsServer, conn, name)
}

func OnSubscriber(resp http.ResponseWriter, req *http.Request, rds *redis.Client) {

}

func OnAllSubscribers(resp http.ResponseWriter, req *http.Request, rds *redis.Client) {

}
