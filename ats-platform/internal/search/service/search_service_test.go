package service

import (
	"context"
	"testing"

    "github.com/example/ats-platform/internal/search/model"
    "github.com/example/ats-platform/internal/search/repository"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// MockESRepository is a mock implementation of ESRepository for testing
type MockESRepository struct {
    mock.Mock
}

func (m *MockESRepository) Index(ctx context.Context, doc *model.ResumeDocument) error {
    args := m.Called(ctx, doc)
    return args.Error(0)
}

func (m *MockESRepository) GetByID(ctx context.Context, id string) (*model.ResumeDocument, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*model.ResumeDocument), args.Error(1)
}

func (m *MockESRepository) Delete(ctx context.Context, id string) error {
    args := m.Called(ctx, id)
    return args.Error(0)
}

func (m *MockESRepository) Search(ctx context.Context, filter repository.SearchFilter) (*repository.SearchResult, error) {
    args := m.Called(ctx, filter)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*repository.SearchResult), args.Error(1)
}

func (m *MockESRepository) UpdateStatus(ctx context.Context, id string, status string) error {
    args := m.Called(ctx, id, status)
    return args.Error(0)
}

func TestSearchService_Search(t *testing.T) {
    mockRepo := new(MockESRepository)
    svc := NewSearchService(mockRepo)

    expected := &repository.SearchResult{
        Documents: []model.ResumeDocument{{ResumeID: "1", Name: "John"}},
        Total:     1,
    }

    mockRepo.On("Search", mock.Anything, mock.Anything).Return(expected, nil)

    result, err := svc.Search(context.Background(), repository.SearchFilter{Query: "John"})
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
    mockRepo.AssertExpectations(t)
}

func TestSearchService_Search_WithDefaults(t *testing.T) {
    mockRepo := new(MockESRepository)
    svc := NewSearchService(mockRepo)

    expected := &repository.SearchResult{
        Documents: []model.ResumeDocument{{ResumeID: "1", Name: "John"}},
        Total:     1,
    }

    mockRepo.On("Search", mock.Anything, repository.SearchFilter{
        Page:    2,
        PageSize: 10,
    }).Return(expected, nil)

    result, err := svc.Search(context.Background(), repository.SearchFilter{Page: 2, PageSize: 20})
    assert.NoError(t, err)
    assert.Equal(t, int(filter.Page), 2)
    assert.Equal(t, filter.PageSize, 10)
    mockRepo.AssertExpectations(t)
}

func TestSearchService_Search_PageSizeTooLarge(t *testing.T) {
    mockRepo := new(MockESRepository)
    svc := NewSearchService(mockRepo)

    expected := &repository.SearchResult{
        Documents: []model.ResumeDocument{},
        Total:     0,
    }
    mockRepo.On("Search", mock.Anything, repository.SearchFilter{
        Page:    1,
        PageSize: 150, // > 100
    }).Return(expected, nil)

    result, err := svc.Search(context.Background(), repository.SearchFilter{Page: 1, PageSize: 150})
    assert.NoError(t, err)
    assert.Equal(t, filter.PageSize, 10, // Should be capped at 10
    mockRepo.AssertExpectations(t)
}

func TestSearchService_IndexResume(t *testing.T) {
    mockRepo := new(MockESRepository)
    svc := NewSearchService(mockRepo)

    doc := &model.ResumeDocument{ResumeID: "1", Name: "John"}
    mockRepo.On("Index", mock.Anything, doc).Return(nil)

    err := svc.IndexResume(context.Background(), doc)
    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
}

func TestSearchService_DeleteResume(t *testing.T) {
    mockRepo := new(MockESRepository)
    svc := NewSearchService(mockRepo)
    mockRepo.On("Delete", mock.Anything, "1").Return(nil)
    err := svc.DeleteResume(context.Background(), "1")
    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
}
func TestSearchService_UpdateResumeStatus(t *testing.T) {
    mockRepo := new(MockESRepository)
    svc := NewSearchService(mockRepo)
    mockRepo.On("UpdateStatus", mock.Anything, "1", "parsed").Return(nil)
    err := svc.UpdateResumeStatus(context.Background(), "1", "parsed")
    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
}
