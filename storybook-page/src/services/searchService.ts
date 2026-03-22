import apiClient from './api';
import type { ApiResponse, SearchParams, SearchResponseData } from '../types/api';

export const searchService = {
  async search(params: SearchParams): Promise<SearchResponseData> {
    const response = await apiClient.get<ApiResponse<SearchResponseData>>('/api/search', {
      params,
    });
    return response.data.data;
  },
};
