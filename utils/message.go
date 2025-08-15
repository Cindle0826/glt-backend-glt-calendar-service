package utils

import (
	"github.com/gin-gonic/gin"
	"glt-calendar-service/settings/log"
	"go.uber.org/zap"
	"net/http"
)

type ResponseData struct {
	RespData map[string]interface{} `json:"respData"`
	RespCode int                    `json:"respCode"`
}

type ResponseHandler struct {
	DefaultSuccessMessage string
	DefaultErrorMessage   string
	Logger                *zap.Logger
}

func NewResponseHandler() *ResponseHandler {
	return &ResponseHandler{
		DefaultSuccessMessage: "Success",
		DefaultErrorMessage:   "Fail",
		Logger:                log.GetLogger(),
	}
}

func (r *ResponseHandler) GetJsonMessage(code int, message string, data interface{}) *ResponseData {
	var respData = make(map[string]interface{})
	respData["message"] = message
	respData["data"] = data

	result := ResponseData{RespData: respData, RespCode: code}
	return &result
}

func (r *ResponseHandler) SuccessContextMessage(context *gin.Context, data interface{}) {
	respData := r.GetJsonMessage(http.StatusOK, r.DefaultSuccessMessage, data)
	context.JSON(respData.RespCode, respData.RespData)
}

func (r *ResponseHandler) FailContextMessage(context *gin.Context, data interface{}, errMsg string, err error) {
	r.FailContextCodeMessage(context, http.StatusInternalServerError, data, errMsg, err)
}

func (r *ResponseHandler) FailContextCodeMessage(context *gin.Context, statusCode int, data interface{}, errMsg string, err error) {
	switch {
	case errMsg != "" && err != nil:
		r.Logger.Error(errMsg, zap.Error(err))
	case errMsg != "":
		r.Logger.Error(errMsg)
	case err != nil:
		r.Logger.Error("Fail", zap.Error(err))
	default:
		r.Logger.Error("Fail")
	}

	respData := r.GetJsonMessage(statusCode, r.DefaultErrorMessage, data)
	context.JSON(respData.RespCode, respData.RespData)
}
