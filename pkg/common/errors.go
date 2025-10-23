package common

import "errors"

var (
	// ErrNotConnected 未连接错误
	ErrNotConnected = errors.New("not connected")
	
	// ErrAlreadyConnected 已连接错误
	ErrAlreadyConnected = errors.New("already connected")
	
	// ErrNotFound 未找到错误
	ErrNotFound = errors.New("not found")
	
	// ErrInvalidInput 无效输入错误
	ErrInvalidInput = errors.New("invalid input")
	
	// ErrTimeout 超时错误
	ErrTimeout = errors.New("timeout")
	
	// ErrValidationFailed 验证失败错误
	ErrValidationFailed = errors.New("validation failed")
	
	// ErrProcessingFailed 处理失败错误
	ErrProcessingFailed = errors.New("processing failed")
	
	// ErrStorageFailed 存储失败错误
	ErrStorageFailed = errors.New("storage failed")
	
	// ErrUnauthorized 未授权错误
	ErrUnauthorized = errors.New("unauthorized")
	
	// ErrRateLimitExceeded 速率限制错误
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// AppError 应用错误
type AppError struct {
	Code    string
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// NewAppError 创建应用错误
func NewAppError(code string, message string, cause error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

