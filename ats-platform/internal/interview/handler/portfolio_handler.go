package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "github.com/example/ats-platform/internal/interview/model"
    "github.com/example/ats-platform/internal/interview/service"
)

// PortfolioHandler 作品集处理器
type PortfolioHandler struct {
    portfolioService service.PortfolioService
}

// NewPortfolioHandler 创建作品集处理器
func NewPortfolioHandler(portfolioService service.PortfolioService) *PortfolioHandler {
    return &PortfolioHandler{portfolioService: portfolioService}
}

// CreatePortfolio 创建作品集
func (h *PortfolioHandler) CreatePortfolio(c *gin.Context) {
    resumeID, err := uuid.Parse(c.Param("resume_id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resume_id"})
        return
    }

    var req model.CreatePortfolioRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    portfolio, err := h.portfolioService.CreatePortfolio(c, resumeID, req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusCreated, gin.H{"data": portfolio})
}
