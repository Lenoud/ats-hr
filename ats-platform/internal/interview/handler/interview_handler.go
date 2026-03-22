package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/example/ats-platform/internal/interview/service"
	"github.com/example/ats-platform/internal/shared/response"
)

// InterviewHandler handles HTTP requests for interviews
type InterviewHandler struct {
	interviewSvc service.InterviewService
}

// NewInterviewHandler creates a new InterviewHandler instance
func NewInterviewHandler(interviewSvc service.InterviewService) *InterviewHandler {
	return &InterviewHandler{
		interviewSvc: interviewSvc,
	}
}

// Create handles POST /api/v1/interviews
func (h *InterviewHandler) Create(c *gin.Context) {
	var input service.CreateInterviewInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	interview, err := h.interviewSvc.Create(c.Request.Context(), input)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "interview created successfully", interview)
}

// GetByID handles GET /api/v1/interviews/:id
func (h *InterviewHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid interview id")
		return
	}

	interview, err := h.interviewSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrInterviewNotFound {
			response.NotFound(c, "interview not found")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, interview)
}

// ListByResumeID handles GET /api/v1/resumes/:id/interviews
func (h *InterviewHandler) ListByResumeID(c *gin.Context) {
	resumeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid resume id")
		return
	}

	interviews, err := h.interviewSvc.ListByResumeID(c.Request.Context(), resumeID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, interviews)
}

// UpdateStatus handles PUT /api/v1/interviews/:id/status
func (h *InterviewHandler) UpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid interview id")
		return
	}

	var input service.UpdateInterviewStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	interview, err := h.interviewSvc.UpdateStatus(c.Request.Context(), id, input)
	if err != nil {
		if err == service.ErrInterviewNotFound {
			response.NotFound(c, "interview not found")
			return
		}
		if err == service.ErrInvalidStatusTransition {
			response.BadRequest(c, "invalid status transition")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, interview)
}

// Delete handles DELETE /api/v1/interviews/:id
func (h *InterviewHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid interview id")
		return
	}

	if err := h.interviewSvc.Delete(c.Request.Context(), id); err != nil {
		if err == service.ErrInterviewNotFound {
			response.NotFound(c, "interview not found")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "interview deleted successfully", nil)
}
