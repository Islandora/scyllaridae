package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	scyllaridae "github.com/lehigh-university-libraries/scyllaridae/internal/config"
	"github.com/lehigh-university-libraries/scyllaridae/pkg/api"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jwt"
)

type contextKey string

const cmdKey contextKey = "scyllaridaeCmd"
const msgKey contextKey = "scyllaridaeMsg"

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

func (s *Server) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		statusWriter := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		auth := ""
		if s.Config.ForwardAuth {
			auth = r.Header.Get("Authorization")
		}
		message, err := api.DecodeAlpacaMessage(r, auth)
		if err != nil {
			slog.Error("Error decoding alpaca message", "err", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		cmd, err := scyllaridae.BuildExecCommand(message, s.Config)
		if err != nil {
			slog.Error("Error building command", "err", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), cmdKey, cmd)
		ctx = context.WithValue(ctx, msgKey, message)
		next.ServeHTTP(statusWriter, r.WithContext(ctx))
		duration := time.Since(start)

		slog.Info("Incoming request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", statusWriter.statusCode,
			"duration", duration,
			"client_ip", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"command", cmd.String(),
		)
	})
}

// JWTAuthMiddleware validates a JWT token and adds claims to the context
func JWTAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := r.Header.Get("Authorization")
		if a == "" || len(a) <= 7 || !strings.EqualFold(a[:7], "bearer ") {
			if os.Getenv("SKIP_JWT_VERIFY") != "true" {
				http.Error(w, "Missing Authorization header", http.StatusBadRequest)
				return
			}
		}

		if os.Getenv("SKIP_JWT_VERIFY") != "true" {
			tokenString := a[7:]
			err := verifyJWT(tokenString)
			if err != nil {
				slog.Error("JWT verification failed", "err", err)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func verifyJWT(tokenString string) error {
	keySet, err := fetchJWKS()
	if err != nil {
		return fmt.Errorf("unable to fetch JWKS: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	token, err := jwt.Parse([]byte(tokenString),
		jwt.WithKeySet(keySet),
		jwt.WithContext(ctx),
		jwt.WithVerify(jwa.RS256, keySet),
	)
	if err != nil {
		return fmt.Errorf("unable to parse token: %v", err)
	}

	err = jwt.Validate(token)
	if err != nil {
		return fmt.Errorf("unable to validate token: %v", err)
	}

	return nil
}

// fetchJWKS fetches the JSON Web Key Set (JWKS) from the given URI
func fetchJWKS() (jwk.Set, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	jwksURI := os.Getenv("JWKS_URI")
	return jwk.Fetch(ctx, jwksURI)
}
