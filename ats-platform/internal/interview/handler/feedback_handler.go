package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/example/ats-platform/internal/interview/service"
	"github.com/example/ats-platform/internal/shared/response"
)

// FeedbackHandler handles HTTP requests for interview feedback
type FeedbackHandler struct {
	feedbackSvc service.FeedbackService
}

// NewFeedbackHandler creates a new FeedbackHandler instance
func NewFeedbackHandler(feedbackSvc service.FeedbackService) *FeedbackHandler {
	return &FeedbackHandler{
		feedbackSvc: feedbackSvc,
	}
}

// Submit handles POST /api/v1/interviews/:id/feedback
func (h *FeedbackHandler) Submit(c *gin.Context) {
	interviewID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid interview id")
		return
	}

	var input service.SubmitFeedbackInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	feedback, err := h.feedbackSvc.Submit(c.Request.Context(), interviewID, input)
	if err != nil {
		if err == service.ErrInterviewNotFound {
			response.NotFound(c, "interview not found")
			return
		}
		if err == service.ErrFeedbackAlreadyExists {
			response.BadRequest(c, "feedback already exists for this interview")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "feedback submitted successfully", feedback)
}

// GetByInterviewID handles GET /api/v1/interviews/:id/feedback
func (h *FeedbackHandler) GetByInterviewID(c *gin.Context) {
	interviewID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid interview id")
		return
	}

	feedback, err := h.feedbackSvc.GetByInterviewID(c.Request.Context(), interviewID)
	if err != nil {
		if err == service.ErrFeedbackNotFound {
			response.NotFound(c, "feedback not found")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, feedback)
}
