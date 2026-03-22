package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/example/ats-platform/internal/interview/service"
	"github.com/example/ats-platform/internal/shared/response"
)

// PortfolioHandler handles HTTP requests for portfolios
type PortfolioHandler struct {
	portfolioSvc service.PortfolioService
}

// NewPortfolioHandler creates a new PortfolioHandler instance
func NewPortfolioHandler(portfolioSvc service.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{
		portfolioSvc: portfolioSvc,
	}
}

// Create handles POST /api/v1/resumes/:id/portfolios
func (h *PortfolioHandler) Create(c *gin.Context) {
	resumeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid resume id")
		return
	}

	var input service.CreatePortfolioInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	portfolio, err := h.portfolioSvc.Create(c.Request.Context(), resumeID, input)
	if err != nil {
		if err == service.ErrInvalidFileType {
			response.BadRequest(c, "invalid file type, only pdf, link, image are allowed")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "portfolio created successfully", portfolio)
}

// ListByResumeID handles GET /api/v1/resumes/:id/portfolios
func (h *PortfolioHandler) ListByResumeID(c *gin.Context) {
	resumeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid resume id")
		return
	}

	portfolios, err := h.portfolioSvc.ListByResumeID(c.Request.Context(), resumeID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, portfolios)
}

// Delete handles DELETE /api/v1/portfolios/:id
func (h *PortfolioHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid portfolio id")
		return
	}

	if err := h.portfolioSvc.Delete(c.Request.Context(), id); err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "portfolio deleted successfully", nil)
}
