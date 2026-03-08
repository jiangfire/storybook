package service

import (
	"errors"

	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/repository"
	"gorm.io/gorm"
)

var ErrForbidden = errors.New("forbidden")

func EnsureProjectAccess(db *gorm.DB, projectID, userID uint) (*model.Project, bool, error) {
	projectRepo := repository.NewProjectRepository(db)

	project, err := projectRepo.FindByID(projectID)
	if err != nil {
		return nil, false, err
	}

	if project.OwnerID == userID {
		return project, true, nil
	}

	isMember, err := projectRepo.IsMember(projectID, userID)
	if err != nil {
		return nil, false, err
	}
	if !isMember {
		return nil, false, ErrForbidden
	}

	return project, false, nil
}

func EnsureStoryAccess(db *gorm.DB, storyID, userID uint) (*model.UserStory, *model.Project, bool, error) {
	storyRepo := repository.NewStoryRepository(db)
	story, err := storyRepo.FindByID(storyID)
	if err != nil {
		return nil, nil, false, err
	}

	project, isOwner, err := EnsureProjectAccess(db, story.ProjectID, userID)
	if err != nil {
		return nil, nil, false, err
	}
	return story, project, isOwner, nil
}

// EnsureTechLeadAccess 检查用户是否是技术负责人
func EnsureTechLeadAccess(db *gorm.DB, projectID, userID uint) (bool, error) {
	// 检查是否是项目技术负责人
	var techLead model.ProjectTechLead
	if err := db.Where("project_id = ? AND user_id = ?", projectID, userID).First(&techLead).Error; err == nil {
		return true, nil
	}
	return false, nil
}

// IsTechLeadOrAdmin 检查用户是否是技术负责人或管理员
func IsTechLeadOrAdmin(db *gorm.DB, userID uint, userRole string) bool {
	if userRole == model.RoleAdmin || userRole == model.RoleTechLead {
		return true
	}
	return false
}

// CanReviewStory 检查用户是否有权限审批故事
func CanReviewStory(db *gorm.DB, story *model.UserStory, userID uint, userRole string) (bool, error) {
	// 管理员可以审批任何故事
	if userRole == model.RoleAdmin {
		return true, nil
	}

	// 技术负责人可以审批其负责项目的故事
	if userRole == model.RoleTechLead {
		isTechLead, err := EnsureTechLeadAccess(db, story.ProjectID, userID)
		if err != nil {
			return false, err
		}
		return isTechLead, nil
	}

	return false, nil
}
