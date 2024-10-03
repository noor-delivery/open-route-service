package main

import (
	"database/sql"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"io"
	"log"
	"main/logger"
	"net/http"
	"os"
	"strings"
	"time"
)

func In[T comparable](item T, items ...T) bool {
	for _, i := range items {
		if i == item {
			return true
		}
	}

	return false
}

// region Claims

type UserJwtClaims struct {
	Id        int            `json:"id" db:"id"`
	FirstName sql.NullString `json:"first_name" db:"first_name"`
	LastName  sql.NullString `json:"last_name" db:"last_name"`
	Role      string         `json:"role" db:"role"`
}

type MyCustomClaims struct {
	UserJwtClaims
	Type string `json:"type"`
	jwt.RegisteredClaims
}

// endregion

var jwtSecret []byte
var targetDomain string
var logs logger.LoggerInterface

// region Middleware

// Middleware to validate JWT token
func validateJWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.ParseWithClaims(tokenString, &MyCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}

			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Extract claims and validate role
		claims, ok := token.Claims.(*MyCustomClaims)
		if !ok || !In(claims.Role, "ADMIN", "USER", "COURIER", "MANAGER", "CLIENT", "VENDOR") {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// endregion

// region Handler

// Proxy handler to forward the request to OpenRouteService
func proxyHandler(w http.ResponseWriter, r *http.Request) {
	targetURL := targetDomain + r.URL.Path

	// Forward the request
	req, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "Error creating request", http.StatusInternalServerError)
		logs.ErrorF("Error creating request: %v", err)
		return
	}

	// Copy headers from original request
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Error forwarding request", http.StatusBadGateway)
		logs.ErrorF("Error forwarding request: %v", err)
		return
	}
	defer resp.Body.Close()

	// Copy response headers and body
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	if _, err = io.Copy(w, resp.Body); err != nil {
		http.Error(w, "Error proxying response", http.StatusInternalServerError)
		logs.ErrorF("Error proxying response: %v", err)
	}
}

// endregion

func main() {
	// Initialize things
	jwtSecret = []byte(os.Getenv("JWT_SIGN_KEY"))
	targetDomain = os.Getenv("TARGET_DOMAIN")
	logs = logger.NewLogger()
	logs.StartListener()

	// Define the proxy route
	http.Handle("/", validateJWT(http.HandlerFunc(proxyHandler)))

	if os.Getenv("APP_ENV") == "PROD" {
		// Get ssl file paths
		sslKeyPath := os.Getenv("SSL_KEY_PATH")
		sslCertPath := os.Getenv("SSL_CERT_PATH")

		// Start the server with SSL
		log.Println("Proxy server running on port 443 with SSL")
		if err := http.ListenAndServeTLS(":443", sslCertPath, sslKeyPath, nil); err != nil {
			log.Fatal(err)
		}
		return
	}

	// Start the server without SSL
	log.Println("Proxy server running on port 80")
	log.Fatal(http.ListenAndServe(":80", nil))
}
