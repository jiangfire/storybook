import api from './api';
import type { ApiResponse } from '../types/api';
import type { User, UserRole } from '../types/models';

export interface CreateUserRequest {
  email: string;
  username: string;
  password: string;
  role: UserRole;
}

export interface UpdateUserRequest {
  username?: string;
  role?: UserRole;
  password?: string;
}

export interface UsersResponse {
  users: User[];
  total: number;
  page: number;
  limit: number;
}

export interface UserWorkloadDetail {
  user: User;
  active_stories: Array<{
    id: number;
    title: string;
    status: string;
    story_points?: number;
    project?: { id: number; name: string };
  }>;
  active_tasks: Array<{
    id: number;
    title: string;
    status: string;
    estimated_hours?: number;
    story?: { id: number; title: string };
  }>;
  statistics: {
    total_story_points: number;
    total_estimated_hours: number;
    stories_completed_30d: number;
    stories_in_progress: number;
    tasks_completed_30d: number;
    tasks_in_progress: number;
    avg_completion_days: number;
  };
}

export const userManagementService = {
  // 获取用户列表
  getUsers: async (params?: { role?: UserRole; search?: string; page?: number; limit?: number }) => {
    const response = await api.get<ApiResponse<UsersResponse>>('/api/admin/users', { params });
    return response.data.data;
  },

  // 创建用户
  createUser: async (data: CreateUserRequest) => {
    const response = await api.post<ApiResponse>('/api/admin/users', data);
    return response.data.data;
  },

  // 更新用户
  updateUser: async (id: number, data: UpdateUserRequest) => {
    const response = await api.put<ApiResponse>(`/api/admin/users/${id}`, data);
    return response.data.data;
  },

  // 删除用户
  deleteUser: async (id: number) => {
    const response = await api.delete<ApiResponse>(`/api/admin/users/${id}`);
    return response.data.data;
  },

  // 获取用户工作负载详情
  getUserWorkload: async (id: number) => {
    const response = await api.get<ApiResponse<UserWorkloadDetail>>(`/api/admin/users/${id}/workload`);
    return response.data.data;
  },
};
