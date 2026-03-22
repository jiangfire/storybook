import apiClient from './api';
import type {
  AIGenerateStoryRequest,
  AIGeneratedStoryResponse,
  AISplitStoryRequest,
  AISplitStoryData,
  AIConfigResponse,
  AIConfigTestResponse,
  AIConfigUpdateRequest,
  ApiResponse,
  INVESTCheckData,
} from '../types/api';

export const aiService = {
  async generateStory(data: AIGenerateStoryRequest): Promise<AIGeneratedStoryResponse> {
    const response = await apiClient.post<ApiResponse<AIGeneratedStoryResponse>>(
      '/api/ai/generate-story',
      data
    );
    return response.data.data;
  },

  async getConfig(): Promise<AIConfigResponse> {
    const response = await apiClient.get<ApiResponse<AIConfigResponse>>('/api/admin/ai/config');
    return response.data.data;
  },

  async updateConfig(data: AIConfigUpdateRequest): Promise<AIConfigResponse> {
    const response = await apiClient.put<ApiResponse<AIConfigResponse>>(
      '/api/admin/ai/config',
      data
    );
    return response.data.data;
  },

  async testConfig(data?: AIConfigUpdateRequest): Promise<AIConfigTestResponse> {
    const response = await apiClient.post<ApiResponse<AIConfigTestResponse>>(
      '/api/admin/ai/config/test',
      data ?? {}
    );
    return response.data.data;
  },

  async splitStory(storyId: number, data?: AISplitStoryRequest): Promise<AISplitStoryData> {
    const response = await apiClient.post<ApiResponse<AISplitStoryData>>(
      `/api/ai/stories/${storyId}/split`,
      data
    );
    return response.data.data;
  },

  async checkInvest(storyId: number): Promise<INVESTCheckData> {
    const response = await apiClient.get<ApiResponse<INVESTCheckData>>(
      `/api/ai/stories/${storyId}/invest-check`
    );
    return response.data.data;
  },
};
