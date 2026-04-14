package testutil

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Crowley723/conduit/internal/config"
	"github.com/Crowley723/conduit/internal/middlewares"
	"github.com/Crowley723/conduit/internal/mocks"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/common/model"
	"go.uber.org/mock/gomock"
)

// chiContextKey is the context key used by chi router
type chiContextKeyType string

const chiContextKey chiContextKeyType = "chi.RouteContext"

// TestContext holds everything needed for testing
type TestContext struct {
	AppContext          *middlewares.AppContext
	Request             *http.Request
	Response            *httptest.ResponseRecorder
	MockController      *gomock.Controller
	MockSession         *mocks.MockSessionProvider
	MockOidcProvider    *mocks.MockOIDCProvider
	MockStorageProvider *mocks.MockStorageProvider
	LogHandler          *TestLogHandler
}

func NewTestContext(t *testing.T) *TestContext {
	cfg := &config.Config{
		Features: &config.FeaturesConfig{
			MTLSManagement: config.MTLSManagement{
				Enabled: false,
			},
		},
	}

	logHandler := NewTestLogHandler()
	logger := slog.New(logHandler)

	ctrl := gomock.NewController(t)

	mockSession := mocks.NewMockSessionProvider(ctrl)
	mockOIDCProvider := mocks.NewMockOIDCProvider(ctrl)
	mockStorageProvider := mocks.NewMockStorageProvider(ctrl)

	rr := httptest.NewRecorder()

	appCtx := &middlewares.AppContext{
		Context:        context.Background(),
		Config:         cfg,
		Logger:         logger,
		SessionManager: mockSession,
		OIDCProvider:   mockOIDCProvider,
		Storage:        mockStorageProvider,
		Request:        nil,
		Response:       rr,
	}

	return &TestContext{
		AppContext:          appCtx,
		Request:             nil,
		Response:            rr,
		MockController:      ctrl,
		MockSession:         mockSession,
		MockOidcProvider:    mockOIDCProvider,
		MockStorageProvider: mockStorageProvider,
	}
}

// NewTestContextWithURL creates a complete test setup with sensible defaults
func NewTestContextWithURL(t *testing.T, method, url string) *TestContext {
	cfg := &config.Config{
		Features: &config.FeaturesConfig{
			MTLSManagement: config.MTLSManagement{
				Enabled: false,
			},
		},
	}

	logHandler := NewTestLogHandler()
	logger := slog.New(logHandler)

	ctrl := gomock.NewController(t)

	mockSession := mocks.NewMockSessionProvider(ctrl)
	mockOIDCProvider := mocks.NewMockOIDCProvider(ctrl)
	mockStorageProvider := mocks.NewMockStorageProvider(ctrl)

	req := httptest.NewRequest(method, url, nil)
	rr := httptest.NewRecorder()

	appCtx := &middlewares.AppContext{
		Context:        req.Context(),
		Config:         cfg,
		Logger:         logger,
		SessionManager: mockSession,
		OIDCProvider:   mockOIDCProvider,
		Storage:        mockStorageProvider,
		Request:        req,
		Response:       rr,
	}

	return &TestContext{
		AppContext:     appCtx,
		Request:        req,
		Response:       rr,
		MockController: ctrl, MockSession: mockSession,
		MockOidcProvider:    mockOIDCProvider,
		MockStorageProvider: mockStorageProvider,
		LogHandler:          logHandler,
	}
}

// NewTestContextWithRealCache creates a test context with a real MemCache instead of mock
func NewTestContextWithRealCache(method, url string) *TestContext {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.Config{}

	req := httptest.NewRequest(method, url, nil)
	rr := httptest.NewRecorder()

	appCtx := &middlewares.AppContext{
		Context:        req.Context(),
		Config:         cfg,
		Logger:         logger,
		SessionManager: nil,
		OIDCProvider:   nil,
		Storage:        nil,
		Request:        req,
		Response:       rr,
	}

	return &TestContext{
		AppContext: appCtx,
		Request:    req,
		Response:   rr,
	}
}

// Finish should be called at the end of tests to clean up mocks
func (tc *TestContext) Finish() {
	if tc.MockController != nil {
		tc.MockController.Finish()
	}
}

func (tc *TestContext) AssertLogContains(t *testing.T, level slog.Level, message string) {
	if !tc.LogHandler.ContainsMessage(level, message) {
		t.Errorf("Expected to find log entry with level %v containing message: %s", level, message)
	}
}

func (tc *TestContext) AssertLogCount(t *testing.T, level slog.Level, expectedCount int) {
	count := tc.LogHandler.CountByLevel(level)
	if count != expectedCount {
		t.Errorf("Expected %d log entries at level %v, got %d", expectedCount, level, count)
	}
}

func (tc *TestContext) GetLogRecords() []TestLogRecord {
	return tc.LogHandler.GetRecords()
}

func (tc *TestContext) ClearLogRecords() {
	tc.LogHandler.Reset()
}

// CallHandler executes a handler with the test context
func (tc *TestContext) CallHandler(handler middlewares.AppHandler) {
	handler(tc.AppContext)
}

// AssertStatus checks the HTTP status code
func (tc *TestContext) AssertStatus(t *testing.T, expectedStatus int) {
	if tc.Response.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, tc.Response.Code)
	}
}

// AssertContentType checks the content type header
func (tc *TestContext) AssertContentType(t *testing.T, expectedType string) {
	if ct := tc.Response.Header().Get("Content-Type"); ct != expectedType {
		t.Errorf("Expected content type %s, got %s", expectedType, ct)
	}
}

// GetJSONResponse parses the response body as JSON
func (tc *TestContext) GetJSONResponse(t *testing.T) map[string]interface{} {
	var response map[string]interface{}
	if err := json.Unmarshal(tc.Response.Body.Bytes(), &response); err != nil {
		t.Fatalf("Could not parse JSON response: %v", err)
	}
	return response
}

// GetJSONResponseArray parses the response body as a JSON array
func (tc *TestContext) GetJSONResponseArray(t *testing.T) []interface{} {
	var response []interface{}
	if err := json.Unmarshal(tc.Response.Body.Bytes(), &response); err != nil {
		t.Fatalf("Could not parse JSON array response: %v", err)
	}
	return response
}

// AssertJSONField checks a specific field in a JSON response
func (tc *TestContext) AssertJSONField(t *testing.T, field string, expected any) {
	response := tc.GetJSONResponse(t)
	if actual, ok := response[field]; !ok || actual != expected {
		t.Errorf("Expected %s to be %s, got %v", field, expected, response[field])
	}
}

func (tc *TestContext) AssertJSONBool(t *testing.T, field string, expected bool) {
	response := tc.GetJSONResponse(t)
	actual, exists := response[field]

	if !exists {
		t.Errorf("Field %s not found in response", field)
		return
	}

	actualBool, ok := actual.(bool)
	if !ok {
		t.Errorf("Expected %s to be a boolean, got %T", field, actual)
		return
	}

	if actualBool != expected {
		t.Errorf("Expected %s to be %v, got %v", field, expected, actualBool)
	}
}

// AssertJSONString checks a specific string field in a JSON response
func (tc *TestContext) AssertJSONString(t *testing.T, field string, expected string) {
	response := tc.GetJSONResponse(t)
	actual, exists := response[field]

	if !exists {
		t.Errorf("Field %s not found in response", field)
		return
	}

	actualString, ok := actual.(string)
	if !ok {
		t.Errorf("Expected %s to be a string, got %T", field, actual)
		return
	}

	if actualString != expected {
		t.Errorf("Expected %s to be %q, got %q", field, expected, actualString)
	}
}

// AssertJSONObject validates an object field with expected key-value pairs
func (tc *TestContext) AssertJSONObject(t *testing.T, field string, expectedFields map[string]interface{}) {
	response := tc.GetJSONResponse(t)
	actual, exists := response[field]

	if !exists {
		t.Errorf("Field %s not found in response", field)
		return
	}

	actualObj, ok := actual.(map[string]interface{})
	if !ok {
		t.Errorf("Expected %s to be an object, got %T", field, actual)
		return
	}

	for key, expectedValue := range expectedFields {
		if actualValue, keyExists := actualObj[key]; !keyExists {
			t.Errorf("Expected field %s.%s to exist", field, key)
		} else if actualValue != expectedValue {
			t.Errorf("Expected %s.%s to be %v, got %v", field, key, expectedValue, actualValue)
		}
	}
}

// AssertUser validates a user object in the JSON response
func (tc *TestContext) AssertUser(t *testing.T, field string, expectedUser interface{}) {
	response := tc.GetJSONResponse(t)
	actual, exists := response[field]

	if !exists {
		t.Errorf("Field %s not found in response", field)
		return
	}

	user, ok := actual.(map[string]interface{})
	if !ok {
		t.Errorf("Expected %s to be a user object, got %T", field, actual)
		return
	}

	// Handle different user types - you'll need to import your models package
	switch u := expectedUser.(type) {
	case map[string]interface{}:
		// Compare as key-value pairs
		for key, expectedValue := range u {
			if actualValue, keyExists := user[key]; !keyExists {
				t.Errorf("Expected field %s.%s to exist", field, key)
			} else if actualValue != expectedValue {
				t.Errorf("Expected %s.%s to be %v, got %v", field, key, expectedValue, actualValue)
			}
		}
	default:
		// For any struct type, convert to map for comparison
		userBytes, err := json.Marshal(expectedUser)
		if err != nil {
			t.Errorf("Failed to marshal expected user: %v", err)
			return
		}

		var expectedUserMap map[string]interface{}
		if err := json.Unmarshal(userBytes, &expectedUserMap); err != nil {
			t.Errorf("Failed to unmarshal expected user: %v", err)
			return
		}

		// Compare only non-empty/non-nil fields from expected user
		for key, expectedValue := range expectedUserMap {
			// Skip nil values and empty strings unless they're explicitly set
			if expectedValue == nil || expectedValue == "" {
				continue
			}

			if actualValue, keyExists := user[key]; !keyExists {
				t.Errorf("Expected field %s.%s to exist", field, key)
			} else if actualValue != expectedValue {
				t.Errorf("Expected %s.%s to be %v, got %v", field, key, expectedValue, actualValue)
			}
		}
	}
}

// WithConfig allows you to override the default config for specific tests
func (tc *TestContext) WithConfig(cfg *config.Config) *TestContext {
	tc.AppContext.Config = cfg
	return tc
}

// WithLogger allows you to override the default logger for specific tests
func (tc *TestContext) WithLogger(logger *slog.Logger) *TestContext {
	tc.AppContext.Logger = logger
	return tc
}

// WithSessionManager allows you to override the session manager with a different mock or implementation
func (tc *TestContext) WithSessionManager(sm middlewares.SessionProvider) *TestContext {
	tc.AppContext.SessionManager = sm
	return tc
}

func (tc *TestContext) WithQueryParam(key, value string) *TestContext {
	q := tc.Request.URL.Query()
	q.Add(key, value)
	tc.Request.URL.RawQuery = q.Encode()
	return tc
}

// WithURLParam sets a URL parameter in the chi route context (for path parameters like /resource/{id})
func (tc *TestContext) WithURLParam(key, value string) *TestContext {
	rctx := tc.Request.Context().Value(chiContextKey)
	if rctx == nil {
		// Create a new chi context if one doesn't exist
		ctx := chi.NewRouteContext()
		ctx.URLParams.Add(key, value)
		tc.Request = tc.Request.WithContext(context.WithValue(tc.Request.Context(), chiContextKey, ctx))
		tc.AppContext.Request = tc.Request
		tc.AppContext.Context = tc.Request.Context()
	} else if chiCtx, ok := rctx.(*chi.Context); ok {
		chiCtx.URLParams.Add(key, value)
	}
	return tc
}

func (tc *TestContext) AssertLocationHeader(t *testing.T, expectedURL string) {
	location := tc.Response.Header().Get("Location")
	if location != expectedURL {
		t.Errorf("Expected redirect Location header to be '%s', got '%s'", expectedURL, location)
	}
}

func (tc *TestContext) AssertLogsContainMessage(t *testing.T, level slog.Level, message string) {
	if !tc.LogHandler.ContainsMessage(level, message) {
		t.Errorf("Expected logs to contain '%s'", message)
	}
}

func (tc *TestContext) WithHeader(key, value string) *TestContext {
	tc.Request.Header.Set(key, value)
	return tc
}

func (tc *TestContext) AssertJSONArrayLength(t *testing.T, expected int) {
	response := tc.GetJSONResponseArray(t)
	if len(response) != expected {
		t.Errorf("Expected JSON array length %d, got %d", expected, len(response))
	}
}

// WithRequest allows you to set a custom request (useful for tests that don't use URL constructor)
func (tc *TestContext) WithRequest(req *http.Request) *TestContext {
	tc.Request = req
	tc.AppContext.Request = req
	tc.AppContext.Context = req.Context()
	return tc
}

// ExpectSessionIsAuthenticated sets up an expectation for session.IsAuthenticated()
func (tc *TestContext) ExpectSessionIsAuthenticated(result bool) *gomock.Call {
	return tc.MockSession.EXPECT().IsAuthenticated(tc.AppContext).Return(result)
}

// ExpectSessionGetUser sets up an expectation for session.GetUser()
func (tc *TestContext) ExpectSessionGetUser(user interface{}, ok bool) *gomock.Call {
	return tc.MockSession.EXPECT().GetUser(tc.AppContext).Return(user, ok)
}

// CreateSample creates a model.Sample for use in vectors
func (tc *TestContext) CreateSample(labels model.LabelSet, value float64, timestamp time.Time) *model.Sample {
	return &model.Sample{
		Metric:    model.Metric(labels),
		Value:     model.SampleValue(value),
		Timestamp: model.Time(timestamp.Unix() * 1000),
	}
}
