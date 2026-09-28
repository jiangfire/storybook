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
  AIStoryChatRequest,
  AIStoryChatResponse,
  AIRefineACData,
  AISummaryData,
  AITranslateData,
  AIDoRCheckData,
  INVESTCheckData,
} from '../types/api';

export const aiService = {
  async generateStory(data: AIGenerateStoryRequest): Promise<AIGeneratedStoryResponse> {
    // 后端 AI 生成超时为 45s，前端全局超时只有 30s；这里单独放宽，避免慢请求前端先报错
    const response = await apiClient.post<ApiResponse<AIGeneratedStoryResponse>>(
      '/api/ai/generate-story',
      data,
      { timeout: 60000 }
    );
    return response.data.data;
  },

  async storyChat(data: AIStoryChatRequest): Promise<AIStoryChatResponse> {
    // 与 generate-story 同样的 45s 后端超时，前端放宽到 60s
    const response = await apiClient.post<ApiResponse<AIStoryChatResponse>>(
      '/api/ai/story-chat',
      data,
      { timeout: 60000 }
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

  /** 根据反馈优化验收标准（返回建议列表，不直接落库） */
  async refineAC(storyId: number, feedback: string): Promise<AIRefineACData> {
    const response = await apiClient.post<ApiResponse<AIRefineACData>>(
      `/api/ai/stories/${storyId}/refine-ac`,
      { feedback },
      { timeout: 60000 }
    );
    return response.data.data;
  },

  /** 干系人摘要 */
  async summarizeStory(storyId: number): Promise<AISummaryData> {
    const response = await apiClient.get<ApiResponse<AISummaryData>>(
      `/api/ai/stories/${storyId}/summary`,
      { timeout: 45000 }
    );
    return response.data.data;
  },

  /** 翻译故事（title/description/AC） */
  async translateStory(storyId: number, language: 'en' | 'zh'): Promise<AITranslateData> {
    const response = await apiClient.post<ApiResponse<AITranslateData>>(
      `/api/ai/stories/${storyId}/translate`,
      { language },
      { timeout: 60000 }
    );
    return response.data.data;
  },

  /** DoR（准备就绪）检查，不依赖 LLM */
  async checkDoR(storyId: number): Promise<AIDoRCheckData> {
    const response = await apiClient.get<ApiResponse<AIDoRCheckData>>(
      `/api/ai/stories/${storyId}/dor-check`
    );
    return response.data.data;
  },
};
