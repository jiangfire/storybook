import apiClient from './api';
import type {
  ApiResponse,
  AssignBugRequest,
  BugItem,
  BugListParams,
  BugListResponse,
  CreateBugRequest,
  UpdateBugStatusRequest,
} from '../types/api';

export const bugService = {
  async getProjectBugs(projectId: number, params?: BugListParams): Promise<BugListResponse> {
    const response = await apiClient.get<ApiResponse<BugListResponse>>(
      `/api/projects/${projectId}/bugs`,
      {
        params,
      }
    );
    return response.data.data;
  },

  async createBug(projectId: number, data: CreateBugRequest): Promise<BugItem> {
    const response = await apiClient.post<ApiResponse<BugItem>>(
      `/api/projects/${projectId}/bugs`,
      data
    );
    return response.data.data;
  },

  async getBug(id: number): Promise<BugItem> {
    const response = await apiClient.get<ApiResponse<BugItem>>(`/api/bugs/${id}`);
    return response.data.data;
  },

  async updateBugStatus(id: number, data: UpdateBugStatusRequest): Promise<BugItem> {
    const response = await apiClient.patch<ApiResponse<BugItem>>(`/api/bugs/${id}/status`, data);
    return response.data.data;
  },

  async assignBug(id: number, data: AssignBugRequest): Promise<BugItem> {
    const response = await apiClient.patch<ApiResponse<BugItem>>(`/api/bugs/${id}/assign`, data);
    return response.data.data;
  },
};
