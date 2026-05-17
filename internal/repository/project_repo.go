package repository

import (
	"github.com/jiangfire/storybook/internal/model"
	"gorm.io/gorm"
)

type ProjectRepository struct {
	BaseRepository[model.Project]
}

func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{
		BaseRepository: NewBaseRepository[model.Project](db),
	}
}

func (r *ProjectRepository) FindByIDWithOwner(projectID uint) (*model.Project, error) {
	var project model.Project
	if err := r.DB().Preload("Owner").First(&project, projectID).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *ProjectRepository) ExistsByOwnerAndName(ownerID uint, name string, excludeID ...uint) (bool, error) {
	var count int64
	query := r.DB().Model(&model.Project{}).
		Where("owner_id = ? AND name = ?", ownerID, name)
	if len(excludeID) > 0 {
		query = query.Where("id <> ?", excludeID[0])
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *ProjectRepository) CreateWithTransaction(project *model.Project, member *model.ProjectMember, columns []model.BoardColumn) error {
	tx := r.DB().Begin()
	if err := tx.Create(project).Error; err != nil {
		tx.Rollback()
		return err
	}
	member.ProjectID = project.ID
	for i := range columns {
		columns[i].ProjectID = project.ID
	}
	if err := tx.Create(member).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Create(&columns).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (r *ProjectRepository) AddMember(member *model.ProjectMember) error {
	return r.DB().Create(member).Error
}

func (r *ProjectRepository) RemoveMember(projectID, userID uint) (int64, error) {
	result := r.DB().Where("project_id = ? AND user_id = ?", projectID, userID).Delete(&model.ProjectMember{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (r *ProjectRepository) IsMember(projectID, userID uint) (bool, error) {
	var count int64
	if err := r.DB().Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *ProjectRepository) GetMember(projectID, userID uint) (*model.ProjectMember, error) {
	var member model.ProjectMember
	if err := r.DB().Where("project_id = ? AND user_id = ?", projectID, userID).First(&member).Error; err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *ProjectRepository) ListByUser(userID uint, page, limit int) ([]model.Project, int64, error) {
	var total int64
	tx := r.DB().Model(&model.Project{}).
		Joins("LEFT JOIN project_members ON project_members.project_id = projects.id").
		Where("projects.owner_id = ? OR project_members.user_id = ?", userID, userID).
		Distinct("projects.id")
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var projects []model.Project
	offset := (page - 1) * limit
	if err := tx.Offset(offset).Limit(limit).Order("projects.created_at DESC").Find(&projects).Error; err != nil {
		return nil, 0, err
	}
	return projects, total, nil
}

func (r *ProjectRepository) CountMembers(projectID uint) (int64, error) {
	var count int64
	if err := r.DB().Model(&model.ProjectMember{}).Where("project_id = ?", projectID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *ProjectRepository) ListMembers(projectID uint) ([]model.ProjectMember, error) {
	var members []model.ProjectMember
	if err := r.DB().Preload("User").Where("project_id = ?", projectID).Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

// ListMemberCandidates 列出尚未加入该项目、且不是该项目 Owner 的用户,
// 供项目 Owner 添加成员时挑选;字段裁剪到展示页所需,按 email 升序。
func (r *ProjectRepository) ListMemberCandidates(projectID, ownerID uint) ([]model.User, error) {
	subQuery := r.DB().Model(&model.ProjectMember{}).
		Select("user_id").
		Where("project_id = ?", projectID)

	var users []model.User
	if err := r.DB().Model(&model.User{}).
		Select("id, email, role, avatar_url, created_at").
		Where("id <> ?", ownerID).
		Where("id NOT IN (?)", subQuery).
		Order("email ASC").
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *ProjectRepository) HasTechLead(projectID, userID uint) (bool, error) {
	var count int64
	if err := r.DB().Model(&model.ProjectTechLead{}).Where("project_id = ? AND user_id = ?", projectID, userID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *ProjectRepository) AddTechLead(lead *model.ProjectTechLead) error {
	return r.DB().Create(lead).Error
}

func (r *ProjectRepository) RemoveTechLead(projectID, userID uint) (int64, error) {
	result := r.DB().Where("project_id = ? AND user_id = ?", projectID, userID).Delete(&model.ProjectTechLead{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (r *ProjectRepository) ListTechLeads(projectID uint) ([]model.ProjectTechLead, error) {
	var leads []model.ProjectTechLead
	if err := r.DB().Preload("User").Where("project_id = ?", projectID).Find(&leads).Error; err != nil {
		return nil, err
	}
	return leads, nil
}

func (r *ProjectRepository) GetOverview(projectID uint) (*model.Project, []model.ProjectMember, []model.UserStory, []model.Sprint, error) {
	var project model.Project
	if err := r.DB().First(&project, projectID).Error; err != nil {
		return nil, nil, nil, nil, err
	}
	var members []model.ProjectMember
	if err := r.DB().Preload("User").Where("project_id = ?", projectID).Find(&members).Error; err != nil {
		return nil, nil, nil, nil, err
	}
	var stories []model.UserStory
	if err := r.DB().Where("project_id = ? AND archived = ?", projectID, false).Find(&stories).Error; err != nil {
		return nil, nil, nil, nil, err
	}
	var sprints []model.Sprint
	if err := r.DB().Where("project_id = ?", projectID).Order("start_date ASC").Find(&sprints).Error; err != nil {
		return nil, nil, nil, nil, err
	}
	return &project, members, stories, sprints, nil
}
