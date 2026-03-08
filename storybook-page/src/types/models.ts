// 用户角色
export type UserRole = 'product' | 'developer' | 'tester' | 'tech_lead';

// 用户
export interface User {
  id: number;
  email: string;
  role: UserRole;
  avatar_url?: string;
  created_at: string;
}

// 故事类型
export type StoryType = 'feature' | 'bug' | 'chore';

// 故事状态
export type StoryStatus = 'pending' | 'backlog' | 'ready' | 'in_progress' | 'test' | 'done';

// 敏捷模式
export type AgileMode = 'scrum' | 'kanban';

// 验收标准状态
export type ACStatus = 'pending' | 'passed' | 'failed';

// 验收标准
export interface AcceptanceCriteria {
  id: string;
  ref?: string;
  description: string;
  status: ACStatus;
  evidence?: string;
  order: number;
}

// 用户故事
export interface Story {
  id: number;
  project_id: number;
  title: string;
  description?: string;
  story_type: StoryType;
  status: StoryStatus;
  priority: number;
  story_points?: number;
  position: number;
  assigned_to?: User;
  created_by: User;
  acceptance_criteria: AcceptanceCriteria[];
  tags?: string[];
  created_at: string;
  updated_at: string;
  sprint_id?: number;
}

// 看板项（后端 board 接口返回的轻量结构）
export interface StoryBoardItem {
  id: number;
  title: string;
  story_type: StoryType;
  status: StoryStatus;
  priority: number;
  story_points?: number;
  position?: number;
  assigned_to?: User;
  assignee?: User;
  acceptance_criteria?: AcceptanceCriteria[];
  acceptance_criteria_summary?: ACSummary;
}

// AC 摘要（用于卡片展示）
export interface ACSummary {
  total: number;
  passed: number;
  pending: number;
  completion_percentage: number;
}

// 项目成员角色
export type ProjectRole = 'product' | 'developer' | 'tester';

// 项目成员
export interface ProjectMember {
  user: User;
  role_in_project: ProjectRole;
}

// 项目
export interface Project {
  id: number;
  name: string;
  description?: string;
  agile_mode: AgileMode;
  owner: User;
  members?: ProjectMember[];
  member_count?: number;
  story_count?: number;
  is_owner?: boolean;
  created_at: string;
  updated_at?: string;
}

// 状态分布
export interface StatusBreakdown {
  backlog: number;
  ready: number;
  in_progress: number;
  test: number;
  done: number;
}

// 项目统计
export interface ProjectStatistics {
  total_stories: number;
  status_breakdown: StatusBreakdown;
  completion_rate: number;
  active_members: number;
  avg_story_points: number;
}

// 项目概览
export interface ProjectOverview {
  project: Pick<Project, 'id' | 'name'>;
  statistics: ProjectStatistics;
  recent_activities: Activity[];
}

// 活动实体类型
export type EntityType = 'story' | 'task' | 'project' | 'bug';

// 活动操作类型
export type ActionType = 'created' | 'updated' | 'status_changed' | 'assigned' | 'commented';

// 活动记录
export interface Activity {
  id: number;
  entity_type: EntityType;
  entity_id: number;
  action: ActionType;
  old_value?: Record<string, unknown>;
  new_value?: Record<string, unknown>;
  user: User & { avatar_url?: string };
  created_at: string;
}

// 任务状态
export type TaskStatus = 'todo' | 'in_progress' | 'blocked' | 'done';

// 任务
export interface Task {
  id: number;
  story_id: number;
  title: string;
  description?: string;
  status: TaskStatus;
  priority: number;
  estimated_hours?: number;
  progress: number;
  assigned_to?: User;
  created_by: User;
  created_at: string;
  updated_at: string;
}

// 测试用例
export interface TestCase {
  id: number;
  story_id: number;
  title: string;
  description?: string;
  steps: string[];
  expected_result?: string;
  status: ACStatus;
  created_by: User;
  created_at: string;
}

// 缺陷严重程度
export type BugSeverity = 'low' | 'medium' | 'high' | 'critical';

// 缺陷状态
export type BugStatus = 'open' | 'in_progress' | 'resolved' | 'closed';

// 缺陷
export interface Bug {
  id: number;
  project_id: number;
  story_id?: number;
  title: string;
  description?: string;
  severity: BugSeverity;
  status: BugStatus;
  assigned_to?: User;
  created_by: User;
  created_at: string;
  updated_at: string;
}

// 冲刺状态
export type SprintStatus = 'planned' | 'active' | 'completed';

// 冲刺
export interface Sprint {
  id: number;
  project_id: number;
  name: string;
  goal?: string;
  start_date: string;
  end_date: string;
  status: SprintStatus;
  created_at: string;
}

// 用户工作负载
export interface UserWorkload {
  user: User;
  active_stories: number;
  active_tasks: number;
  total_story_points: number;
  estimated_hours: number;
  completion_rate: number;
}
