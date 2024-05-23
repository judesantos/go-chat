package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"yt/chat/lib/utils/log"
	"yt/chat/server/chat/datasource"

	"github.com/google/uuid"
)

type ContextKey string

const CONTEXT_KEY = ContextKey("subscriber")

// Auth middleware - verify token (if provided). Otherwise, username is
// required for non-registered messaging?
func Authenticate(fn http.HandlerFunc) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Check if the request method is OPTIONS (preflight request)
		if r.Method == "OPTIONS" {
			// Respond with a 200 status code
			w.WriteHeader(http.StatusOK)
			return
		}

		ep := r.URL.Path
		if ep == "/login" {
			// Login precedes authentication. Ignore
			fn(w, r)
			return
		}

		var token, name string

		if r.Method == http.MethodPost {

			// User provided token acquired by prior login

			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Error reading request body", http.StatusInternalServerError)
				return
			}
			// Restore body for the next handler - inefficient, why did they decide to do it this way?
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			// Decode the JSON data from the buffer. Preserve request.Body for subsequent reads
			var user datasource.Subscriber
			err = json.Unmarshal(bodyBytes, &user)
			if err != nil {
				http.Error(w, "Error decoding JSON data", http.StatusBadRequest)
				return
			}
			name = user.Name

		} else if r.Method == http.MethodGet {

			// Non-registered subscriber messaging

			var s_token []string
			var s_name []string

			s_token, tok := r.URL.Query()["token"]
			s_name, nok := r.URL.Query()["name"]

			if tok && len(s_token) == 1 {
				token = s_token[0]
			} else if nok && len(s_name) == 1 {
				name = s_name[0]
			}

		} else {
			log.GetLogger().Warn("This is a different type of request")
		}

		srcIp := r.RemoteAddr
		userAgent := r.Header.Get("User-Agent")
		var msg string = ""

		if len(token) > 0 {

			user, err := ValidateToken(token)
			// Audit. No exceptions
			msg = fmt.Sprintf("Authenticated request: (%s)[ip=%s;user-agent=%s,user=%s]",
				ep, srcIp, userAgent, user.GetName())
			log.GetLogger().Info(msg)

			if err != nil {

				log.GetLogger().Warn("Forbidden request. Denied.")
				http.Error(w, "Forbidden", http.StatusForbidden)

			} else {

				ctx := context.WithValue(r.Context(), CONTEXT_KEY, user)
				// Call the endpoint handler
				fn(w, r.WithContext(ctx))
			}

		} else if len(name) > 0 {

			// Continue with request using anonymous user
			anon := datasource.Subscriber{
				Id:   uuid.New().String(),
				Name: name,
				Type: datasource.SUBSCRIBER_TYPE_ANONYMOUS,
			}
			// Audit. No exceptions
			msg = fmt.Sprintf("Anonymous request: (%s)[ip=%s;user-agent=%s,user=%s]",
				ep, srcIp, userAgent, anon.Id)
			log.GetLogger().Info(msg)

			ctx := context.WithValue(r.Context(), CONTEXT_KEY, &anon)
			// Call the endpoint handler
			fn(w, r.WithContext(ctx))

		} else {

			// Audit. No exceptions
			msg = fmt.Sprintf("Invalid request: (%s)[ip=%s;user-agent=%s]. Denied.",
				ep, srcIp, userAgent)
			log.GetLogger().Warn(msg)

			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Login or userid required"))

		}
	})
}
