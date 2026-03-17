import apiClient from './api';
import type {
  ApiResponse,
  LoginRequest,
  RegisterRequest,
  RefreshTokenRequest,
  AuthResponse,
} from '../types/api';

export const authService = {
  /**
   * 用户登录
   */
  async login(data: LoginRequest): Promise<AuthResponse> {
    const response = await apiClient.post<ApiResponse<AuthResponse>>('/api/auth/login', data);
    return response.data.data;
  },

  /**
   * 用户注册
   */
  async register(data: RegisterRequest): Promise<AuthResponse> {
    const response = await apiClient.post<ApiResponse<AuthResponse>>('/api/auth/register', data);
    return response.data.data;
  },

  /**
   * 刷新Token
   */
  async refreshToken(data: RefreshTokenRequest): Promise<{ token: string; expires_at: string }> {
    const response = await apiClient.post<ApiResponse<{ token: string; expires_at: string }>>(
      '/api/auth/refresh',
      data
    );
    return response.data.data;
  },

  /**
   * 退出登录（前端清除Token即可）
   */
  logout(): void {
    localStorage.removeItem('token');
    localStorage.removeItem('refresh_token');
  },
};
