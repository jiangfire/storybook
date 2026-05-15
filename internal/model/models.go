package model

import (
	"encoding/json"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	RoleProduct   = "product"   // 产品经理
	RoleDeveloper = "developer" // 开发人员
	RoleTester    = "tester"    // 测试人员
	RoleTechLead  = "tech_lead" // 技术负责人
	RoleAdmin     = "admin"     // 管理员
)

const (
	AgileModeScrum  = "scrum"
	AgileModeKanban = "kanban"
)

const (
	StoryTypeFeature = "feature"
	StoryTypeBug     = "bug"
	StoryTypeChore   = "chore"
)

const (
	StoryStatusPending    = "pending"     // 待审批（新增）
	StoryStatusBacklog    = "backlog"     // 待办
	StoryStatusReady      = "ready"       // 就绪
	StoryStatusInProgress = "in_progress" // 进行中
	StoryStatusTest       = "test"        // 测试中
	StoryStatusDone       = "done"        // 已完成
)

const (
	ReviewStatusPending  = "pending"
	ReviewStatusApproved = "approved"
	ReviewStatusRejected = "rejected"
)

const (
	ACStatusPending = "pending"
	ACStatusPassed  = "passed"
	ACStatusFailed  = "failed"
)

const (
	SprintStatusPlanned   = "planned"
	SprintStatusActive    = "active"
	SprintStatusCompleted = "completed"
	SprintStatusCancelled = "cancelled"
)

const (
	BugSeverityLow      = "low"
	BugSeverityMedium   = "medium"
	BugSeverityHigh     = "high"
	BugSeverityCritical = "critical"
)

const (
	BugStatusOpen       = "open"
	BugStatusInProgress = "in_progress"
	BugStatusResolved   = "resolved"
	BugStatusClosed     = "closed"
)

const (
	TaskStatusTodo       = "todo"
	TaskStatusInProgress = "in_progress"
	TaskStatusBlocked    = "blocked"
	TaskStatusDone       = "done"
)

const (
	NotificationStoryAssigned    = "story.assigned"
	NotificationStoryClaimed     = "story.claimed"
	NotificationStoryReleased    = "story.released"
	NotificationStoryReviewed    = "story.reviewed"
	NotificationTaskAssigned     = "task.assigned"
	NotificationBugAssigned      = "bug.assigned"
	NotificationSprintStarted    = "sprint.started"
	NotificationSprintCompleted  = "sprint.completed"

	NotificationEntityStory  = "story"
	NotificationEntityTask   = "task"
	NotificationEntityBug    = "bug"
	NotificationEntitySprint = "sprint"
)

// User 系统用户。
type User struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	Username            string     `gorm:"size:100;uniqueIndex" json:"username"`
	Email               string     `gorm:"size:255;uniqueIndex;not null" json:"email"`
	HashedPassword      string     `gorm:"size:255;not null" json:"-"`
	Role                string     `gorm:"size:20;not null" json:"role"`
	AvatarURL           string     `gorm:"size:500" json:"avatar_url,omitempty"`
	FailedLoginAttempts int        `gorm:"not null;default:0" json:"-"`
	LockedUntil         *time.Time `json:"-"`
	LastLoginAt         *time.Time      `json:"last_login_at,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	DeletedAt           gorm.DeletedAt  `gorm:"index" json:"-"`
	Version             int             `gorm:"default:0" json:"version"`
}

// Project 项目。
type Project struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:255;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	OwnerID     uint      `gorm:"not null;index" json:"owner_id"`
	Owner       *User     `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	AgileMode   string    `gorm:"size:20;not null;default:kanban;index" json:"agile_mode"`
	Archived    bool      `gorm:"not null;default:false;index" json:"archived"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Version     int            `gorm:"default:0" json:"version"`
}

// ProjectMember 项目成员。
type ProjectMember struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ProjectID     uint      `gorm:"not null;uniqueIndex:idx_project_member" json:"project_id"`
	Project       *Project  `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	UserID        uint      `gorm:"not null;uniqueIndex:idx_project_member" json:"user_id"`
	User          *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	RoleInProject string         `gorm:"size:20;not null" json:"role_in_project"`
	JoinedAt      time.Time      `json:"joined_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	Version       int            `gorm:"default:0" json:"version"`
}

// BoardColumn 看板列。
type BoardColumn struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProjectID uint      `gorm:"not null;index;uniqueIndex:idx_project_position" json:"project_id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Position  int       `gorm:"not null;uniqueIndex:idx_project_position" json:"position"`
	Color     string    `gorm:"size:20;default:#6B7280" json:"color"`
	WIPLimit  int            `gorm:"default:0" json:"wip_limit"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Version   int            `gorm:"default:0" json:"version"`
}

// Sprint 冲刺。
type Sprint struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProjectID uint      `gorm:"not null;index" json:"project_id"`
	Project   *Project  `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	Name      string    `gorm:"size:120;not null" json:"name"`
	Goal      string    `gorm:"type:text" json:"goal,omitempty"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Status    string    `gorm:"size:20;not null;default:planned;index" json:"status"`
	CreatedBy uint           `gorm:"not null;index" json:"created_by"`
	Creator   *User          `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Version   int            `gorm:"default:0" json:"version"`
}

// UserStory 用户故事。
type UserStory struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	ProjectID          uint           `gorm:"not null;index" json:"project_id"`
	Project            *Project       `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	Title              string         `gorm:"size:500;not null" json:"title"`
	Description        string         `gorm:"type:text" json:"description,omitempty"`
	StoryType          string         `gorm:"size:50;not null;default:feature" json:"story_type"`
	Status             string         `gorm:"size:50;not null;default:backlog;index" json:"status"`
	ReviewStatus       string         `gorm:"size:20;not null;default:pending;index" json:"review_status"`
	ReviewComment      string         `gorm:"type:text" json:"review_comment,omitempty"`
	Archived           bool           `gorm:"not null;default:false;index" json:"archived"`
	Priority           int            `gorm:"default:0;index" json:"priority"`
	Points             *int           `json:"story_points,omitempty"`
	AssignedTo         *uint          `gorm:"index" json:"assigned_to,omitempty"`
	Assignee           *User          `gorm:"foreignKey:AssignedTo" json:"assignee,omitempty"`
	CreatedBy          uint           `gorm:"not null" json:"created_by"`
	Creator            *User          `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	SprintID           *uint          `gorm:"index" json:"sprint_id,omitempty"`
	Sprint             *Sprint        `gorm:"foreignKey:SprintID" json:"sprint,omitempty"`
	Position           float64        `gorm:"default:0;index" json:"position"`
	AcceptanceCriteria datatypes.JSON `gorm:"type:jsonb;not null" json:"acceptance_criteria"`
	Tags               datatypes.JSON `gorm:"type:jsonb;not null" json:"tags,omitempty"`
	CodeReferences     datatypes.JSON `gorm:"type:jsonb;not null" json:"code_references,omitempty"`
	ReviewedBy         *uint          `gorm:"index" json:"reviewed_by,omitempty"`
	Reviewer           *User          `gorm:"foreignKey:ReviewedBy" json:"reviewer,omitempty"`
	ReviewedAt         *time.Time      `json:"reviewed_at,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	DeletedAt          gorm.DeletedAt  `gorm:"index" json:"-"`
	Version            int             `gorm:"default:0" json:"version"`
}

// BugReport 缺陷记录。
type BugReport struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	ProjectID   uint       `gorm:"not null;index" json:"project_id"`
	Project     *Project   `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	StoryID     *uint      `gorm:"index" json:"story_id,omitempty"`
	Story       *UserStory `gorm:"foreignKey:StoryID" json:"story,omitempty"`
	Title       string     `gorm:"size:255;not null" json:"title"`
	Description string     `gorm:"type:text" json:"description,omitempty"`
	Severity    string     `gorm:"size:20;not null;default:medium;index" json:"severity"`
	Status      string     `gorm:"size:20;not null;default:open;index" json:"status"`
	ReportedBy  uint       `gorm:"not null;index" json:"reported_by"`
	Reporter    *User      `gorm:"foreignKey:ReportedBy" json:"reporter,omitempty"`
	AssignedTo  *uint      `gorm:"index" json:"assigned_to,omitempty"`
	Assignee    *User      `gorm:"foreignKey:AssignedTo" json:"assignee,omitempty"`
	ResolvedAt  *time.Time      `json:"resolved_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	DeletedAt   gorm.DeletedAt  `gorm:"index" json:"-"`
	Version     int             `gorm:"default:0" json:"version"`
}

// BugComment 缺陷评论。记录用户在缺陷上下文中的讨论与状态更新备注，
// 仅作者本人或项目管理员可编辑/删除,所有项目成员均可查看。
type BugComment struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	BugID     uint           `gorm:"not null;index" json:"bug_id"`
	Bug       *BugReport     `gorm:"foreignKey:BugID" json:"bug,omitempty"`
	AuthorID  uint           `gorm:"not null;index" json:"author_id"`
	Author    *User          `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	Body      string         `gorm:"type:text;not null" json:"body"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Version   int            `gorm:"default:0" json:"version"`
}

// Task 子任务。
type Task struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	ProjectID      uint           `gorm:"not null;index" json:"project_id"`
	Project        *Project       `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	StoryID        uint           `gorm:"not null;index" json:"story_id"`
	Story          *UserStory     `gorm:"foreignKey:StoryID" json:"story,omitempty"`
	Title          string         `gorm:"size:255;not null" json:"title"`
	Description    string         `gorm:"type:text" json:"description,omitempty"`
	Status         string         `gorm:"size:20;not null;default:todo;index" json:"status"`
	Priority       int            `gorm:"not null;default:0;index" json:"priority"`
	Progress       int            `gorm:"not null;default:0" json:"progress"`
	EstimatedHours float64        `gorm:"not null;default:0" json:"estimated_hours"`
	AssignedTo     *uint          `gorm:"index" json:"assigned_to,omitempty"`
	Assignee       *User          `gorm:"foreignKey:AssignedTo" json:"assignee,omitempty"`
	CreatedBy      uint           `gorm:"not null;index" json:"created_by"`
	Creator        *User          `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	CodeReferences datatypes.JSON  `gorm:"type:jsonb;not null" json:"code_references,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	DeletedAt      gorm.DeletedAt  `gorm:"index" json:"-"`
	Version        int             `gorm:"default:0" json:"version"`
}

// ActivityLog 活动日志。
type ActivityLog struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	EntityType string         `gorm:"size:50;not null;index:idx_entity" json:"entity_type"`
	EntityID   uint           `gorm:"not null;index:idx_entity" json:"entity_id"`
	Action     string         `gorm:"size:50;not null" json:"action"`
	OldValue   datatypes.JSON `gorm:"type:jsonb" json:"old_value,omitempty"`
	NewValue   datatypes.JSON `gorm:"type:jsonb" json:"new_value,omitempty"`
	UserID     uint           `gorm:"not null;index" json:"user_id"`
	User       *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ProjectID  *uint           `gorm:"index" json:"project_id,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	DeletedAt  gorm.DeletedAt  `gorm:"index" json:"-"`
	Version    int             `gorm:"default:0" json:"version"`
}

// TestCase 测试用例。
type TestCase struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	StoryID        uint           `gorm:"not null;index" json:"story_id"`
	Story          *UserStory     `gorm:"foreignKey:StoryID" json:"story,omitempty"`
	Title          string         `gorm:"size:255;not null" json:"title"`
	Description    string         `gorm:"type:text" json:"description,omitempty"`
	Steps          datatypes.JSON `gorm:"type:jsonb;not null" json:"steps"`
	ExpectedResult string         `gorm:"type:text" json:"expected_result,omitempty"`
	Status         string         `gorm:"size:20;not null;default:pending" json:"status"`
	CreatedBy      uint           `gorm:"not null;index" json:"created_by"`
	Creator        *User          `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt  `gorm:"index" json:"-"`
	Version    int             `gorm:"default:0" json:"version"`
}

// AcceptanceCriterion 验收标准。
type AcceptanceCriterion struct {
	ID          string     `json:"id"`
	Ref         string     `json:"ref,omitempty"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Evidence    string     `json:"evidence,omitempty"`
	VerifiedBy  *uint      `json:"verified_by,omitempty"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
	Order       int        `json:"order"`
	Notes       string     `json:"notes,omitempty"`
}

func MarshalJSON(value any) datatypes.JSON {
	if value == nil {
		return datatypes.JSON([]byte("null"))
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return datatypes.JSON([]byte("null"))
	}
	return datatypes.JSON(raw)
}

func ParseAcceptanceCriteria(raw datatypes.JSON) ([]AcceptanceCriterion, error) {
	if len(raw) == 0 {
		return []AcceptanceCriterion{}, nil
	}

	var criteria []AcceptanceCriterion
	if err := json.Unmarshal(raw, &criteria); err != nil {
		return nil, err
	}
	return criteria, nil
}

func ParseJSONMap(raw datatypes.JSON) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}

	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return value, nil
}

// ProjectTechLead 项目技术负责人（多对多关联）
type ProjectTechLead struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ProjectID  uint      `gorm:"not null;uniqueIndex:idx_project_techlead" json:"project_id"`
	UserID     uint      `gorm:"not null;uniqueIndex:idx_project_techlead" json:"user_id"`
	User       *User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	AssignedAt time.Time       `json:"assigned_at"`
	DeletedAt  gorm.DeletedAt  `gorm:"index" json:"-"`
	Version    int             `gorm:"default:0" json:"version"`
}

// Notification 站内通知。一条 Notification 表示一个用户应当看到的事件，
// 实体类型/ID 用于跳转，Type 区分语义（story.assigned 等），Metadata 透传
// 业务字段，前端可据此渲染不同图标 / 文案。
type Notification struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	UserID     uint           `gorm:"not null;index:idx_notif_user_read" json:"user_id"`
	ActorID    *uint          `gorm:"index" json:"actor_id,omitempty"`
	Actor      *User          `gorm:"foreignKey:ActorID" json:"actor,omitempty"`
	Type       string         `gorm:"size:64;not null;index" json:"type"`
	EntityType string         `gorm:"size:50;not null;index:idx_notif_entity" json:"entity_type"`
	EntityID   uint           `gorm:"not null;index:idx_notif_entity" json:"entity_id"`
	ProjectID  *uint          `gorm:"index" json:"project_id,omitempty"`
	Title      string         `gorm:"size:255;not null" json:"title"`
	Body       string         `gorm:"type:text" json:"body,omitempty"`
	Metadata   datatypes.JSON `gorm:"type:jsonb" json:"metadata,omitempty"`
	ReadAt     *time.Time     `gorm:"index:idx_notif_user_read" json:"read_at,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}
