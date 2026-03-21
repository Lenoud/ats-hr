package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/example/ats-platform/internal/resume/service"
	"github.com/example/ats-platform/internal/shared/response"
)

// ResumeHandler handles HTTP requests for resumes
type ResumeHandler struct {
	svc service.ResumeService
}

// NewResumeHandler creates a new ResumeHandler instance
func NewResumeHandler(svc service.ResumeService) *ResumeHandler {
	return &ResumeHandler{
		svc: svc,
	}
}

// Create handles POST /resumes
func (h *ResumeHandler) Create(c *gin.Context) {
	var input service.CreateResumeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	resume, err := h.svc.Create(c.Request.Context(), input)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, resume)
}

// GetByID handles GET /resumes/:id
func (h *ResumeHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid resume id")
		return
	}

	resume, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrResumeNotFound {
			response.NotFound(c, "resume not found")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, resume)
}

// List handles GET /resumes with pagination and filtering
func (h *ResumeHandler) List(c *gin.Context) {
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "page_size", 10)
	status := c.Query("status")
	source := c.Query("source")

	resumes, total, err := h.svc.List(c.Request.Context(), page, pageSize, status, source)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessPage(c, resumes, total, page, pageSize)
}

// Update handles PUT /resumes/:id
func (h *ResumeHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid resume id")
		return
	}

	var input service.UpdateResumeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	resume, err := h.svc.Update(c.Request.Context(), id, input)
	if err != nil {
		if err == service.ErrResumeNotFound {
			response.NotFound(c, "resume not found")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, resume)
}

// Delete handles DELETE /resumes/:id
func (h *ResumeHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid resume id")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		if err == service.ErrResumeNotFound {
			response.NotFound(c, "resume not found")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "resume deleted successfully", nil)
}

// UpdateStatus handles PUT /resumes/:id/status
func (h *ResumeHandler) UpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid resume id")
		return
	}

	var input struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	resume, err := h.svc.UpdateStatus(c.Request.Context(), id, input.Status)
	if err != nil {
		if err == service.ErrResumeNotFound {
			response.NotFound(c, "resume not found")
			return
		}
		if err == service.ErrInvalidStatusTransition {
			response.BadRequest(c, "invalid status transition")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, resume)
}

// parseIntQuery parses an integer query parameter with a default value
func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	value := c.Query(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}
