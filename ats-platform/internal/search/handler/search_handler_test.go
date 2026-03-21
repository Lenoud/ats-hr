package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	 "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"

    "github.com/example/ats-platform/internal/search/model"
    "github.com/example/ats-platform/internal/search/repository"
)

// MockSearchService is a mock implementation of SearchService for testing
type MockSearchService struct {
    mock.Mock
}

func (m *MockSearchService) Search(ctx context.Context, filter repository.SearchFilter) (*repository.SearchResult, error) {
    args := m.Called(ctx, filter)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*repository.SearchResult), args.Error(1)
}

func (m *MockSearchService) IndexResume(ctx context.Context, doc *model.ResumeDocument) error {
    args := m.Called(ctx, doc)
    return args.Error(0)
}

func (m *MockSearchService) DeleteResume(ctx context.Context, id string) error {
    args := m.Called(ctx, id)
    return args.Error(0)
}

func (m *MockSearchService) UpdateResumeStatus(ctx context.Context, id string, status string) error {
    args := m.Called(ctx, id, status)
    return args.Error(0)
}

func TestSearchHandler_Search(t *testing.T) {
    gin.SetMode(gin.TestMode)

    mockSvc := new(MockSearchService)
    handler := NewSearchHandler(mockSvc)

    expectedResult := &repository.SearchResult{
        Documents: []model.ResumeDocument{
            {ResumeID: "1", Name: "John Doe", Email: "john@example.com", Status: "parsed"},
        },
        Total: 1,
    }

    mockSvc.On("Search", mock.Anything, mock.Anything).Return(expectedResult, nil)

    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Request = httptest.NewRequest("GET", "/api/v1/search?query=John", nil)

    handler.Search(c)

    assert.Equal(t, 200, w.Code)

    var resp map[string]any
    json.Unmarshal(w.Body.Bytes(), &resp)
    assert.Equal(t, float64(0), resp["code"])
    data := resp["data"].(map[string]any)
    assert.Equal(t, float64(1), data["total"])
}

