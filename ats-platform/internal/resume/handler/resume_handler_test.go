package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/example/ats-platform/internal/resume/model"
	"github.com/example/ats-platform/internal/resume/service"
)

// MockResumeService is a mock implementation of service.ResumeService
type MockResumeService struct {
	mock.Mock
}

func (m *MockResumeService) Create(ctx context.Context, input service.CreateResumeInput) (*model.Resume, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Resume), args.Error(1)
}

func (m *MockResumeService) GetByID(ctx context.Context, id uuid.UUID) (*model.Resume, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Resume), args.Error(1)
}

func (m *MockResumeService) List(ctx context.Context, page, pageSize int, status, source string) ([]model.Resume, int64, error) {
	args := m.Called(ctx, page, pageSize, status, source)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]model.Resume), args.Get(1).(int64), args.Error(2)
}

func (m *MockResumeService) Update(ctx context.Context, id uuid.UUID, input service.UpdateResumeInput) (*model.Resume, error) {
	args := m.Called(ctx, id, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Resume), args.Error(1)
}

func (m *MockResumeService) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockResumeService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*model.Resume, error) {
	args := m.Called(ctx, id, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Resume), args.Error(1)
}

func setupTestRouter(handler *ResumeHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func TestResumeHandler_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		input := service.CreateResumeInput{
			Name:    "John Doe",
			Email:   "john@example.com",
			Phone:   "1234567890",
			Source:  "LinkedIn",
			FileURL: "https://example.com/resume.pdf",
		}

		expectedResume := &model.Resume{
			ID:      uuid.New(),
			Name:    input.Name,
			Email:   input.Email,
			Phone:   input.Phone,
			Source:  input.Source,
			FileURL: input.FileURL,
			Status:  model.StatusPending,
		}

		mockSvc.On("Create", mock.Anything, input).Return(expectedResume, nil).Once()

		router := setupTestRouter(handler)
		router.POST("/resumes", handler.Create)

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("POST", "/resumes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, float64(0), resp["code"])
		assert.Equal(t, "success", resp["message"])

		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid input - missing name", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		input := map[string]string{
			"email": "john@example.com",
		}

		router := setupTestRouter(handler)
		router.POST("/resumes", handler.Create)

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("POST", "/resumes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid input - invalid email", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		input := map[string]string{
			"name":  "John Doe",
			"email": "invalid-email",
		}

		router := setupTestRouter(handler)
		router.POST("/resumes", handler.Create)

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("POST", "/resumes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		input := service.CreateResumeInput{
			Name:  "John Doe",
			Email: "john@example.com",
		}

		mockSvc.On("Create", mock.Anything, input).Return(nil, errors.New("database error")).Once()

		router := setupTestRouter(handler)
		router.POST("/resumes", handler.Create)

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("POST", "/resumes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockSvc.AssertExpectations(t)
	})
}

func TestResumeHandler_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		resumeID := uuid.New()
		expectedResume := &model.Resume{
			ID:     resumeID,
			Name:   "John Doe",
			Email:  "john@example.com",
			Status: model.StatusPending,
		}

		mockSvc.On("GetByID", mock.Anything, resumeID).Return(expectedResume, nil).Once()

		router := setupTestRouter(handler)
		router.GET("/resumes/:id", handler.GetByID)

		req := httptest.NewRequest("GET", "/resumes/"+resumeID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, float64(0), resp["code"])

		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid UUID", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		router := setupTestRouter(handler)
		router.GET("/resumes/:id", handler.GetByID)

		req := httptest.NewRequest("GET", "/resumes/invalid-uuid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		resumeID := uuid.New()
		mockSvc.On("GetByID", mock.Anything, resumeID).Return(nil, service.ErrResumeNotFound).Once()

		router := setupTestRouter(handler)
		router.GET("/resumes/:id", handler.GetByID)

		req := httptest.NewRequest("GET", "/resumes/"+resumeID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		mockSvc.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		resumeID := uuid.New()
		mockSvc.On("GetByID", mock.Anything, resumeID).Return(nil, errors.New("database error")).Once()

		router := setupTestRouter(handler)
		router.GET("/resumes/:id", handler.GetByID)

		req := httptest.NewRequest("GET", "/resumes/"+resumeID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockSvc.AssertExpectations(t)
	})
}

func TestResumeHandler_List(t *testing.T) {
	t.Run("success - default pagination", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		resumes := []model.Resume{
			{ID: uuid.New(), Name: "John Doe", Email: "john@example.com"},
			{ID: uuid.New(), Name: "Jane Smith", Email: "jane@example.com"},
		}

		mockSvc.On("List", mock.Anything, 1, 10, "", "").Return(resumes, int64(2), nil).Once()

		router := setupTestRouter(handler)
		router.GET("/resumes", handler.List)

		req := httptest.NewRequest("GET", "/resumes", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, float64(0), resp["code"])

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(2), data["total"])
		assert.Equal(t, float64(1), data["page"])
		assert.Equal(t, float64(10), data["page_size"])

		mockSvc.AssertExpectations(t)
	})

	t.Run("success - with pagination and filters", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		resumes := []model.Resume{
			{ID: uuid.New(), Name: "John Doe", Email: "john@example.com", Status: model.StatusParsed},
		}

		mockSvc.On("List", mock.Anything, 2, 20, model.StatusParsed, "LinkedIn").Return(resumes, int64(1), nil).Once()

		router := setupTestRouter(handler)
		router.GET("/resumes", handler.List)

		req := httptest.NewRequest("GET", "/resumes?page=2&page_size=20&status=parsed&source=LinkedIn", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid page param", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		// Should use default page value (1)
		mockSvc.On("List", mock.Anything, 1, 10, "", "").Return([]model.Resume{}, int64(0), nil).Once()

		router := setupTestRouter(handler)
		router.GET("/resumes", handler.List)

		req := httptest.NewRequest("GET", "/resumes?page=invalid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		mockSvc.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		mockSvc.On("List", mock.Anything, 1, 10, "", "").Return(nil, int64(0), errors.New("database error")).Once()

		router := setupTestRouter(handler)
		router.GET("/resumes", handler.List)

		req := httptest.NewRequest("GET", "/resumes", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockSvc.AssertExpectations(t)
	})
}

func TestResumeHandler_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		resumeID := uuid.New()
		input := service.UpdateResumeInput{
			Name:  "Updated Name",
			Email: "updated@example.com",
			Phone: "9876543210",
		}

		updatedResume := &model.Resume{
			ID:     resumeID,
			Name:   input.Name,
			Email:  input.Email,
			Phone:  input.Phone,
			Status: model.StatusPending,
		}

		mockSvc.On("Update", mock.Anything, resumeID, input).Return(updatedResume, nil).Once()

		router := setupTestRouter(handler)
		router.PUT("/resumes/:id", handler.Update)

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("PUT", "/resumes/"+resumeID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, float64(0), resp["code"])

		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid UUID", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		router := setupTestRouter(handler)
		router.PUT("/resumes/:id", handler.Update)

		input := service.UpdateResumeInput{Name: "Updated"}
		body, _ := json.Marshal(input)
		req := httptest.NewRequest("PUT", "/resumes/invalid-uuid", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		resumeID := uuid.New()
		input := service.UpdateResumeInput{Name: "Updated"}

		mockSvc.On("Update", mock.Anything, resumeID, input).Return(nil, service.ErrResumeNotFound).Once()

		router := setupTestRouter(handler)
		router.PUT("/resumes/:id", handler.Update)

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("PUT", "/resumes/"+resumeID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		mockSvc.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		resumeID := uuid.New()
		input := service.UpdateResumeInput{Name: "Updated"}

		mockSvc.On("Update", mock.Anything, resumeID, input).Return(nil, errors.New("database error")).Once()

		router := setupTestRouter(handler)
		router.PUT("/resumes/:id", handler.Update)

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("PUT", "/resumes/"+resumeID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockSvc.AssertExpectations(t)
	})
}

func TestResumeHandler_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		resumeID := uuid.New()
		mockSvc.On("Delete", mock.Anything, resumeID).Return(nil).Once()

		router := setupTestRouter(handler)
		router.DELETE("/resumes/:id", handler.Delete)

		req := httptest.NewRequest("DELETE", "/resumes/"+resumeID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, float64(0), resp["code"])
		assert.Equal(t, "resume deleted successfully", resp["message"])

		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid UUID", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		router := setupTestRouter(handler)
		router.DELETE("/resumes/:id", handler.Delete)

		req := httptest.NewRequest("DELETE", "/resumes/invalid-uuid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		resumeID := uuid.New()
		mockSvc.On("Delete", mock.Anything, resumeID).Return(service.ErrResumeNotFound).Once()

		router := setupTestRouter(handler)
		router.DELETE("/resumes/:id", handler.Delete)

		req := httptest.NewRequest("DELETE", "/resumes/"+resumeID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		mockSvc.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		resumeID := uuid.New()
		mockSvc.On("Delete", mock.Anything, resumeID).Return(errors.New("database error")).Once()

		router := setupTestRouter(handler)
		router.DELETE("/resumes/:id", handler.Delete)

		req := httptest.NewRequest("DELETE", "/resumes/"+resumeID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockSvc.AssertExpectations(t)
	})
}

func TestResumeHandler_UpdateStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		resumeID := uuid.New()
		input := map[string]string{"status": model.StatusProcessing}

		updatedResume := &model.Resume{
			ID:     resumeID,
			Name:   "John Doe",
			Email:  "john@example.com",
			Status: model.StatusProcessing,
		}

		mockSvc.On("UpdateStatus", mock.Anything, resumeID, model.StatusProcessing).Return(updatedResume, nil).Once()

		router := setupTestRouter(handler)
		router.PUT("/resumes/:id/status", handler.UpdateStatus)

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("PUT", "/resumes/"+resumeID.String()+"/status", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, float64(0), resp["code"])

		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid UUID", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		router := setupTestRouter(handler)
		router.PUT("/resumes/:id/status", handler.UpdateStatus)

		input := map[string]string{"status": "processing"}
		body, _ := json.Marshal(input)
		req := httptest.NewRequest("PUT", "/resumes/invalid-uuid/status", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing status in request", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		router := setupTestRouter(handler)
		router.PUT("/resumes/:id/status", handler.UpdateStatus)

		resumeID := uuid.New()
		input := map[string]string{}
		body, _ := json.Marshal(input)
		req := httptest.NewRequest("PUT", "/resumes/"+resumeID.String()+"/status", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		resumeID := uuid.New()
		input := map[string]string{"status": model.StatusProcessing}

		mockSvc.On("UpdateStatus", mock.Anything, resumeID, model.StatusProcessing).Return(nil, service.ErrResumeNotFound).Once()

		router := setupTestRouter(handler)
		router.PUT("/resumes/:id/status", handler.UpdateStatus)

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("PUT", "/resumes/"+resumeID.String()+"/status", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid status transition", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		resumeID := uuid.New()
		input := map[string]string{"status": model.StatusProcessing}

		mockSvc.On("UpdateStatus", mock.Anything, resumeID, model.StatusProcessing).Return(nil, service.ErrInvalidStatusTransition).Once()

		router := setupTestRouter(handler)
		router.PUT("/resumes/:id/status", handler.UpdateStatus)

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("PUT", "/resumes/"+resumeID.String()+"/status", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		mockSvc.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := new(MockResumeService)
		handler := NewResumeHandler(mockSvc)

		resumeID := uuid.New()
		input := map[string]string{"status": model.StatusProcessing}

		mockSvc.On("UpdateStatus", mock.Anything, resumeID, model.StatusProcessing).Return(nil, errors.New("database error")).Once()

		router := setupTestRouter(handler)
		router.PUT("/resumes/:id/status", handler.UpdateStatus)

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("PUT", "/resumes/"+resumeID.String()+"/status", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockSvc.AssertExpectations(t)
	})
}

func TestParseIntQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("valid int", func(t *testing.T) {
		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			result := parseIntQuery(c, "page", 10)
			c.JSON(http.StatusOK, result)
		})

		req := httptest.NewRequest("GET", "/test?page=5", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp int
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, 5, resp)
	})

	t.Run("invalid int - use default", func(t *testing.T) {
		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			result := parseIntQuery(c, "page", 10)
			c.JSON(http.StatusOK, result)
		})

		req := httptest.NewRequest("GET", "/test?page=invalid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp int
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, 10, resp)
	})

	t.Run("missing param - use default", func(t *testing.T) {
		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			result := parseIntQuery(c, "page", 10)
			c.JSON(http.StatusOK, result)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp int
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, 10, resp)
	})
}
