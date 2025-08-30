//go:build ignore
// +build ignore

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type UserHandler struct {
	userService    UserService
	cacheService   CacheService
	auditService   AuditService
	metricsService MetricsService
}

type UserService interface {
	GetByID(ctx context.Context, id int) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	UpdateLastSeen(ctx context.Context, userID int) error
	GetUserPreferences(ctx context.Context, userID int) (*UserPreferences, error)
}

type CacheService interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte, ttl time.Duration) error
}

type AuditService interface {
	LogUserAccess(userID int, ip string, userAgent string) error
}

type MetricsService interface {
	IncrementCounter(metric string, tags map[string]string)
	RecordDuration(metric string, duration time.Duration, tags map[string]string)
}

type User struct {
	ID          int          `json:"id"`
	Name        string       `json:"name"`
	Email       string       `json:"email"`
	Profile     *Profile     `json:"profile"`
	Settings    *Settings    `json:"settings"`
	Activity    *Activity    `json:"activity"`
	Preferences *UserPreferences `json:"preferences"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type Profile struct {
	DisplayName string  `json:"display_name"`
	Avatar      *Avatar `json:"avatar"`
	Bio         string  `json:"bio"`
	Location    string  `json:"location"`
}

type Avatar struct {
	URL       string `json:"url"`
	Thumbnail string `json:"thumbnail"`
}

type Settings struct {
	Theme            string `json:"theme"`
	Language         string `json:"language"`
	NotificationsEnabled bool   `json:"notifications_enabled"`
	PrivacyLevel     string `json:"privacy_level"`
}

type Activity struct {
	LastLogin    *time.Time `json:"last_login"`
	LastSeen     *time.Time `json:"last_seen"`
	LoginCount   int        `json:"login_count"`
	SessionCount int        `json:"session_count"`
}

type UserPreferences struct {
	EmailNotifications bool     `json:"email_notifications"`
	Categories         []string `json:"categories"`
	TimeZone          string   `json:"timezone"`
}

func NewUserHandler(userService UserService, cacheService CacheService, auditService AuditService, metricsService MetricsService) *UserHandler {
	return &UserHandler{
		userService:    userService,
		cacheService:   cacheService,
		auditService:   auditService,
		metricsService: metricsService,
	}
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	// Extract user ID from URL path
	vars := mux.Vars(r)
	userIDStr := vars["id"]
	
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Check cache first
	cacheKey := fmt.Sprintf("user:%d", userID)
	if cachedData, err := h.cacheService.Get(cacheKey); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write(cachedData)
		
		h.metricsService.IncrementCounter("user_requests", map[string]string{
			"cache": "hit",
			"endpoint": "get_user",
		})
		return
	}

	// Get user from service
	ctx := context.WithValue(r.Context(), "request_id", generateRequestID())
	user, err := h.userService.GetByID(ctx, userID)
	if err != nil {
		log.Printf("Error fetching user %d: %v", userID, err)
		http.Error(w, "User not found", http.StatusNotFound)
		
		h.metricsService.IncrementCounter("user_requests", map[string]string{
			"status": "error",
			"endpoint": "get_user",
		})
		return
	}

	// Update user's last seen timestamp asynchronously
	go func() {
		if err := h.userService.UpdateLastSeen(context.Background(), userID); err != nil {
			log.Printf("Failed to update last seen for user %d: %v", userID, err)
		}
	}()

	// Get user preferences
	preferences, err := h.userService.GetUserPreferences(ctx, userID)
	if err != nil {
		log.Printf("Failed to get preferences for user %d: %v", userID, err)
		// Continue without preferences rather than failing
	}
	user.Preferences = preferences

	// Build response - potential nil pointer dereferences
	userResponse := map[string]interface{}{
		"id":       user.ID,
		"email":    user.Email,
		"profile":  user.Profile.DisplayName, // Potential nil pointer if Profile is nil
		"settings": user.Settings.Theme,      // Potential nil pointer if Settings is nil
		"lastSeen": user.Activity.LastLogin.Format(time.RFC3339), // Multiple potential nil pointers
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	}
	
	// Access nested properties without validation
	if user.Profile != nil && user.Profile.Avatar != nil {
		userResponse["avatar"] = user.Profile.Avatar.URL // Assumes Avatar exists
		userResponse["avatar_thumbnail"] = user.Profile.Avatar.Thumbnail
	}
	
	// Add activity information with potential nil access
	if user.Activity != nil {
		userResponse["login_count"] = user.Activity.LoginCount
		userResponse["session_count"] = user.Activity.SessionCount
		if user.Activity.LastSeen != nil {
			userResponse["last_seen"] = user.Activity.LastSeen.Format(time.RFC3339)
		}
	}

	// Add preferences if available
	if user.Preferences != nil {
		userResponse["preferences"] = map[string]interface{}{
			"email_notifications": user.Preferences.EmailNotifications,
			"categories": user.Preferences.Categories,
			"timezone": user.Preferences.TimeZone,
		}
	}

	// Serialize response
	responseData, err := json.Marshal(userResponse)
	if err != nil {
		log.Printf("Error marshaling user response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Cache the response for 5 minutes
	go func() {
		if err := h.cacheService.Set(cacheKey, responseData, 5*time.Minute); err != nil {
			log.Printf("Failed to cache user data: %v", err)
		}
	}()

	// Log user access for audit
	clientIP := getClientIP(r)
	userAgent := r.Header.Get("User-Agent")
	go func() {
		if err := h.auditService.LogUserAccess(userID, clientIP, userAgent); err != nil {
			log.Printf("Failed to log user access: %v", err)
		}
	}()

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(responseData)

	// Record metrics
	duration := time.Since(startTime)
	h.metricsService.RecordDuration("user_request_duration", duration, map[string]string{
		"endpoint": "get_user",
		"status": "success",
	})
	
	// Log successful retrieval with potential nil pointer access
	log.Printf("Retrieved user: %s (%d) - %s", user.Email, user.ID, user.Profile.DisplayName)
}

func (h *UserHandler) GetUserByEmail(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "Email parameter required", http.StatusBadRequest)
		return
	}

	// Validate email format (basic validation)
	if !strings.Contains(email, "@") {
		http.Error(w, "Invalid email format", http.StatusBadRequest)
		return
	}

	ctx := context.WithValue(r.Context(), "request_id", generateRequestID())
	user, err := h.userService.GetByEmail(ctx, email)
	if err != nil {
		log.Printf("Error fetching user by email %s: %v", email, err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Potential nil pointer access without proper checks
	response := map[string]interface{}{
		"id": user.ID,
		"name": user.Name,
		"email": user.Email,
		"display_name": user.Profile.DisplayName, // Nil pointer risk
		"theme": user.Settings.Theme,             // Nil pointer risk
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP if multiple are present
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}
	
	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

func generateRequestID() string {
	// Simple request ID generation - in production would use UUID
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}