package service

import "errors"

var (
	ErrInvalidTransition    = errors.New("invalid_transition")
	ErrAlreadyClaimed       = errors.New("already_claimed")
	ErrClaimNotAllowed      = errors.New("claim_not_allowed")
	ErrNotClaimed           = errors.New("not_claimed")
	ErrNoReleasePermission  = errors.New("no_release_permission")
	ErrNoStatusPermission   = errors.New("no_status_permission")
	ErrNoProgressPermission = errors.New("no_progress_permission")
	ErrACNotFound           = errors.New("ac_not_found")
	ErrACCorrupted          = errors.New("ac_corrupted")
	// ErrACConcurrentModify 表示乐观锁重试次数耗尽仍无法写入 AC 变更，
	// 调用方应让客户端刷新后重试。
	ErrACConcurrentModify = errors.New("ac_concurrent_modify")
	ErrNoSplittableAC     = errors.New("no_splittable_ac")
)

type ValidationIssue struct {
	Field   string
	Message string
}

type ValidationError struct {
	Issues []ValidationIssue
}

func (e *ValidationError) Error() string {
	return "validation_failed"
}

func NewValidationError(issues ...ValidationIssue) *ValidationError {
	return &ValidationError{Issues: issues}
}
