package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/example/ats-platform/internal/resume/model"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	// Drop table if exists and create fresh
	_ = db.Exec(`DROP TABLE IF EXISTS resumes`)

	// Create table manually to avoid PostgreSQL-specific defaults
	err = db.Exec(`
		CREATE TABLE resumes (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT,
			phone TEXT,
			source TEXT,
			file_url TEXT,
			parsed_data TEXT,
			status TEXT DEFAULT 'pending',
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	return db
}

func createTestResume() *model.Resume {
	return &model.Resume{
		Name:   "John Doe",
		Email:  "john@example.com",
		Phone:  "+1234567890",
		Source: "LinkedIn",
		Status: model.StatusPending,
	}
}

func TestGormRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository(db)

	t.Run("should create resume successfully", func(t *testing.T) {
		resume := createTestResume()
		resume.ID = uuid.New()
		resume.ParsedData = map[string]any{"key": "value"}

		err := repo.Create(context.Background(), resume)

		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, resume.ID)
	})

	t.Run("should fail on duplicate ID", func(t *testing.T) {
		id := uuid.New()
		resume1 := createTestResume()
		resume1.ID = id

		resume2 := createTestResume()
		resume2.ID = id

		err := repo.Create(context.Background(), resume1)
		require.NoError(t, err)

		err = repo.Create(context.Background(), resume2)
		assert.Error(t, err)
	})
}

func TestGormRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository(db)

	t.Run("should get resume by ID", func(t *testing.T) {
		resume := createTestResume()
		resume.ID = uuid.New()
		resume.ParsedData = map[string]any{"key": "value"}
		err := repo.Create(context.Background(), resume)
		require.NoError(t, err)

		found, err := repo.GetByID(context.Background(), resume.ID)

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, resume.ID, found.ID)
		assert.Equal(t, resume.Name, found.Name)
		assert.Equal(t, resume.Email, found.Email)
	})

	t.Run("should return ErrNotFound when not found", func(t *testing.T) {
		_, err := repo.GetByID(context.Background(), uuid.New())

		assert.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
	})
}

func TestGormRepository_List(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository(db)

	// Create test data
	resumes := []*model.Resume{
		{ID: uuid.New(), Name: "Alice", Email: "alice@example.com", Source: "LinkedIn", Status: model.StatusPending},
		{ID: uuid.New(), Name: "Bob", Email: "bob@example.com", Source: "Indeed", Status: model.StatusParsed},
		{ID: uuid.New(), Name: "Charlie", Email: "charlie@example.com", Source: "LinkedIn", Status: model.StatusParsed},
		{ID: uuid.New(), Name: "Diana", Email: "diana@example.com", Source: "Direct", Status: model.StatusProcessing},
		{ID: uuid.New(), Name: "Eve", Email: "eve@example.com", Source: "Indeed", Status: model.StatusPending},
	}

	for _, r := range resumes {
		err := repo.Create(context.Background(), r)
		require.NoError(t, err)
	}

	t.Run("should list all resumes with pagination", func(t *testing.T) {
		filter := ListFilter{Page: 1, PageSize: 2}

		results, total, err := repo.List(context.Background(), filter)

		assert.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, int64(5), total)
	})

	t.Run("should filter by status", func(t *testing.T) {
		filter := ListFilter{Page: 1, PageSize: 10, Status: model.StatusParsed}

		results, total, err := repo.List(context.Background(), filter)

		assert.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, int64(2), total)
		for _, r := range results {
			assert.Equal(t, model.StatusParsed, r.Status)
		}
	})

	t.Run("should filter by source", func(t *testing.T) {
		filter := ListFilter{Page: 1, PageSize: 10, Source: "LinkedIn"}

		results, total, err := repo.List(context.Background(), filter)

		assert.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, int64(2), total)
		for _, r := range results {
			assert.Equal(t, "LinkedIn", r.Source)
		}
	})

	t.Run("should filter by status and source", func(t *testing.T) {
		filter := ListFilter{Page: 1, PageSize: 10, Status: model.StatusPending, Source: "Indeed"}

		results, total, err := repo.List(context.Background(), filter)

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "Indeed", results[0].Source)
		assert.Equal(t, model.StatusPending, results[0].Status)
	})

	t.Run("should handle empty result set", func(t *testing.T) {
		filter := ListFilter{Page: 1, PageSize: 10, Status: model.StatusArchived}

		results, total, err := repo.List(context.Background(), filter)

		assert.NoError(t, err)
		assert.Len(t, results, 0)
		assert.Equal(t, int64(0), total)
	})

	t.Run("should handle pagination offset", func(t *testing.T) {
		filter := ListFilter{Page: 2, PageSize: 2}

		results, total, err := repo.List(context.Background(), filter)

		assert.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, int64(5), total)
	})
}

func TestGormRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository(db)

	t.Run("should update resume successfully", func(t *testing.T) {
		resume := createTestResume()
		resume.ID = uuid.New()
		err := repo.Create(context.Background(), resume)
		require.NoError(t, err)

		resume.Name = "Jane Doe"
		resume.Email = "jane@example.com"
		resume.Status = model.StatusParsed

		err = repo.Update(context.Background(), resume)

		assert.NoError(t, err)

		updated, err := repo.GetByID(context.Background(), resume.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Jane Doe", updated.Name)
		assert.Equal(t, "jane@example.com", updated.Email)
		assert.Equal(t, model.StatusParsed, updated.Status)
	})
}

func TestGormRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository(db)

	t.Run("should soft delete resume", func(t *testing.T) {
		resume := createTestResume()
		resume.ID = uuid.New()
		err := repo.Create(context.Background(), resume)
		require.NoError(t, err)

		err = repo.Delete(context.Background(), resume.ID)
		assert.NoError(t, err)

		_, err = repo.GetByID(context.Background(), resume.ID)
		assert.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("should return error when deleting non-existent resume", func(t *testing.T) {
		err := repo.Delete(context.Background(), uuid.New())

		assert.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
	})
}

func TestGormRepository_UpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository(db)

	t.Run("should update status successfully", func(t *testing.T) {
		resume := createTestResume()
		resume.ID = uuid.New()
		resume.Status = model.StatusPending
		err := repo.Create(context.Background(), resume)
		require.NoError(t, err)

		err = repo.UpdateStatus(context.Background(), resume.ID, model.StatusProcessing)

		assert.NoError(t, err)

		updated, err := repo.GetByID(context.Background(), resume.ID)
		assert.NoError(t, err)
		assert.Equal(t, model.StatusProcessing, updated.Status)
	})

	t.Run("should return error when updating status of non-existent resume", func(t *testing.T) {
		err := repo.UpdateStatus(context.Background(), uuid.New(), model.StatusProcessing)

		assert.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
	})
}

func TestListFilter_DefaultValues(t *testing.T) {
	t.Run("should apply default page values", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewGormRepository(db)

		resume := createTestResume()
		resume.ID = uuid.New()
		err := repo.Create(context.Background(), resume)
		require.NoError(t, err)

		filter := ListFilter{Page: 0, PageSize: 0}
		results, total, err := repo.List(context.Background(), filter)

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, int64(1), total)
	})
}
