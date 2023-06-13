package controller

import (
	"fde_ctrl/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

func getPageQuery(c *gin.Context) response.PageQuery {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil || pageSize < 1 {
		pageSize = 10
	} else if pageSize > 200 {
		pageSize = 200
	}
	sortDirectrion := c.DefaultQuery("sort_directrion", "desc")
	pageStatus, err := strconv.ParseBool(c.DefaultQuery("page_status", "true"))
	return response.PageQuery{
		PageEnable:    pageStatus,
		Page:          page,
		PageSize:      pageSize,
		SortDirection: sortDirectrion,
	}
}
