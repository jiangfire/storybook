import api from './api';
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
  getUsers: (params?: { role?: UserRole; search?: string; page?: number; limit?: number }) => {
    return api.get<UsersResponse>('/admin/users', { params });
  },

  // 创建用户
  createUser: (data: CreateUserRequest) => {
    return api.post('/admin/users', data);
  },

  // 更新用户
  updateUser: (id: number, data: UpdateUserRequest) => {
    return api.put(`/admin/users/${id}`, data);
  },

  // 删除用户
  deleteUser: (id: number) => {
    return api.delete(`/admin/users/${id}`);
  },

  // 获取用户工作负载详情
  getUserWorkload: (id: number) => {
    return api.get<UserWorkloadDetail>(`/admin/users/${id}/workload`);
  },
};
