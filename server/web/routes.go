package web

import (
	"net/http"
	"yt/chatbot/server/chat"
	"yt/chatbot/server/chat/model"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

func getServiceHandler(
	wsSrvr *chat.Server,
	rds *redis.Client,
	channelDs model.IChannelDS,
	subscriberDs model.ISubscriberDS,
	h func(
		http.ResponseWriter,
		*http.Request,
		*chat.Server,
		*redis.Client,
		model.IChannelDS,
		model.ISubscriberDS,
	),
) func(http.ResponseWriter, *http.Request) {

	return func(resp http.ResponseWriter, req *http.Request) {
		h(resp, req, wsSrvr, rds, channelDs, subscriberDs)
	}
}

func GetRoutes(
	wsSrvr *chat.Server,
	rds *redis.Client,
	channelDs model.IChannelDS,
	subscriberDs model.ISubscriberDS,
) *mux.Router {

	r := mux.NewRouter()

	wsConnectReqHdlr := getServiceHandler(
		wsSrvr,
		rds,
		channelDs,
		subscriberDs,
		OnSocketConnect,
	)

	r.HandleFunc("/ws", wsConnectReqHdlr).Methods("GET")

	//channelsHdlr := getServiceHandler(rds, OnSubscriber)
	//r.HandleFunc("/channels", channelsHdlr).Methods("GET")

	//channelHdlr := getServiceHandler(rds, OnSubscriber)
	//r.HandleFunc("/channels", channelHdlr).Methods("GET")

	//subscriberHdlr := getServiceHandler(rds, OnSubscriber)
	//r.HandleFunc("/subscriber/{id}", subscriberHdlr).Methods("GET")

	//allSubscribersHdlr := getServiceHandler(rds, OnAllSubscribers)
	//r.HandleFunc("/subscribers", allSubscribersHdlr).Methods("GET")

	return r
}
