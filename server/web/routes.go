package web

import (
	"net/http"
	"yt/chat/lib/config"
	"yt/chat/server/chat"
	"yt/chat/server/chat/auth"
	"yt/chat/server/chat/model"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
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
) *http.Handler {

	var handler http.Handler
	r := mux.NewRouter()

	// Setup CORS permissions - allowed settings are found in dotenv config
	// Allow sites listed in dotenv config
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{config.GetValue("ALLOWED_ORIGINS")},
		AllowCredentials: true,
	})
	handler = c.Handler(r)

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

	return &handler
}
