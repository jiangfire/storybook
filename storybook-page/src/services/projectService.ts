import apiClient from './api';
import type {
  ApiResponse,
  CreateProjectRequest,
  ProjectListParams,
  ProjectListResponse,
  AddProjectMemberRequest,
  SprintListResponse,
  BurndownReport,
  CreateSprintRequest,
  UpdateSprintStatusRequest,
  SprintSummary,
  VelocityReportData,
  QualityReportData,
  ProjectMembersResponse,
  ProjectMemberCandidatesResponse,
} from '../types/api';
import type { Project, ProjectOverview } from '../types/models';

export const projectService = {
  /**
   * 获取项目列表
   */
  async getProjects(params?: ProjectListParams): Promise<ProjectListResponse> {
    const response = await apiClient.get<ApiResponse<ProjectListResponse>>('/api/projects', {
      params,
    });
    return response.data.data;
  },

  /**
   * 获取项目详情
   */
  async getProject(id: number): Promise<Project> {
    const response = await apiClient.get<ApiResponse<Project>>(`/api/projects/${id}`);
    return response.data.data;
  },

  /**
   * 创建项目
   */
  async createProject(data: CreateProjectRequest): Promise<Project> {
    const response = await apiClient.post<ApiResponse<Project>>('/api/projects', data);
    return response.data.data;
  },

  /**
   * 更新项目
   */
  async updateProject(id: number, data: Partial<CreateProjectRequest>): Promise<Project> {
    const response = await apiClient.put<ApiResponse<Project>>(`/api/projects/${id}`, data);
    return response.data.data;
  },

  /**
   * 删除项目
   */
  async deleteProject(id: number): Promise<void> {
    await apiClient.delete(`/api/projects/${id}`);
  },

  /**
   * 获取项目概览
   */
  async getProjectOverview(id: number): Promise<ProjectOverview> {
    const response = await apiClient.get<ApiResponse<ProjectOverview>>(
      `/api/projects/${id}/overview`
    );
    return response.data.data;
  },

  /**
   * 获取项目成员列表
   */
  async getProjectMembers(id: number) {
    const response = await apiClient.get<ApiResponse<ProjectMembersResponse>>(
      `/api/projects/${id}/members`
    );
    return response.data.data;
  },

  /**
   * 获取可添加的项目成员候选人
   */
  async getProjectMemberCandidates(id: number) {
    const response = await apiClient.get<ApiResponse<ProjectMemberCandidatesResponse>>(
      `/api/projects/${id}/member-candidates`
    );
    return response.data.data;
  },

  /**
   * 添加项目成员
   */
  async addProjectMember(id: number, data: AddProjectMemberRequest) {
    const response = await apiClient.post<ApiResponse>(`/api/projects/${id}/members`, data);
    return response.data.data;
  },

  /**
   * 移除项目成员
   */
  async removeProjectMember(id: number, userId: number) {
    const response = await apiClient.delete<ApiResponse>(`/api/projects/${id}/members/${userId}`);
    return response.data.data;
  },

  /**
   * 获取项目冲刺列表
   */
  async getSprints(id: number): Promise<SprintListResponse> {
    const response = await apiClient.get<ApiResponse<SprintListResponse>>(
      `/api/projects/${id}/sprints`
    );
    return response.data.data;
  },

  /**
   * 创建冲刺
   */
  async createSprint(id: number, data: CreateSprintRequest): Promise<SprintSummary> {
    const response = await apiClient.post<ApiResponse<SprintSummary>>(
      `/api/projects/${id}/sprints`,
      data
    );
    return response.data.data;
  },

  /**
   * 更新冲刺状态
   */
  async updateSprintStatus(id: number, data: UpdateSprintStatusRequest) {
    const response = await apiClient.patch<ApiResponse>(`/api/sprints/${id}/status`, data);
    return response.data.data;
  },

  /**
   * 获取项目燃尽图
   */
  async getBurndown(id: number, sprintId: number): Promise<BurndownReport> {
    const response = await apiClient.get<ApiResponse<BurndownReport>>(
      `/api/projects/${id}/reports/burndown`,
      { params: { sprint_id: sprintId } }
    );
    return response.data.data;
  },

  /**
   * 获取项目速度报表
   */
  async getVelocity(id: number): Promise<VelocityReportData> {
    const response = await apiClient.get<ApiResponse<VelocityReportData>>(
      `/api/projects/${id}/reports/velocity`
    );
    return response.data.data;
  },

  /**
   * 获取项目质量报表
   */
  async getQuality(id: number): Promise<QualityReportData> {
    const response = await apiClient.get<ApiResponse<QualityReportData>>(
      `/api/projects/${id}/reports/quality`
    );
    return response.data.data;
  },
};
