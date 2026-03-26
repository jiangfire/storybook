import apiClient from './api';
import type {
  ApiResponse,
  SearchCapabilitiesResponse,
  SearchParams,
  SearchResponseData,
  SemanticStorySearchResponse,
} from '../types/api';

export const searchService = {
  async search(params: SearchParams): Promise<SearchResponseData> {
    const response = await apiClient.get<ApiResponse<SearchResponseData>>('/api/search', {
      params,
    });
    return response.data.data;
  },

  async searchSemanticStories(q: string, limit = 5): Promise<SemanticStorySearchResponse> {
    const response = await apiClient.get<ApiResponse<SemanticStorySearchResponse>>(
      '/api/search/semantic',
      {
        params: { q, limit },
      }
    );
    return response.data.data;
  },

  async getCapabilities(): Promise<SearchCapabilitiesResponse> {
    const response = await apiClient.get<ApiResponse<SearchCapabilitiesResponse>>(
      '/api/search/capabilities'
    );
    return response.data.data;
  },
};
