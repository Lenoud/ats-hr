package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/example/ats-platform/internal/search/repository"
	"github.com/example/ats-platform/internal/search/service"
	"github.com/example/ats-platform/internal/shared/response"
)

// SearchHandler handles HTTP requests for search operations
type SearchHandler struct {
	svc service.SearchService
}

// NewSearchHandler creates a new SearchHandler instance
func NewSearchHandler(svc service.SearchService) *SearchHandler {
	return &SearchHandler{
		svc: svc,
	}
}

// Search handles GET /api/v1/search
func (h *SearchHandler) Search(c *gin.Context) {
	filter := repository.SearchFilter{
		Query:         c.Query("query"),
		Status:        c.Query("status"),
		Source:        c.Query("source"),
		Page:          parseIntQuery(c, "page", 1),
		PageSize:      parseIntQuery(c, "page_size", 10),
		MinExperience: parseIntQuery(c, "min_exp", 0),
		MaxExperience: parseIntQuery(c, "max_exp", 0),
	}

	// Parse skills (comma-separated)
	if skills := c.Query("skills"); skills != "" {
		filter.Skills = strings.Split(skills, ",")
	}

	result, err := h.svc.Search(c.Request.Context(), filter)
	if err != nil {
		response.InternalError(c, "search failed")
		return
	}

	response.SuccessPage(c, result.Documents, result.Total, filter.Page, filter.PageSize)
}

// AdvancedSearch handles POST /api/v1/search/advanced
func (h *SearchHandler) AdvancedSearch(c *gin.Context) {
	var filter repository.SearchFilter
	if err := c.ShouldBindJSON(&filter); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.Search(c.Request.Context(), filter)
	if err != nil {
		response.InternalError(c, "search failed")
		return
	}

	response.SuccessPage(c, result.Documents, result.Total, filter.Page, filter.PageSize)
}

// parseIntQuery parses an integer query parameter with a default value
func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	val := c.Query(key)
	if val == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return intValue
}
