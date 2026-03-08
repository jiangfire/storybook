import axios, { AxiosError, InternalAxiosRequestConfig, AxiosResponse } from 'axios';
import { ApiResponse } from '../types/api';

// API 基础URL
const BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

// 创建 Axios 实例
const apiClient = axios.create({
  baseURL: BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器 - 添加 Token
apiClient.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = localStorage.getItem('token');
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error: AxiosError) => {
    return Promise.reject(error);
  }
);

// 响应拦截器 - 统一处理错误和 Token 刷新
apiClient.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
    return response;
  },
  async (error: AxiosError<ApiResponse>) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean };

    // Token 过期，尝试刷新
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;

      try {
        const refreshToken = localStorage.getItem('refresh_token');
        if (refreshToken) {
          const response = await axios.post(`${BASE_URL}/api/auth/refresh`, {
            refresh_token: refreshToken,
          });

          const { token } = response.data.data;
          localStorage.setItem('token', token);

          if (originalRequest.headers) {
            originalRequest.headers.Authorization = `Bearer ${token}`;
          }

          return apiClient(originalRequest);
        }
      } catch (refreshError) {
        // 刷新失败，清除 Token 并跳转登录
        localStorage.removeItem('token');
        localStorage.removeItem('refresh_token');
        window.location.href = '/login';
        return Promise.reject(refreshError);
      }
    }

    // 统一错误处理
    const errorMessage = error.response?.data?.message || '请求失败，请稍后重试';

    // 特殊错误码处理
    switch (error.response?.status) {
      case 401:
        // 未登录，跳转到登录页
        if (window.location.pathname !== '/login') {
          window.location.href = '/login';
        }
        break;
      case 403:
        // 权限不足
        console.error('权限不足');
        break;
      case 404:
        // 资源不存在
        console.error('资源不存在');
        break;
      case 500:
        // 服务器错误
        console.error('服务器错误');
        break;
    }

    return Promise.reject({
      ...error,
      message: errorMessage,
    });
  }
);

export default apiClient;
