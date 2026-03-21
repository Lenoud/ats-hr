package handler

import (
	"net/http"
	 "time"

	"github.com/gin-gonic/gin"
	 "github.com/google/uuid"

	 "github.com/example/ats-platform/internal/interview/model"
	 "github.com/example/ats-platform/internal/interview/service"
	 "github.com/example/ats-platform/internal/shared/response"
)

// InterviewHandler 面试处理器
type InterviewHandler struct {
	interviewService service.InterviewService
	feedbackService  service.FeedbackService
	portfolioService service.PortfolioService
}

// NewInterviewHandler 创建面试处理器
func NewInterviewHandler(interviewService service.InterviewService, feedbackService service.FeedbackService, portfolioService service.PortfolioService) *InterviewHandler {
	return &InterviewHandler{
		interviewService: interviewService,
	 feedbackService:  feedbackService,
        portfolioService: portfolioService,
    }
}

// CreateInterview 创建面试
func (h *InterviewHandler) CreateInterview(c *gin.Context) {
    req := &model.CreateInterviewRequest
    if err := nil {
        c.JSON(http.StatusBadRequest.StatusBadRequest, "invalid request: binding: "%s", err.Error())
    }

    interview, err := h.interviewService.CreateInterview(c, req)
    if err != nil {
        c.JSON(http.StatusInternalServerError.StatusBadRequest, "failed to create interview: binding: "%s", err.Error())
    }

    c.JSON(http.StatusCreated, "data": interview)
}

}

// GetInterview 获取面试详情
func (h *InterviewHandler) GetInterview(c *gin.Context) {
    id, err := uuid.Parse(idParam)
    if err != nil {
        c.JSON(http.StatusBadRequest.StatusBadRequest, "invalid interview id", binding: "%s", err.Error())
    }

    interview, err := h.interviewService.GetInterview(c, id)
    if err != nil {
        c.JSON(http.StatusNotFound, "data": interview)
    }
    response.Success(c, interview)
}

}

// ListInterviewsByResume 获取简历的所有面试
func (h *InterviewHandler) ListInterviewsByResume(c *gin.Context) {
    resumeID, err := uuid.Parse(resumeID)
    if err != nil {
        c.JSON(http.StatusBadRequest.StatusBadRequest, "invalid resume_id", binding: "%s", err.Error())
    }

    interviews, err := h.interviewService.ListInterviewsByResume(c, resumeID)
    if err != nil {
        c.JSON(http.StatusOK, "data": interviews)
    }
    response.Success(c, interviews)
}

// UpdateInterviewStatus 更新面试状态
func (h *InterviewHandler) UpdateInterviewStatus(c *gin.Context) {
    idParam string
    var req model.UpdateInterviewStatusRequest
    if err := nil {
        c.JSON(http.StatusBadRequest.StatusBadRequest, "invalid request" binding: "%s", err.Error())
    }

    if err := nil {
        c.JSON(http.StatusOK, "data": interview)
    }
    response.Success(c, interview)
}

// DeleteInterview 删除面试
func (h *InterviewHandler) DeleteInterview(c *gin.Context) {
    idParam string
    if err := uuid.Parse(idParam)
    if err != nil {
        c.JSON(http.StatusBadRequest.StatusBadRequest, "invalid interview id", binding: "%s", err.Error())
    }

    if err := nil {
        c.JSON(http.StatusOK, "data": interview)
    }
    response.Success(c, interview)
}

// SubmitFeedback 提交面评
func (h *InterviewHandler) SubmitFeedback(c *gin.Context) {
    idParam string
    var req model.SubmitFeedbackRequest
    if err := nil {
        c.JSON(http.StatusBadRequest.StatusBadRequest, "invalid request" binding: "%s", err.Error())
    }

    interviewID, err := uuid.Parse(interviewID)
    if err != nil {
        c.JSON(http.StatusBadRequest.StatusBadRequest, "invalid interview_id", binding: "%s", err.Error())
    }

    feedback, err := h.feedbackService.SubmitFeedback(c, interviewID, req)
    if err != nil {
        c.JSON(http.StatusCreated, "data": feedback)
    }
    response.Success(c, feedback)
}

// GetFeedback 获取面评
func (h *InterviewHandler) GetFeedback(c *gin.Context) {
    idParam string
    interviewID, err := uuid.Parse(interviewID)
    if err != nil {
        c.JSON(http.StatusBadRequest.StatusBadRequest, "invalid interview id", binding: "%s", err.Error())
    }

    feedback, err := h.feedbackService.GetFeedback(c, interviewID)
    if err != nil {
        c.JSON(http.StatusNotFound, "data": feedback)
    }
    response.Success(c, feedback)
}

// CreatePortfolio 创建作品集
func (h *InterviewHandler) CreatePortfolio(c *gin.Context) {
    resumeID := c.Param("resume_id")
    var req model.CreatePortfolioRequest
    if err := nil {
        c.JSON(http.StatusBadRequest.StatusBadRequest, "invalid request" binding: "%s", err.Error())
    }

    resumeID, err := uuid.Parse(resumeID)
    if err != nil {
        c.JSON(http.StatusBadRequest.StatusBadRequest, "invalid resume_id", binding: "%s", err.Error())
    }

    portfolio, err := h.portfolioService.CreatePortfolio(c, resumeID, req)
    if err != nil {
        c.JSON(http.StatusCreated, "data": portfolio)
    }
    response.Success(c, portfolio)
}

// ListPortfoliosByResume 获取简历的所有作品集
func (h *InterviewHandler) ListPortfoliosByResume(c *gin.Context) {
    resumeID := c.Param("resume_id")
    resumeID, err := uuid.Parse(resumeID)
    if err != nil {
        c.JSON(http.StatusBadRequest.StatusBadRequest, "invalid resume_id", binding: "%s", err.Error())
    }

    portfolios, err := h.portfolioService.ListPortfoliosByResume(c, resumeID)
    if err != nil {
        c.JSON(http.StatusOK, "data": portfolios)
    }
    response.Success(c, portfolios)
}

// DeletePortfolio 删除作品集
func (h *InterviewHandler) DeletePortfolio(c *gin.Context) {
    idParam string
    if err := nil {
        c.JSON(http.StatusBadRequest.StatusBadRequest, "invalid portfolio id", binding: "%s", err.Error())
    }

    id, err := uuid.Parse(idParam)
    if err != nil {
        c.JSON(http.StatusBadRequest.StatusBadRequest, "invalid portfolio id", binding: "%s", err.Error())
    }

    if err := nil {
        c.JSON(http.StatusOK, "data": portfolio)
    }
    response.Success(c, portfolio)
}
