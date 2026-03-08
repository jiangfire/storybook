package api

// Code 是业务错误码。
type Code int

const (
	CodeSuccess Code = 0

	CodeBadRequest Code = 40001

	CodeUnauthorized Code = 40101
	CodeTokenExpired Code = 40102

	CodeForbidden Code = 40301
	CodeNotFound  Code = 40401
	CodeConflict  Code = 40901

	CodeTooManyRequests Code = 42901

	CodeInternal Code = 50001
)
