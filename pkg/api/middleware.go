package api

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"
)

// ---- Context Keys ----

type contextKey string

const (
	contextKeyRequestID contextKey = "requestID"
	contextKeyAPIKey    contextKey = "apiKey"
)

// ---- Middleware Types ----

// AuthConfig configuração de autenticação
type AuthConfig struct {
	Enabled bool
	APIKeys []string // Lista de API keys válidas
}

// ---- Middlewares ----

// Logger middleware para logging de requisições
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrapper para capturar status code
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(ww, r)

		log.Printf(
			"[%s] %s %s %d %v",
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
			ww.statusCode,
			time.Since(start),
		)
	})
}

// Recoverer middleware para recuperação de panics
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC] %v", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"success":false,"error":{"code":"INTERNAL_ERROR","message":"Internal server error"}}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// CORS middleware para Cross-Origin Resource Sharing
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Verifica se a origem é permitida
			allowed := false
			for _, o := range allowedOrigins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
			w.Header().Set("Access-Control-Max-Age", "86400")

			// Handle preflight
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ContentTypeJSON middleware para forçar Content-Type JSON nas respostas
func ContentTypeJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// APIKeyAuth middleware para autenticação via API Key
func APIKeyAuth(config AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Tenta obter API key do header
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				// Tenta do Authorization header (Bearer token)
				auth := r.Header.Get("Authorization")
				if strings.HasPrefix(auth, "Bearer ") {
					apiKey = strings.TrimPrefix(auth, "Bearer ")
				}
			}

			if apiKey == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"success":false,"error":{"code":"UNAUTHORIZED","message":"API key required"}}`))
				return
			}

			// Valida API key
			valid := false
			for _, key := range config.APIKeys {
				if key == apiKey {
					valid = true
					break
				}
			}

			if !valid {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"success":false,"error":{"code":"INVALID_API_KEY","message":"Invalid API key"}}`))
				return
			}

			// Adiciona API key ao contexto
			ctx := context.WithValue(r.Context(), contextKeyAPIKey, apiKey)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequestID middleware para adicionar request ID
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		w.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(r.Context(), contextKeyRequestID, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RateLimit middleware básico de rate limiting (em memória)
func RateLimit(requestsPerMinute int) func(http.Handler) http.Handler {
	// Mapa simples de IP -> timestamps
	requests := make(map[string][]time.Time)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr

			now := time.Now()
			windowStart := now.Add(-time.Minute)

			// Remove requisições antigas
			var recent []time.Time
			for _, t := range requests[ip] {
				if t.After(windowStart) {
					recent = append(recent, t)
				}
			}
			requests[ip] = recent

			// Verifica limite
			if len(recent) >= requestsPerMinute {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"success":false,"error":{"code":"RATE_LIMITED","message":"Too many requests"}}`))
				return
			}

			// Adiciona requisição atual
			requests[ip] = append(requests[ip], now)

			next.ServeHTTP(w, r)
		})
	}
}

// ---- Helper Types ----

// responseWriter wrapper para capturar status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// ---- Helper Functions ----

func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}

// GetRequestID obtém o request ID do contexto
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(contextKeyRequestID).(string); ok {
		return id
	}
	return ""
}

// GetAPIKey obtém a API key do contexto
func GetAPIKey(ctx context.Context) string {
	if key, ok := ctx.Value(contextKeyAPIKey).(string); ok {
		return key
	}
	return ""
}
