package handlers

import (
	"net/http"
	"strconv"

	"socks5-dpi-proxy/internal/storage"

	"github.com/gin-gonic/gin"
)

type RequestHandler struct {
	storage *storage.Storage
}

func NewRequestHandler(storage *storage.Storage) *RequestHandler {
	return &RequestHandler{
		storage: storage,
	}
}

func (h *RequestHandler) GetRequests(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}
	
	if limit > 1000 {
		limit = 1000 // Cap at 1000 requests
	}
	
	requests, err := h.storage.GetRequests(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"requests": requests,
		"count":    len(requests),
	})
}

func (h *RequestHandler) GetRequestDetails(c *gin.Context) {
	requestID := c.Param("id")
	
	request, err := h.storage.GetRequestByID(requestID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}
	
	// Get DPI events for this request
	events, err := h.storage.GetDPIEvents(requestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"request": request,
		"events":  events,
	})
}

func (h *RequestHandler) GetDomainStatistics(c *gin.Context) {
	domain := c.Param("domain")
	hoursStr := c.DefaultQuery("hours", "24")
	
	hours, err := strconv.Atoi(hoursStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid hours parameter"})
		return
	}
	
	if hours > 24 {
		hours = 24 // Limit to 24 hours
	}
	
	stats, err := h.storage.GetDomainStatistics(domain, hours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, stats)
}
