import apiClient from './api';
import type {
  ApiResponse,
  SearchCapabilitiesResponse,
  SearchParams,
  SearchResponseData,
  SemanticStorySearchResponse,
  SimilarStoriesRequest,
  SimilarStoriesResponse,
  SuggestTagsRequest,
  SuggestTagsResponse,
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

  /**
   * 相似故事查重（依赖 pgvector，未启用时接口报错，调用方需降级处理）
   */
  async findSimilarStories(
    data: SimilarStoriesRequest
  ): Promise<SimilarStoriesResponse['similar_stories']> {
    const response = await apiClient.post<ApiResponse<SimilarStoriesResponse>>(
      '/api/stories/similar',
      data
    );
    return response.data.data.similar_stories;
  },

  /**
   * 标签建议（依赖 pgvector，未启用时接口报错，调用方需降级处理）
   */
  async suggestTags(data: SuggestTagsRequest): Promise<string[]> {
    const response = await apiClient.post<ApiResponse<SuggestTagsResponse>>('/api/tags/suggest', data);
    return response.data.data.tags;
  },
};
