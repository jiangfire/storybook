package service

import (
	"errors"
	"sort"

	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/repository"
	"gorm.io/gorm"
)

var ErrForbidden = errors.New("forbidden")

func EnsureProjectAccess(db *gorm.DB, projectID, userID uint) (*model.Project, bool, error) {
	projectRepo := repository.NewProjectRepository(db)

	project, err := projectRepo.FindByIDWithOwner(projectID)
	if err != nil {
		return nil, false, err
	}

	if project.OwnerID == userID {
		return project, true, nil
	}

	var user model.User
	if err := db.Select("id, role").First(&user, userID).Error; err != nil {
		return nil, false, err
	}
	if user.Role == model.RoleAdmin {
		return project, false, nil
	}

	isMember, err := projectRepo.IsMember(projectID, userID)
	if err != nil {
		return nil, false, err
	}
	if isMember {
		return project, false, nil
	}

	if user.Role == model.RoleTechLead {
		isAssigned, err := EnsureTechLeadAccess(db, projectID, userID)
		if err != nil {
			return nil, false, err
		}
		if isAssigned {
			return project, false, nil
		}
	}

	return nil, false, ErrForbidden
}

func EnsureStoryAccess(db *gorm.DB, storyID, userID uint) (*model.UserStory, *model.Project, bool, error) {
	storyRepo := repository.NewStoryRepository(db)
	story, err := storyRepo.FindByIDWithDetails(storyID)
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
	var count int64
	if err := db.Model(&model.ProjectTechLead{}).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func AccessibleProjectIDs(db *gorm.DB, userID uint, userRole string) ([]uint, error) {
	if userRole == model.RoleAdmin {
		var projectIDs []uint
		if err := db.Model(&model.Project{}).Order("id ASC").Pluck("id", &projectIDs).Error; err != nil {
			return nil, err
		}
		return projectIDs, nil
	}

	ids := make(map[uint]struct{})

	var ownerIDs []uint
	if err := db.Model(&model.Project{}).Where("owner_id = ?", userID).Pluck("id", &ownerIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range ownerIDs {
		ids[id] = struct{}{}
	}

	var memberIDs []uint
	if err := db.Model(&model.ProjectMember{}).Where("user_id = ?", userID).Pluck("project_id", &memberIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range memberIDs {
		ids[id] = struct{}{}
	}

	if userRole == model.RoleTechLead {
		var techLeadIDs []uint
		if err := db.Model(&model.ProjectTechLead{}).Where("user_id = ?", userID).Pluck("project_id", &techLeadIDs).Error; err != nil {
			return nil, err
		}
		for _, id := range techLeadIDs {
			ids[id] = struct{}{}
		}
	}

	projectIDs := make([]uint, 0, len(ids))
	for id := range ids {
		projectIDs = append(projectIDs, id)
	}
	sort.Slice(projectIDs, func(i, j int) bool { return projectIDs[i] < projectIDs[j] })
	return projectIDs, nil
}

// CanReviewStory 检查用户是否有权限审批故事
func CanReviewStory(db *gorm.DB, story *model.UserStory, userID uint, userRole string) (bool, error) {
	if userRole == model.RoleAdmin {
		return true, nil
	}
	if userRole == model.RoleTechLead {
		return EnsureTechLeadAccess(db, story.ProjectID, userID)
	}

	return false, nil
}
