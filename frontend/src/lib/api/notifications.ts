import { notificationApi } from './client';

export interface Notification {
  notification_id: string;
  user_id: string;
  type: string;
  title: string;
  body: string; // Changed from message to body
  link?: string;
  is_read: boolean; // Changed from read to is_read
  metadata?: Record<string, string>;
  created_at: string;
}

export const notificationService = {
  async getNotifications(userId: string, page = 1, limit = 20, unreadOnly = false): Promise<{
    notifications: Notification[];
    total: number;
    unread_count: number;
    page: number;
    limit: number;
  }> {
    const response = await notificationApi.get('/', {
      params: { user_id: userId, page, limit, unread_only: unreadOnly },
    });
    return response.data;
  },

  async markAsRead(notificationId: string, userId: string): Promise<{ message: string }> {
    const response = await notificationApi.put(`/${notificationId}/read`, {
      notification_id: notificationId,
      user_id: userId,
    });
    return response.data;
  },

  async markAllAsRead(userId: string): Promise<{ message: string; count: number }> {
    const response = await notificationApi.put('/read-all', {
      user_id: userId,
    });
    return response.data;
  },

  async deleteNotification(notificationId: string, userId: string): Promise<{ message: string }> {
    const response = await notificationApi.delete(`/${notificationId}`, {
      params: { user_id: userId },
    });
    return response.data;
  },

  async getUnreadCount(userId: string): Promise<{ count: number }> {
    const response = await notificationApi.get('/unread-count', {
      params: { user_id: userId },
    });
    return response.data;
  },
};

