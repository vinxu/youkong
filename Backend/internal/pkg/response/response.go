package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 错误码定义
const (
	CodeSuccess       = 0
	CodeParamError    = 1001
	CodeUnauthorized  = 1002
	CodeTokenExpired  = 1003
	CodeNotFound      = 1004
	CodeForbidden     = 1005
	CodeInternalError = 5000
)

// 错误消息
var codeMessages = map[int]string{
	CodeSuccess:       "成功",
	CodeParamError:    "参数错误",
	CodeUnauthorized:  "未授权",
	CodeTokenExpired:  "Token已过期",
	CodeNotFound:      "资源不存在",
	CodeForbidden:     "无权限",
	CodeInternalError: "服务器内部错误",
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: codeMessages[CodeSuccess],
		Data:    data,
	})
}

func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: message,
		Data:    data,
	})
}

func Error(c *gin.Context, code int, message string) {
	httpStatus := http.StatusOK
	switch code {
	case CodeUnauthorized, CodeTokenExpired:
		httpStatus = http.StatusUnauthorized
	case CodeForbidden:
		httpStatus = http.StatusForbidden
	case CodeNotFound:
		httpStatus = http.StatusNotFound
	case CodeInternalError:
		httpStatus = http.StatusInternalServerError
	}

	if message == "" {
		message = codeMessages[code]
	}

	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
	})
}

func ParamError(c *gin.Context, message string) {
	if message == "" {
		message = codeMessages[CodeParamError]
	}
	c.JSON(http.StatusBadRequest, Response{
		Code:    CodeParamError,
		Message: message,
	})
}

func Unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, Response{
		Code:    CodeUnauthorized,
		Message: codeMessages[CodeUnauthorized],
	})
}

func TokenExpired(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, Response{
		Code:    CodeTokenExpired,
		Message: codeMessages[CodeTokenExpired],
	})
}

func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = codeMessages[CodeNotFound]
	}
	c.JSON(http.StatusNotFound, Response{
		Code:    CodeNotFound,
		Message: message,
	})
}

func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = codeMessages[CodeForbidden]
	}
	c.JSON(http.StatusForbidden, Response{
		Code:    CodeForbidden,
		Message: message,
	})
}

func InternalError(c *gin.Context, message string) {
	if message == "" {
		message = codeMessages[CodeInternalError]
	}
	c.JSON(http.StatusInternalServerError, Response{
		Code:    CodeInternalError,
		Message: message,
	})
}
