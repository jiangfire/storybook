import { User, Project, Story, StoryBoardItem, Activity } from './models';

// API 统一响应格式
export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
  errors?: ValidationError[];
}

// 验证错误
export interface ValidationError {
  field: string;
  message: string;
}

// 分页响应
export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  limit: number;
}

// ===== 认证相关 =====

// 注册请求
export interface RegisterRequest {
  email: string;
  password: string;
  role: 'product' | 'developer' | 'tester';
}

// 登录请求
export interface LoginRequest {
  email: string;
  password: string;
}

// 认证响应
export interface AuthResponse {
  user: User;
  token: string;
  refresh_token?: string;
  expires_at: string;
}

// 刷新Token请求
export interface RefreshTokenRequest {
  refresh_token: string;
}

// ===== 项目相关 =====

// 创建项目请求
export interface CreateProjectRequest {
  name: string;
  description?: string;
  agile_mode: 'scrum' | 'kanban';
}

// 项目列表查询参数
export interface ProjectListParams {
  page?: number;
  limit?: number;
  search?: string;
}

// 项目列表响应
export interface ProjectListResponse {
  projects: Project[];
  total: number;
  page: number;
  limit: number;
}

// 添加项目成员请求
export interface AddProjectMemberRequest {
  user_id: number;
  role_in_project: 'product' | 'developer' | 'tester';
}

// ===== 故事相关 =====

// 创建故事请求
export interface CreateStoryRequest {
  title: string;
  description?: string;
  story_type: 'feature' | 'bug' | 'chore';
  priority?: number;
  story_points?: 1 | 2 | 3 | 5 | 8 | 13;
  acceptance_criteria?: Array<{
    id?: string;
    ref?: string;
    description: string;
    order: number;
  }>;
  tags?: string[];
}

// 更新故事请求
export interface UpdateStoryRequest {
  title?: string;
  description?: string;
  story_type?: 'feature' | 'bug' | 'chore';
  priority?: number;
  story_points?: 1 | 2 | 3 | 5 | 8 | 13;
  acceptance_criteria?: Array<{
    id?: string;
    ref?: string;
    description: string;
    order: number;
  }>;
  tags?: string[];
}

// 故事列表查询参数
export interface StoryListParams {
  status?: string;
  assignee?: number;
  sort_by?: 'priority' | 'assignee' | 'position';
  order?: 'asc' | 'desc';
  include_archived?: boolean;
}

// 故事列表响应
export interface StoryListResponse {
  stories: Story[];
  total: number;
}

// 看板数据
export interface BoardData {
  project_id: number;
  columns: Array<{
    position: number;
    name: string;
    status: string;
    stories: StoryBoardItem[];
    count: number;
  }>;
}

// 更新故事状态请求
export interface UpdateStoryStatusRequest {
  status: string;
  position?: number;
}

// 更新AC状态请求
export interface UpdateACStatusRequest {
  status: 'pending' | 'passed' | 'failed';
  evidence?: string;
}

// 活动历史查询参数
export interface ActivityListParams {
  action?: string;
  page?: number;
  limit?: number;
}

// 活动历史响应
export interface ActivityListResponse {
  activities: Activity[];
  total: number;
  page: number;
  limit: number;
}

// 关联代码引用请求
export interface AddCodeRefRequest {
  reference: string;
}

// 规划故事到冲刺
export interface PlanStoryToSprintRequest {
  sprint_id?: number | null;
}

export interface AssignStoryRequest {
  assigned_to?: number | null;
}

// ===== 任务相关 =====

// 创建任务请求
export interface CreateTaskRequest {
  title: string;
  description?: string;
  priority?: number;
  estimated_hours?: number;
}

// 更新任务请求
export interface UpdateTaskRequest {
  title?: string;
  description?: string;
  priority?: number;
  estimated_hours?: number;
}

// 更新任务状态请求
export interface UpdateTaskStatusRequest {
  status: 'todo' | 'in_progress' | 'blocked' | 'done';
}

// 更新任务进度请求
export interface UpdateTaskProgressRequest {
  progress: number;
}

// 任务列表查询参数
export interface TaskListParams {
  status?: string;
  assignee?: number;
}

// ===== 测试用例相关 =====

// 创建测试用例请求
export interface CreateTestCaseRequest {
  title: string;
  description?: string;
  steps: string[];
  expected_result?: string;
}

// ===== 缺陷相关 =====

// 创建缺陷请求
export interface CreateBugRequest {
  story_id?: number;
  title: string;
  description?: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  assigned_to?: number;
}

// 缺陷列表查询参数
export interface BugListParams {
  status?: string;
  severity?: string;
  assignee?: number;
}

// 更新缺陷状态请求
export interface UpdateBugStatusRequest {
  status: 'open' | 'in_progress' | 'resolved' | 'closed';
}

// 指派缺陷请求
export interface AssignBugRequest {
  assigned_to?: number;
}

// ===== 冲刺相关 =====

// 创建冲刺请求
export interface CreateSprintRequest {
  name: string;
  goal?: string;
  start_date: string;
  end_date: string;
}

export interface SprintSummary {
  id: number;
  project_id: number;
  name: string;
  goal?: string;
  start_date: string;
  end_date: string;
  status: 'planned' | 'active' | 'completed';
  total_stories: number;
  done_stories: number;
  created_at: string;
  updated_at: string;
}

export interface SprintListResponse {
  sprints: SprintSummary[];
}

// 更新冲刺状态请求
export interface UpdateSprintStatusRequest {
  status: 'planned' | 'active' | 'completed';
}

export interface StorySprintPlanResponse {
  story_id: number;
  sprint_id?: number | null;
  updated_at: string;
}

// ===== 报表相关 =====

// 速率报表数据
export interface VelocityReport {
  sprints: Array<{
    id: number;
    name: string;
    story_points_completed: number;
  }>;
  average_velocity: number;
}

// 质量报表数据
export interface QualityReport {
  total_bugs: number;
  bugs_by_severity: Record<string, number>;
  bugs_by_status: Record<string, number>;
  bug_creation_rate: number;
  bug_resolution_rate: number;
}

// 燃尽图数据
export interface BurndownData {
  dates: string[];
  ideal: number[];
  actual: number[];
}

export interface BurndownPoint {
  date: string;
  remaining_points: number;
}

export interface BurndownReport {
  project_id: number;
  sprint: {
    id: number;
    name: string;
    start_date: string;
    end_date: string;
  };
  baseline_points: number;
  points: BurndownPoint[];
}

// ===== 搜索相关 =====

// 搜索查询参数
export interface SearchParams {
  q: string;
  type?: 'all' | 'project' | 'story' | 'bug';
  limit?: number;
}

// 搜索结果
export interface SearchResult {
  type: 'project' | 'story' | 'bug';
  id: number;
  title: string;
  description?: string;
  project_id?: number;
  project_name?: string;
}

// ===== AI相关 =====

// AI生成故事请求
export interface AIGenerateStoryRequest {
  requirement: string;
}

// AI生成故事响应
export interface AIGenerateStoryResponse {
  suggested_title: string;
  suggested_description: string;
  suggested_ac: Array<{
    description: string;
  }>;
  suggested_story_points: number;
}

// AI拆分故事请求
export interface AISplitStoryRequest {
  target_count?: number;
}

// AI拆分故事响应
export interface AISplitStoryResponse {
  sub_stories: Array<{
    title: string;
    description: string;
    story_points: number;
  }>;
}

// INVEST检查响应
export interface INVESTCheckResponse {
  story_id: number;
  invest_score: number;
  checks: {
    independent: CheckResult;
    negotiable: CheckResult;
    valuable: CheckResult;
    estimable: CheckResult;
    small: CheckResult;
    testable: CheckResult;
  };
  suggestions: string[];
}

// 检查结果
export interface CheckResult {
  score: number;
  status: 'pass' | 'warning' | 'fail';
  feedback: string;
}

// ===== Dashboard相关 =====

// 个人工作台数据
export interface DashboardData {
  user: User;
  my_stories: {
    assigned: Array<{
      id: number;
      title: string;
      project?: string;
      status: string;
      priority: number;
      story_type?: string;
    }>;
    created: Array<{
      id: number;
      title: string;
      project?: string;
      status: string;
      priority: number;
      story_type?: string;
    }>;
  };
  statistics: {
    total_assigned: number;
    in_progress: number;
    completed: number;
  };
}

// ===== WebSocket相关 =====

// WebSocket消息类型
export type WSMessageType =
  | 'story.status_changed'
  | 'story.ac_updated'
  | 'story.created'
  | 'story.updated'
  | 'story.deleted'
  | 'task.updated'
  | 'bug.created';

// WebSocket消息基础
export interface WSMessage<T = unknown> {
  type: WSMessageType;
  data: T;
  timestamp: string;
}

// 故事状态变更消息
export interface StoryStatusChangedMessage {
  story_id: number;
  project_id: number;
  old_status: string;
  new_status: string;
  actor: {
    id: number;
    email: string;
  };
}

// AC状态更新消息
export interface StoryACUpdatedMessage {
  story_id: number;
  ac_id: string;
  ac_status: string;
  actor: {
    id: number;
    email: string;
  };
}
