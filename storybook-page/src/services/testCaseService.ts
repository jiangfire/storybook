import apiClient from './api';
import type {
  ApiResponse,
  CreateTestCaseRequest,
  TestCaseItem,
  TestCaseListResponse,
  UpdateACStatusRequest,
} from '../types/api';

export const testCaseService = {
  async getStoryTestCases(storyId: number): Promise<TestCaseListResponse> {
    const response = await apiClient.get<ApiResponse<TestCaseListResponse>>(
      `/api/stories/${storyId}/test-cases`
    );
    return response.data.data;
  },

  async createTestCase(storyId: number, data: CreateTestCaseRequest): Promise<TestCaseItem> {
    const response = await apiClient.post<ApiResponse<TestCaseItem>>(
      `/api/stories/${storyId}/test-cases`,
      data
    );
    return response.data.data;
  },

  async updateTestCaseStatus(
    id: number,
    data: Pick<UpdateACStatusRequest, 'status'>
  ): Promise<TestCaseItem> {
    const response = await apiClient.patch<ApiResponse<TestCaseItem>>(
      `/api/test-cases/${id}/status`,
      data
    );
    return response.data.data;
  },
};
