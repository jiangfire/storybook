import apiClient from './api';
import type {
  ApiResponse,
  CreateTaskRequest,
  TaskItem,
  TaskListParams,
  TaskListResponse,
  TaskSplitResponse,
  UpdateTaskProgressRequest,
  UpdateTaskRequest,
  UpdateTaskStatusRequest,
} from '../types/api';

export const taskService = {
  async getStoryTasks(storyId: number, params?: TaskListParams): Promise<TaskListResponse> {
    const response = await apiClient.get<ApiResponse<TaskListResponse>>(`/api/stories/${storyId}/tasks`, {
      params,
    });
    return response.data.data;
  },

  async createTask(storyId: number, data: CreateTaskRequest): Promise<TaskItem> {
    const response = await apiClient.post<ApiResponse<TaskItem>>(`/api/stories/${storyId}/tasks`, data);
    return response.data.data;
  },

  async splitFromAC(storyId: number): Promise<TaskSplitResponse> {
    const response = await apiClient.post<ApiResponse<TaskSplitResponse>>(
      `/api/stories/${storyId}/tasks/split-from-ac`
    );
    return response.data.data;
  },

  async getTask(id: number): Promise<TaskItem> {
    const response = await apiClient.get<ApiResponse<TaskItem>>(`/api/tasks/${id}`);
    return response.data.data;
  },

  async updateTask(id: number, data: UpdateTaskRequest): Promise<TaskItem> {
    const response = await apiClient.put<ApiResponse<TaskItem>>(`/api/tasks/${id}`, data);
    return response.data.data;
  },

  async deleteTask(id: number): Promise<void> {
    await apiClient.delete(`/api/tasks/${id}`);
  },

  async updateTaskStatus(id: number, data: UpdateTaskStatusRequest): Promise<TaskItem> {
    const response = await apiClient.patch<ApiResponse<TaskItem>>(`/api/tasks/${id}/status`, data);
    return response.data.data;
  },

  async updateTaskProgress(id: number, data: UpdateTaskProgressRequest): Promise<TaskItem> {
    const response = await apiClient.patch<ApiResponse<TaskItem>>(`/api/tasks/${id}/progress`, data);
    return response.data.data;
  },

  async claimTask(id: number): Promise<TaskItem> {
    const response = await apiClient.post<ApiResponse<TaskItem>>(`/api/tasks/${id}/claim`);
    return response.data.data;
  },

  async releaseTask(id: number): Promise<TaskItem> {
    const response = await apiClient.delete<ApiResponse<TaskItem>>(`/api/tasks/${id}/claim`);
    return response.data.data;
  },

  async addCodeRef(id: number, reference: string): Promise<{ task_id: number; code_references: string[] }> {
    const response = await apiClient.post<ApiResponse<{ task_id: number; code_references: string[] }>>(
      `/api/tasks/${id}/code-refs`,
      { reference }
    );
    return response.data.data;
  },
};

