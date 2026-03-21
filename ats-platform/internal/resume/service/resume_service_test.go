package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/example/ats-platform/internal/resume/model"
	"github.com/example/ats-platform/internal/resume/repository"
)

// MockRepository is a mock implementation of ResumeRepository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, resume *model.Resume) error {
	args := m.Called(ctx, resume)
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Resume, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Resume), args.Error(1)
}

func (m *MockRepository) List(ctx context.Context, filter repository.ListFilter) ([]model.Resume, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]model.Resume), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) Update(ctx context.Context, resume *model.Resume) error {
	args := m.Called(ctx, resume)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

// helper function to create a test resume
func createTestResume(id uuid.UUID) *model.Resume {
	return &model.Resume{
		ID:      id,
		Name:    "John Doe",
		Email:   "john@example.com",
		Phone:   "+1234567890",
		Source:  "LinkedIn",
		FileURL: "https://example.com/resume.pdf",
		Status:  model.StatusPending,
	}
}

func TestResumeService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully create resume", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewResumeService(mockRepo)

		input := CreateResumeInput{
			Name:    "John Doe",
			Email:   "john@example.com",
			Phone:   "+1234567890",
			Source:  "LinkedIn",
			FileURL: "https://example.com/resume.pdf",
		}

		mockRepo.On("Create", ctx, mock.AnythingOfType("*model.Resume")).Return(nil).Run(func(args mock.Arguments) {
			resume := args.Get(1).(*model.Resume)
			resume.ID = uuid.New() // Simulate ID generation
		})

		result, err := service.Create(ctx, input)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, input.Name, result.Name)
		assert.Equal(t, input.Email, result.Email)
		assert.Equal(t, input.Phone, result.Phone)
		assert.Equal(t, input.Source, result.Source)
		assert.Equal(t, input.FileURL, result.FileURL)
		assert.Equal(t, model.StatusPending, result.Status)
		assert.NotEqual(t, uuid.Nil, result.ID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error on create", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewResumeService(mockRepo)

		input := CreateResumeInput{
			Name:  "John Doe",
			Email: "john@example.com",
		}

		expectedErr := errors.New("database error")
		mockRepo.On("Create", ctx, mock.AnythingOfType("*model.Resume")).Return(expectedErr)

		result, err := service.Create(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestResumeService_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully get resume by id", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewResumeService(mockRepo)

		id := uuid.New()
		expectedResume := createTestResume(id)

		mockRepo.On("GetByID", ctx, id).Return(expectedResume, nil)

		result, err := service.GetByID(ctx, id)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, expectedResume.ID, result.ID)
		assert.Equal(t, expectedResume.Name, result.Name)
		assert.Equal(t, expectedResume.Email, result.Email)
		mockRepo.AssertExpectations(t)
	})

	t.Run("resume not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewResumeService(mockRepo)

		id := uuid.New()
		mockRepo.On("GetByID", ctx, id).Return(nil, repository.ErrNotFound)

		result, err := service.GetByID(ctx, id)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrResumeNotFound, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewResumeService(mockRepo)

		id := uuid.New()
		expectedErr := errors.New("database error")
		mockRepo.On("GetByID", ctx, id).Return(nil, expectedErr)

		result, err := service.GetByID(ctx, id)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestResumeService_List(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully list resumes", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewResumeService(mockRepo)

		id1 := uuid.New()
		id2 := uuid.New()
		expectedResumes := []model.Resume{*createTestResume(id1), *createTestResume(id2)}
		total := int64(2)

		mockRepo.On("List", ctx, mock.MatchedBy(func(filter repository.ListFilter) bool {
			return filter.Page == 1 && filter.PageSize == 10 &&
				filter.Status == "" && filter.Source == ""
		})).Return(expectedResumes, total, nil)

		result, count, err := service.List(ctx, 1, 10, "", "")

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, total, count)
		mockRepo.AssertExpectations(t)
	})

	t.Run("list with filters", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewResumeService(mockRepo)

		expectedResumes := []model.Resume{}
		total := int64(0)

		mockRepo.On("List", ctx, mock.MatchedBy(func(filter repository.ListFilter) bool {
			return filter.Page == 2 && filter.PageSize == 20 &&
				filter.Status == model.StatusParsed && filter.Source == "LinkedIn"
		})).Return(expectedResumes, total, nil)

		result, count, err := service.List(ctx, 2, 20, model.StatusParsed, "LinkedIn")

		require.NoError(t, err)
		assert.Empty(t, result)
		assert.Equal(t, total, count)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewResumeService(mockRepo)

		expectedErr := errors.New("database error")
		mockRepo.On("List", ctx, mock.AnythingOfType("repository.ListFilter")).Return(nil, int64(0), expectedErr)

		result, count, err := service.List(ctx, 1, 10, "", "")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, int64(0), count)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestResumeService_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully update resume", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewResumeService(mockRepo)

		id := uuid.New()
		existingResume := createTestResume(id)

		input := UpdateResumeInput{
			Name:  "Jane Doe",
			Email: "jane@example.com",
		}

		mockRepo.On("GetByID", ctx, id).Return(existingResume, nil)
		mockRepo.On("Update", ctx, mock.MatchedBy(func(r *model.Resume) bool {
			return r.Name == input.Name && r.Email == input.Email
		})).Return(nil)

		result, err := service.Update(ctx, id, input)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, input.Name, result.Name)
		assert.Equal(t, input.Email, result.Email)
		assert.Equal(t, existingResume.Phone, result.Phone) // Unchanged
		mockRepo.AssertExpectations(t)
	})

	t.Run("update resume not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewResumeService(mockRepo)

		id := uuid.New()
		input := UpdateResumeInput{
			Name: "Jane Doe",
		}

		mockRepo.On("GetByID", ctx, id).Return(nil, repository.ErrNotFound)

		result, err := service.Update(ctx, id, input)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrResumeNotFound, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("update with empty input", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewResumeService(mockRepo)

		id := uuid.New()
		existingResume := createTestResume(id)

		input := UpdateResumeInput{}

		mockRepo.On("GetByID", ctx, id).Return(existingResume, nil)
		mockRepo.On("Update", ctx, mock.MatchedBy(func(r *model.Resume) bool {
			return r.Name == existingResume.Name && r.Email == existingResume.Email
		})).Return(nil)

		result, err := service.Update(ctx, id, input)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, existingResume.Name, result.Name)
		assert.Equal(t, existingResume.Email, result.Email)
		mockRepo.AssertExpectations(t)
	})
}

func TestResumeService_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully delete resume", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewResumeService(mockRepo)

		id := uuid.New()
		mockRepo.On("Delete", ctx, id).Return(nil)

		err := service.Delete(ctx, id)

		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("delete resume not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewResumeService(mockRepo)

		id := uuid.New()
		mockRepo.On("Delete", ctx, id).Return(repository.ErrNotFound)

		err := service.Delete(ctx, id)

		assert.Error(t, err)
		assert.Equal(t, ErrResumeNotFound, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error on delete", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewResumeService(mockRepo)

		id := uuid.New()
		expectedErr := errors.New("database error")
		mockRepo.On("Delete", ctx, id).Return(expectedErr)

		err := service.Delete(ctx, id)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestResumeService_UpdateStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully update status with valid transition", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewResumeService(mockRepo)

		id := uuid.New()
		existingResume := createTestResume(id)
		existingResume.Status = model.StatusPending

		mockRepo.On("GetByID", ctx, id).Return(existingResume, nil)
		mockRepo.On("UpdateStatus", ctx, id, model.StatusProcessing).Return(nil)

		result, err := service.UpdateStatus(ctx, id, model.StatusProcessing)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, model.StatusProcessing, result.Status)
		mockRepo.AssertExpectations(t)
	})

	t.Run("update status resume not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewResumeService(mockRepo)

		id := uuid.New()
		mockRepo.On("GetByID", ctx, id).Return(nil, repository.ErrNotFound)

		result, err := service.UpdateStatus(ctx, id, model.StatusProcessing)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrResumeNotFound, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("update status with invalid transition", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewResumeService(mockRepo)

		id := uuid.New()
		existingResume := createTestResume(id)
		existingResume.Status = model.StatusArchived // No transitions allowed

		mockRepo.On("GetByID", ctx, id).Return(existingResume, nil)

		result, err := service.UpdateStatus(ctx, id, model.StatusPending)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrInvalidStatusTransition, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("update status from parsed to archived", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewResumeService(mockRepo)

		id := uuid.New()
		existingResume := createTestResume(id)
		existingResume.Status = model.StatusParsed

		mockRepo.On("GetByID", ctx, id).Return(existingResume, nil)
		mockRepo.On("UpdateStatus", ctx, id, model.StatusArchived).Return(nil)

		result, err := service.UpdateStatus(ctx, id, model.StatusArchived)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, model.StatusArchived, result.Status)
		mockRepo.AssertExpectations(t)
	})

	t.Run("update status from processing to parsed", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewResumeService(mockRepo)

		id := uuid.New()
		existingResume := createTestResume(id)
		existingResume.Status = model.StatusProcessing

		mockRepo.On("GetByID", ctx, id).Return(existingResume, nil)
		mockRepo.On("UpdateStatus", ctx, id, model.StatusParsed).Return(nil)

		result, err := service.UpdateStatus(ctx, id, model.StatusParsed)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, model.StatusParsed, result.Status)
		mockRepo.AssertExpectations(t)
	})

	t.Run("update status repository error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewResumeService(mockRepo)

		id := uuid.New()
		existingResume := createTestResume(id)
		existingResume.Status = model.StatusPending

		expectedErr := errors.New("database error")
		mockRepo.On("GetByID", ctx, id).Return(existingResume, nil)
		mockRepo.On("UpdateStatus", ctx, id, model.StatusProcessing).Return(expectedErr)

		result, err := service.UpdateStatus(ctx, id, model.StatusProcessing)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})
}
