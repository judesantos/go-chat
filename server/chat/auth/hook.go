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
)

type ContextKey string

const CONTEXT_KEY = ContextKey("subscriber")

// Auth middleware - verify token (if provided). Otherwise, username is
// required for non-registered messaging?
func Authenticate(fn http.HandlerFunc) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ep := r.URL.Path
		if ep == "/login" {
			// Login precedes authentication. Ignore
			fn(w, r)
			return
		}

		var token, name, email string

		if r.Method == http.MethodPost {

			// User provided token acquired from prior login

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
			log.GetLogger().Debug("Process request: " + r.URL.RawQuery)

			s_token, tok := r.URL.Query()["jwt"]
			s_name, nok := r.URL.Query()["name"]
			s_email, eok := r.URL.Query()["email"]

			if tok && len(s_token) == 1 {
				token = s_token[0]
			} else if nok && len(s_name) == 1 {
				name = s_name[0]
			}
			if eok && len(s_email) == 1 {
				email = s_email[0]
			}

		} else {
			log.GetLogger().Warn("This is a different type of request")
		}

		srcIp := r.RemoteAddr
		userAgent := r.Header.Get("User-Agent")
		var msg string = ""

		if len(token) > 0 {

			userClaim, err := ValidateToken(token)
			if err != nil {
				msg = fmt.Sprintf("Authenticated request: (%s)[ip=%s;user-agent=%s]",
					ep, srcIp, userAgent)
				log.GetLogger().Warn("Forbidden request. Denied. " + msg)

				http.Error(w, "Forbidden", http.StatusForbidden)
			}

			// Audit. No exceptions
			msg = fmt.Sprintf("Authenticated request: (%s)[ip=%s;user-agent=%s,user=%s]",
				ep, srcIp, userAgent, userClaim.GetName())
			log.GetLogger().Info(msg)

			// Set as registered subscriber
			user := &datasource.Subscriber{
				Id:   userClaim.Id,
				Name: userClaim.Name,
				Type: datasource.SUBSCRIBER_TYPE_LOGIN,
			}
			ctx := context.WithValue(r.Context(), CONTEXT_KEY, user)
			// Call the endpoint handler
			fn(w, r.WithContext(ctx))

		} else if len(name) > 0 && len(email) > 0 {

			// Continue with request using anonymous user
			anon := datasource.Subscriber{
				Name:  name,
				Email: email,
				Type:  datasource.SUBSCRIBER_TYPE_ANONYMOUS,
			}
			// Audit. No exceptions
			msg = fmt.Sprintf("Anonymous request: (%s)[ip=%s;user-agent=%s,user=%s,email=%s]",
				ep, srcIp, userAgent, name, email)
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
