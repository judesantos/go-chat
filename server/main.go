package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"yt/chat/lib/config"
	"yt/chat/lib/db"
	"yt/chat/lib/utils"
	"yt/chat/lib/utils/log"
	"yt/chat/lib/workermanager"
	"yt/chat/server/chat"
	"yt/chat/server/chat/datasource"
	"yt/chat/server/web"

	_ "net/http/pprof"

	"github.com/go-redis/redis/v8"
)

var logger *log.Logger = log.GetLogger()

func listenAndServe(server *http.Server) {

	environment := config.GetValue("ENV")
	logger.Info("Server running in the '" + environment +
		"' server. Listening on port" + server.Addr + "")

	if environment == "development" {
		// Development only
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Fatal("Error starting server: " + err.Error())
		}

	} else {
		// Production environments
		err := server.ListenAndServeTLS(".ssh/cert.pem", ".ssh/key.pem")
		if err != nil && err != http.ErrServerClosed {
			logger.Fatal("Error starting server: " + err.Error())
		}
	}

}

func main() {

	defer func() {
		logger.Stop()
	}()

	if config.GetValue("ENV") == "development" {
		// Profiler. See: net/http/pprof
		go func() {
			http.ListenAndServe("localhost:6060", nil)
		}()
	}

	parentTimer := utils.PerfTimer{}
	parentTimer.Start()

	timer := utils.PerfTimer{}

	logger.Info("Starting server...")
	logger.Info("Start persistence services...")

	// Persistence storage
	conn, err := db.GetConnection()
	//conn, err := db.GetConnection(config.GetValue("SERVER_DB"))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	logger.Info("Start transport services...")

	// PubSub service
	//

	addr := config.GetValue("PUBSUB_SERVER_HOST") + ":" + config.GetValue("PUBSUB_SERVER_PORT")

	timer.Start()

	rds := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: config.GetValue("PUBSUB_SERVER_PASS"),
	})
	defer rds.Close()

	err = rds.Ping(context.Background()).Err()
	if err != nil {
		println("Error connecting to Redis: ", err)
		return
	}

	timer.Stop()
	logger.Debug(fmt.Sprintf("Redis startup time(ms): %.3f", timer.ElapsedMs()))

	// Setup Data sources
	//

	channelDs := datasource.ChannelSqlite{DbConn: conn}
	subscriberDs := datasource.SubscriberSqlite{DbConn: conn}

	// Start chat server
	//
	logger.Info("Starting chat server...")

	timer.Start()

	wsServer := chat.NewServer(rds, &channelDs, &subscriberDs)
	// Start chat now - creates new thread and listen in the background
	wsServer.Start()

	timer.Stop()
	logger.Debug(fmt.Sprintf("Websocket server startup time(ms): %.3f", timer.ElapsedMs()))

	// Start web server
	//

	routes := web.GetRoutes(wsServer, rds, &channelDs, &subscriberDs)
	httpServer := &http.Server{
		Addr:    ":" + config.GetValue("SERVER_PORT"),
		Handler: routes,
	}

	// Start server now
	go listenAndServe(httpServer)

	// Listen for OS interrupts
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt) // Notify for interrupt signals (e.g., SIGINT)

	parentTimer.Stop()
	logger.Debug(fmt.Sprintf("Chat server startup time(ms): %.3f", parentTimer.ElapsedMs()))

	// Block until we get an interrupt signal
	<-sigCh
	close(sigCh)

	logger.Info("Received interrupt signal. Shutting down...")

	// Shutdown service, wait and complete ongoing tasks
	wsServer.Stop()

	logger.Info("Waiting on services to complete tasks...")

	workermanager.GetInstance().WaitAll() // Block until all workers are done.

	logger.Info("All tasks completed.")
	logger.Info("Shutting down http server...")

	// Create a new context for the server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Fatal("Error shutting down server: " + err.Error())
		os.Exit(-1)
	}

	logger.Info("Server stopped! goodbye.")
}
