import api from './api';
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
  getPendingStories: (params?: {
    project_id?: number;
    search?: string;
    page?: number;
    limit?: number;
  }) => {
    return api.get<PendingStoriesResponse>('/techlead/pending-stories', { params });
  },

  // 审批故事
  reviewStory: (storyId: number, data: ReviewStoryRequest) => {
    return api.post(`/stories/${storyId}/review`, data);
  },

  // 分配故事
  assignStory: (storyId: number, assigneeId?: number) => {
    return api.patch(`/stories/${storyId}/assignee`, { assigned_to: assigneeId });
  },

  // 获取工作负载统计
  getWorkload: (projectId?: number) => {
    return api.get<WorkloadResponse>('/techlead/workload', {
      params: projectId ? { project_id: projectId } : undefined,
    });
  },

  // 获取技术负责人负责的项目列表
  getMyProjects: () => {
    return api.get<{
      projects: Array<{ id: number; name: string; agile_mode: string; pending_stories: number }>;
    }>('/techlead/projects');
  },

  // 获取项目技术负责人列表
  getProjectTechLeads: (projectId: number) => {
    return api.get(`/projects/${projectId}/techleads`);
  },

  // 为项目添加技术负责人（仅admin）
  addTechLead: (projectId: number, userId: number) => {
    return api.post(`/projects/${projectId}/techleads`, { user_id: userId });
  },

  // 移除项目技术负责人（仅admin）
  removeTechLead: (projectId: number, userId: number) => {
    return api.delete(`/projects/${projectId}/techleads/${userId}`);
  },
};
