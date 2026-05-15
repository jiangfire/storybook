import apiClient from './api';
import type {
  ApiResponse,
  NotificationListParams,
  NotificationListResponse,
  NotificationMarkAllReadResponse,
  NotificationUnreadCountResponse,
} from '../types/api';

export const notificationService = {
  async list(params?: NotificationListParams): Promise<NotificationListResponse> {
    const response = await apiClient.get<ApiResponse<NotificationListResponse>>(
      '/api/notifications',
      { params }
    );
    return response.data.data;
  },

  async unreadCount(): Promise<NotificationUnreadCountResponse> {
    const response = await apiClient.get<ApiResponse<NotificationUnreadCountResponse>>(
      '/api/notifications/unread-count'
    );
    return response.data.data;
  },

  async markRead(id: number): Promise<void> {
    await apiClient.post<ApiResponse<{ id: number }>>(`/api/notifications/${id}/read`);
  },

  async markAllRead(): Promise<NotificationMarkAllReadResponse> {
    const response = await apiClient.post<ApiResponse<NotificationMarkAllReadResponse>>(
      '/api/notifications/mark-all-read'
    );
    return response.data.data;
  },
};
