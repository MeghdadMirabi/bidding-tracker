package integrationtests

import (
	bidding "bidding-tracker/internal/biddingService"
	model "bidding-tracker/internal/models"
	"bidding-tracker/internal/repository"
	"bidding-tracker/internal/server"
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// SetupTestRouter initializes the router with in-memory repository for integration testing.
func SetupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryRepo()
	service := bidding.NewBiddingService(repo)
	router := server.SetupRouter(service)
	return router
}

// ExecuteRequest executes an HTTP request and returns the response recorder.
func ExecuteRequest(t *testing.T, router *gin.Engine, method, url string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ExecuteRequestAndParse executes an HTTP request on the given router and parses the response
func ExecuteRequestAndParse(t *testing.T, router *gin.Engine, method, url string, body any) (map[string]any, *httptest.ResponseRecorder) {
	var reqBody []byte
	var err error

	switch v := body.(type) {
	case []byte:
		reqBody = v
	default:
		reqBody, err = json.Marshal(v)
		if err != nil {
			t.Fatalf("failed to marshal body: %v", err)
		}
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, url, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	var resp map[string]any
	if len(w.Body.Bytes()) > 0 {
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		if err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if w.Code == 201 {
			resp = resp["data"].(map[string]any)
		}
	}

	return resp, w
}

// SetupTestRouterWithItems initializes the router and seeds the repo with items.
func SetupTestRouterWithItems(items ...model.Item) *gin.Engine {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryRepo()

	for _, item := range items {
		repo.AddItem(item)
	}

	service := bidding.NewBiddingService(repo)
	router := server.SetupRouter(service)
	return router
}
