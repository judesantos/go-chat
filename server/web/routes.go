package web

import (
	"net/http"
	"yt/chat/lib/auth"
	"yt/chat/server/chat"
	"yt/chat/server/chat/model"

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

	return auth.Authenticate(
		func(resp http.ResponseWriter, req *http.Request) {
			h(resp, req, wsSrvr, rds, channelDs, subscriberDs)
		},
	)
}

func GetRoutes(
	wsSrvr *chat.Server,
	rds *redis.Client,
	channelDs model.IChannelDS,
	subscriberDs model.ISubscriberDS,
) *mux.Router {

	r := mux.NewRouter()

	// Subscriber socket connection request
	//

	f := r.HandleFunc("/ws", getServiceHandler(
		wsSrvr,
		rds,
		channelDs,
		subscriberDs,
		onSocketConnect,
	))
	f.Methods("GET")

	// Subscriber login requests
	//

	f = r.HandleFunc("/login", getServiceHandler(
		wsSrvr,
		rds,
		channelDs,
		subscriberDs,
		onLogin,
	))
	f.Methods("POST")

	return r
}
