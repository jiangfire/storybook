import apiClient from './api';
import type {
  AIGenerateStoryRequest,
  AIGeneratedStoryResponse,
  AISplitStoryRequest,
  AISplitStoryData,
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

