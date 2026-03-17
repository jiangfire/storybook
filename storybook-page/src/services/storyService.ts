import apiClient from './api';
import type {
  ApiResponse,
  CreateStoryRequest,
  UpdateStoryRequest,
  StoryListParams,
  StoryListResponse,
  BoardData,
  UpdateStoryStatusRequest,
  UpdateACStatusRequest,
  ActivityListParams,
  ActivityListResponse,
  AddCodeRefRequest,
  PlanStoryToSprintRequest,
  StorySprintPlanResponse,
  AssignStoryRequest,
} from '../types/api';
import type { Story } from '../types/models';

export const storyService = {
  /**
   * 获取项目的故事列表
   */
  async getStories(projectId: number, params?: StoryListParams): Promise<StoryListResponse> {
    const response = await apiClient.get<ApiResponse<StoryListResponse>>(
      `/api/projects/${projectId}/stories`,
      { params }
    );
    return response.data.data;
  },

  /**
   * 获取看板数据
   */
  async getBoardData(projectId: number): Promise<BoardData> {
    const response = await apiClient.get<ApiResponse<BoardData>>(
      `/api/projects/${projectId}/board`
    );
    return response.data.data;
  },

  /**
   * 获取故事详情
   */
  async getStory(id: number): Promise<Story> {
    const response = await apiClient.get<ApiResponse<Story>>(`/api/stories/${id}`);
    return response.data.data;
  },

  /**
   * 创建故事
   */
  async createStory(projectId: number, data: CreateStoryRequest): Promise<Story> {
    const response = await apiClient.post<ApiResponse<Story>>(
      `/api/projects/${projectId}/stories`,
      data
    );
    return response.data.data;
  },

  /**
   * 更新故事
   */
  async updateStory(id: number, data: UpdateStoryRequest): Promise<Story> {
    const response = await apiClient.put<ApiResponse<Story>>(`/api/stories/${id}`, data);
    return response.data.data;
  },

  /**
   * 删除故事
   */
  async deleteStory(id: number): Promise<void> {
    await apiClient.delete(`/api/stories/${id}`);
  },

  /**
   * 归档故事
   */
  async archiveStory(id: number): Promise<void> {
    await apiClient.patch(`/api/stories/${id}/archive`);
  },

  /**
   * 恢复故事
   */
  async restoreStory(id: number): Promise<void> {
    await apiClient.patch(`/api/stories/${id}/restore`);
  },

  /**
   * 更新故事状态
   */
  async updateStoryStatus(id: number, data: UpdateStoryStatusRequest): Promise<Story> {
    const response = await apiClient.patch<ApiResponse<Story>>(`/api/stories/${id}/status`, data);
    return response.data.data;
  },

  /**
   * 领取故事
   */
  async claimStory(id: number): Promise<Story> {
    const response = await apiClient.post<ApiResponse<Story>>(`/api/stories/${id}/claim`);
    return response.data.data;
  },

  /**
   * 释放故事
   */
  async releaseStory(id: number): Promise<Story> {
    const response = await apiClient.delete<ApiResponse<Story>>(`/api/stories/${id}/claim`);
    return response.data.data;
  },

  /**
   * 分配故事负责人（产品经理）
   */
  async assignStory(id: number, data: AssignStoryRequest): Promise<Story> {
    const response = await apiClient.patch<ApiResponse<Story>>(`/api/stories/${id}/assignee`, data);
    return response.data.data;
  },

  /**
   * 更新验收标准状态
   */
  async updateACStatus(storyId: number, acId: string, data: UpdateACStatusRequest): Promise<void> {
    await apiClient.patch(`/api/stories/${storyId}/acceptance-criteria/${acId}`, data);
  },

  /**
   * 关联代码引用
   */
  async addCodeRef(storyId: number, data: AddCodeRefRequest): Promise<void> {
    await apiClient.post(`/api/stories/${storyId}/code-refs`, data);
  },

  /**
   * 获取活动历史
   */
  async getActivities(storyId: number, params?: ActivityListParams): Promise<ActivityListResponse> {
    const response = await apiClient.get<ApiResponse<ActivityListResponse>>(
      `/api/stories/${storyId}/activities`,
      { params }
    );
    return response.data.data;
  },

  /**
   * 规划故事到冲刺
   */
  async planToSprint(
    storyId: number,
    data: PlanStoryToSprintRequest
  ): Promise<StorySprintPlanResponse> {
    const response = await apiClient.patch<ApiResponse<StorySprintPlanResponse>>(
      `/api/stories/${storyId}/sprint`,
      data
    );
    return response.data.data;
  },
};
