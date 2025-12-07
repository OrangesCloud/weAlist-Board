package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"project-board-api/internal/metrics"
)

// TokenValidationResponse represents the response from token validation for WebSocket authentication.
type TokenValidationResponse struct {
	UserID  string `json:"userId"`
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
}

// UserClient defines the interface for ALL User API and Auth interactions
type UserClient interface {
	// 기존 메서드 유지
	ValidateWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, token string) (bool, error)
	GetUserProfile(ctx context.Context, userID uuid.UUID, token string) (*UserProfile, error)
	GetWorkspaceProfile(ctx context.Context, workspaceID, userID uuid.UUID, token string) (*WorkspaceProfile, error)
	GetWorkspace(ctx context.Context, workspaceID uuid.UUID, token string) (*Workspace, error)

	// 💡 [추가] WebSocket 인증을 위한 메서드
	ValidateToken(ctx context.Context, tokenStr string) (uuid.UUID, error)
}

// WorkspaceValidationResponse represents the response from workspace validation endpoint
type WorkspaceValidationResponse struct {
	WorkspaceID uuid.UUID `json:"workspaceId"`
	UserID      uuid.UUID `json:"userId"`
	Valid       bool      `json:"valid"`
	IsValid     bool      `json:"isValid"`
}

// UserProfile represents basic user profile information
type UserProfile struct {
	UserID   uuid.UUID `json:"userId"`
	Email    string    `json:"email"`
	Provider string    `json:"provider"`
}

// WorkspaceProfile represents workspace-specific user profile
type WorkspaceProfile struct {
	ProfileID       uuid.UUID `json:"profileId"`
	WorkspaceID     uuid.UUID `json:"workspaceId"`
	UserID          uuid.UUID `json:"userId"`
	NickName        string    `json:"nickName"`
	Email           string    `json:"email"`
	ProfileImageURL string    `json:"profileImageUrl"`
}

// Workspace represents workspace information
type Workspace struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	OwnerID     uuid.UUID `json:"ownerId"`
	OwnerName   string    `json:"ownerName"`
	OwnerEmail  string    `json:"ownerEmail"`
	CreatedAt   string    `json:"createdAt"`
	UpdatedAt   string    `json:"updatedAt"`
}

// userClient implements UserClient interface
type userClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	logger     *zap.Logger
	metrics    *metrics.Metrics
}

// NewUserClient creates a new User API client
func NewUserClient(baseURL string, timeout time.Duration, logger *zap.Logger, m *metrics.Metrics) UserClient {
	return &userClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
		logger:  logger,
		metrics: m,
	}
}

// 💡 [추가] ValidateToken 메서드 구현 (WebSocket 인증 로직)
func (c *userClient) ValidateToken(ctx context.Context, tokenStr string) (uuid.UUID, error) {
	// 쿼리 파라미터로 토큰 전달
	url := fmt.Sprintf("%s/api/auth/validate-access-token?token=%s", c.baseURL, tokenStr)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("Failed to create validation request", zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("User service API connection failed", zap.Error(err))
		return uuid.Nil, fmt.Errorf("user service connection error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn("Token validation failed via User Service", zap.Int("status", resp.StatusCode))
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return uuid.Nil, errors.New("token validation failed: unauthorized or forbidden")
		}
		return uuid.Nil, fmt.Errorf("user service returned unexpected status: %d", resp.StatusCode)
	}

	var validationResponse TokenValidationResponse
	if err := json.NewDecoder(resp.Body).Decode(&validationResponse); err != nil {
		c.logger.Error("Failed to decode validation response", zap.Error(err))
		return uuid.Nil, fmt.Errorf("invalid response format from user service")
	}

	if !validationResponse.Valid {
		return uuid.Nil, fmt.Errorf("token explicitly marked invalid: %s", validationResponse.Message)
	}

	userID, err := uuid.Parse(validationResponse.UserID)
	if err != nil {
		c.logger.Error("Invalid UUID format in validation response", zap.String("id", validationResponse.UserID))
		return uuid.Nil, errors.New("invalid user ID format received")
	}

	return userID, nil
}

// buildURL constructs the full URL for User Service API calls
// It intelligently handles base URLs that may or may not include context path
//
// Examples:
//   - baseURL: http://user-service:8080/api/users, endpoint: /workspaces/123
//     -> http://user-service:8080/api/users/api/workspaces/123
//   - baseURL: http://user-service:8080, endpoint: /workspaces/123
//     -> http://user-service:8080/api/workspaces/123
func (c *userClient) buildURL(endpoint string) string {
	// Ensure endpoint starts with /
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	// Check if baseURL already contains context path (e.g., /api/users)
	hasContextPath := strings.Contains(c.baseURL, "/api/users") || strings.Contains(c.baseURL, "/api/boards")

	var finalURL string
	if hasContextPath {
		// Base URL already has context path, add /api before endpoint
		// This handles service-to-service communication in Docker where
		// user-service has context-path: /api/users
		finalURL = c.baseURL + "/api" + endpoint
	} else {
		// Base URL doesn't have context path (local development)
		// Just add /api before endpoint
		finalURL = c.baseURL + "/api" + endpoint
	}

	c.logger.Debug("Built URL for User Service",
		zap.String("base_url", c.baseURL),
		zap.String("endpoint", endpoint),
		zap.String("final_url", finalURL),
		zap.Bool("has_context_path", hasContextPath),
	)

	return finalURL
}

// ValidateWorkspaceMember validates if a user is a member of a workspace
func (c *userClient) ValidateWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, token string) (bool, error) {
	url := c.buildURL(fmt.Sprintf("/workspaces/%s/validate-member/%s", workspaceID.String(), userID.String()))

	c.logger.Debug("Validating workspace member",
		zap.String("url", url),
		zap.String("workspace_id", workspaceID.String()),
		zap.String("user_id", userID.String()),
	)

	var response WorkspaceValidationResponse
	if err := c.doRequest(ctx, "GET", url, token, &response); err != nil {
		c.logger.Error("Failed to validate workspace member",
			zap.Error(err),
			zap.String("workspace_id", workspaceID.String()),
			zap.String("user_id", userID.String()),
		)
		// Graceful degradation: return false on error
		return false, err
	}

	// Check both Valid and IsValid fields for compatibility
	isValid := response.Valid || response.IsValid

	c.logger.Debug("Workspace member validation result",
		zap.Bool("is_valid", isValid),
		zap.String("workspace_id", workspaceID.String()),
		zap.String("user_id", userID.String()),
	)

	return isValid, nil
}

// GetUserProfile retrieves user profile information
func (c *userClient) GetUserProfile(ctx context.Context, userID uuid.UUID, token string) (*UserProfile, error) {
	url := c.buildURL(fmt.Sprintf("/users/%s", userID.String()))

	c.logger.Debug("Getting user profile",
		zap.String("url", url),
		zap.String("user_id", userID.String()),
	)

	var profile UserProfile
	if err := c.doRequest(ctx, "GET", url, token, &profile); err != nil {
		c.logger.Error("Failed to get user profile",
			zap.Error(err),
			zap.String("user_id", userID.String()),
		)
		// Graceful degradation: return empty profile
		return &UserProfile{
			UserID: userID,
			Email:  "",
		}, nil
	}

	c.logger.Debug("User profile retrieved",
		zap.String("user_id", userID.String()),
		zap.String("email", profile.Email),
	)

	return &profile, nil
}

// GetWorkspaceProfile retrieves workspace-specific user profile
func (c *userClient) GetWorkspaceProfile(ctx context.Context, workspaceID, userID uuid.UUID, token string) (*WorkspaceProfile, error) {
	url := c.buildURL(fmt.Sprintf("/profiles/workspace/%s/user/%s", workspaceID.String(), userID.String()))

	c.logger.Debug("Getting workspace profile",
		zap.String("url", url),
		zap.String("workspace_id", workspaceID.String()),
		zap.String("user_id", userID.String()),
	)

	var profile WorkspaceProfile
	if err := c.doRequest(ctx, "GET", url, token, &profile); err != nil {
		c.logger.Error("Failed to get workspace profile",
			zap.Error(err),
			zap.String("workspace_id", workspaceID.String()),
			zap.String("user_id", userID.String()),
		)
		// Graceful degradation: return empty profile
		return &WorkspaceProfile{
			WorkspaceID: workspaceID,
			UserID:      userID,
			NickName:    "",
			Email:       "",
		}, nil
	}

	c.logger.Debug("Workspace profile retrieved",
		zap.String("workspace_id", workspaceID.String()),
		zap.String("user_id", userID.String()),
		zap.String("nickname", profile.NickName),
	)

	return &profile, nil
}

// GetWorkspace retrieves workspace information
func (c *userClient) GetWorkspace(ctx context.Context, workspaceID uuid.UUID, token string) (*Workspace, error) {
	url := c.buildURL(fmt.Sprintf("/workspaces/%s", workspaceID.String()))

	c.logger.Debug("Getting workspace",
		zap.String("url", url),
		zap.String("workspace_id", workspaceID.String()),
	)

	var workspace Workspace
	if err := c.doRequest(ctx, "GET", url, token, &workspace); err != nil {
		c.logger.Error("Failed to get workspace",
			zap.Error(err),
			zap.String("workspace_id", workspaceID.String()),
		)
		// Graceful degradation: return empty workspace
		return &Workspace{
			ID:   workspaceID,
			Name: "",
		}, nil
	}

	c.logger.Debug("Workspace retrieved",
		zap.String("workspace_id", workspaceID.String()),
		zap.String("name", workspace.Name),
	)

	return &workspace, nil
}

// doRequest performs an HTTP request with the given parameters
func (c *userClient) doRequest(ctx context.Context, method, url, token string, result interface{}) error {
	// Track request start time for processing time calculation
	startTime := time.Now()

	// Enhanced logging: Log detailed request information before making the call
	c.logger.Info("Making request to User Service",
		zap.String("method", method),
		zap.String("url", url),
		zap.String("base_url", c.baseURL),
		zap.Bool("has_token", token != ""),
		zap.Duration("timeout", c.timeout),
	)

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		c.logger.Error("Failed to create HTTP request",
			zap.Error(err),
			zap.String("method", method),
			zap.String("url", url),
		)
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	// Log request headers (excluding sensitive token value)
	c.logger.Debug("Request headers set",
		zap.String("content_type", req.Header.Get("Content-Type")),
		zap.Bool("has_authorization", req.Header.Get("Authorization") != ""),
	)

	// Execute request
	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime)

	// Record metrics
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}
	if c.metrics != nil {
		c.metrics.RecordExternalAPICall(url, method, statusCode, duration, err)
	}

	if err != nil {
		c.logger.Error("Failed to execute HTTP request",
			zap.Error(err),
			zap.String("method", method),
			zap.String("url", url),
			zap.Duration("processing_time", duration),
		)
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("Failed to read response body",
			zap.Error(err),
			zap.String("url", url),
			zap.Int("status_code", resp.StatusCode),
			zap.Duration("processing_time", time.Since(startTime)),
		)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Calculate processing time
	processingTime := time.Since(startTime)

	// Enhanced logging: Log response details with processing time
	bodyPreview := string(body)
	if len(bodyPreview) > 500 {
		bodyPreview = bodyPreview[:500] + "... (truncated)"
	}

	// Check status code and log accordingly
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Enhanced error logging with status code and response body
		c.logger.Error("User API returned non-success status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("url", url),
			zap.String("method", method),
			zap.String("response_body", string(body)),
			zap.Bool("has_token", token != ""),
			zap.Duration("processing_time", processingTime),
		)

		// Special handling for 403 Forbidden errors
		if resp.StatusCode == http.StatusForbidden {
			c.logger.Error("403 Forbidden error from User Service",
				zap.String("requested_url", url),
				zap.String("method", method),
				zap.Bool("token_present", token != ""),
				zap.String("response_body", string(body)),
				zap.Duration("processing_time", processingTime),
			)
		}

		return fmt.Errorf("user API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Success response logging with processing time
	c.logger.Info("Received successful response from User Service",
		zap.Int("status_code", resp.StatusCode),
		zap.String("url", url),
		zap.String("method", method),
		zap.Int("body_length", len(body)),
		zap.Duration("processing_time", processingTime),
	)

	c.logger.Debug("Response body preview",
		zap.String("url", url),
		zap.String("response_preview", bodyPreview),
	)

	// Parse response
	if err := json.Unmarshal(body, result); err != nil {
		c.logger.Error("Failed to parse response JSON",
			zap.Error(err),
			zap.String("url", url),
			zap.String("response_body", string(body)),
			zap.Duration("processing_time", processingTime),
		)
		return fmt.Errorf("failed to parse response: %w", err)
	}

	c.logger.Debug("Successfully parsed response",
		zap.String("url", url),
		zap.Duration("total_processing_time", processingTime),
	)

	return nil
}
