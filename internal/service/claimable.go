package service

import (
	"git.neolidy.top/neo/storybook/internal/logging"
	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ClaimableReader exposes the read-only fields needed by claim/release
// helpers without requiring pointer-receiver methods on the value type.
type ClaimableReader interface {
	GetID() uint
	GetProjectID() uint
	GetAssignedTo() *uint
	GetStatus() string
}

// executeClaim runs the standard claim transaction skeleton:
// SELECT FOR UPDATE → already-claimed check → optional extra validation →
// mutate → Save → activity log.
func executeClaim[T ClaimableReader](
	db *gorm.DB,
	entityID uint,
	userID uint,
	entityType string,
	extraValidate func(current T) error,
	mutate func(current *T),
) (T, error) {
	var result T
	err := db.Transaction(func(tx *gorm.DB) error {
		var current T
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, entityID).Error; err != nil {
			return err
		}
		if current.GetAssignedTo() != nil && *current.GetAssignedTo() != userID {
			return ErrAlreadyClaimed
		}
		if extraValidate != nil {
			if err := extraValidate(current); err != nil {
				return err
			}
		}
		oldAssigned := current.GetAssignedTo()
		oldStatus := current.GetStatus()
		mutate(&current)
		if err := tx.Save(&current).Error; err != nil {
			return err
		}
		pid := current.GetProjectID()
		logging.LogIfErr(WriteActivityLog(tx, &pid, userID, entityType, current.GetID(), "claimed",
			map[string]any{"status": oldStatus, "assigned_to": oldAssigned},
			map[string]any{"status": current.GetStatus(), "assigned_to": userID}),
			"write activity log", "entity_id", current.GetID(), "action", "claimed")
		result = current
		return nil
	})
	return result, err
}

// executeRelease runs the standard release transaction skeleton:
// SELECT FOR UPDATE → not-claimed check → permission check → optional extra
// validation → mutate → Save → activity log.
func executeRelease[T ClaimableReader](
	db *gorm.DB,
	entityID uint,
	userID uint,
	role string,
	entityType string,
	extraValidate func(current T) error,
	mutate func(current *T),
) (T, error) {
	var result T
	err := db.Transaction(func(tx *gorm.DB) error {
		var current T
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, entityID).Error; err != nil {
			return err
		}
		if current.GetAssignedTo() == nil {
			return ErrNotClaimed
		}
		if *current.GetAssignedTo() != userID && role != model.RoleProduct && role != model.RoleAdmin {
			return ErrNoReleasePermission
		}
		if extraValidate != nil {
			if err := extraValidate(current); err != nil {
				return err
			}
		}
		oldAssigned := *current.GetAssignedTo()
		oldStatus := current.GetStatus()
		mutate(&current)
		if err := tx.Save(&current).Error; err != nil {
			return err
		}
		pid := current.GetProjectID()
		logging.LogIfErr(WriteActivityLog(tx, &pid, userID, entityType, current.GetID(), "released",
			map[string]any{"status": oldStatus, "assigned_to": oldAssigned},
			map[string]any{"status": current.GetStatus(), "assigned_to": nil}),
			"write activity log", "entity_id", current.GetID(), "action", "released")
		result = current
		return nil
	})
	return result, err
}
