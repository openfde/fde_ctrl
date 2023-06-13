package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type InfraResponse struct {
	Code    int
	Message string
	Data    interface{}
}

func ResponseParamterError(c *gin.Context, err error) {
	parameterResponse := InfraResponse{
		Code:    400,
		Message: err.Error(),
	}
	c.JSON(http.StatusBadRequest, parameterResponse)
}

func Response(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, InfraResponse{
		Code:    200,
		Message: "success",
		Data:    data,
	})
}

func ResponseError(c *gin.Context, statusCode int, err error) {
	parameterResponse := InfraResponse{
		Code:    statusCode,
		Message: err.Error(),
	}
	c.JSON(statusCode, parameterResponse)
}

func ResponseWithPagination(c *gin.Context, page PageQuery, data interface{}) {
	c.JSON(http.StatusOK, Infra{
		Code:    200,
		Message: "success",
		Data: DataWithPagenation{
			Data: data,
			Page: page,
		},
	})
}
