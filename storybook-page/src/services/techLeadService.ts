import api from './api';
import type { ApiResponse, TechLeadsResponse } from '../types/api';
import type { Story, UserWorkload } from '../types/models';

export interface PendingStoriesResponse {
  stories: Story[];
  total: number;
  page: number;
  limit: number;
}

export interface WorkloadResponse {
  workloads: UserWorkload[];
}

export interface ReviewStoryRequest {
  approved: boolean;
  comment?: string;
}

export interface AssignStoryRequest {
  assigned_to?: number;
}

export const techLeadService = {
  // 获取待审批故事列表
  getPendingStories: async (params?: {
    project_id?: number;
    search?: string;
    page?: number;
    limit?: number;
  }) => {
    const response = await api.get<ApiResponse<PendingStoriesResponse>>('/api/techlead/pending-stories', {
      params,
    });
    return response.data.data;
  },

  // 审批故事
  reviewStory: async (storyId: number, data: ReviewStoryRequest) => {
    const response = await api.post<ApiResponse>(`/api/stories/${storyId}/review`, data);
    return response.data.data;
  },

  // 分配故事
  assignStory: async (storyId: number, assigneeId?: number) => {
    const response = await api.patch<ApiResponse>(`/api/stories/${storyId}/assignee`, {
      assigned_to: assigneeId,
    });
    return response.data.data;
  },

  // 获取工作负载统计
  getWorkload: async (projectId?: number) => {
    const response = await api.get<ApiResponse<WorkloadResponse>>('/api/techlead/workload', {
      params: projectId ? { project_id: projectId } : undefined,
    });
    return response.data.data;
  },

  // 获取技术负责人负责的项目列表
  getMyProjects: async () => {
    const response = await api.get<
      ApiResponse<{
        projects: Array<{ id: number; name: string; agile_mode: string; pending_stories: number }>;
      }>
    >('/api/techlead/projects');
    return response.data.data;
  },

  // 获取项目技术负责人列表
  getProjectTechLeads: async (projectId: number) => {
    const response = await api.get<ApiResponse<TechLeadsResponse>>(`/api/projects/${projectId}/techleads`);
    return response.data.data;
  },

  // 为项目添加技术负责人（仅admin）
  addTechLead: async (projectId: number, userId: number) => {
    const response = await api.post<ApiResponse>(`/api/projects/${projectId}/techleads`, {
      user_id: userId,
    });
    return response.data.data;
  },

  // 移除项目技术负责人（仅admin）
  removeTechLead: async (projectId: number, userId: number) => {
    const response = await api.delete<ApiResponse>(`/api/projects/${projectId}/techleads/${userId}`);
    return response.data.data;
  },
};
