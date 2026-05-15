import { create } from 'zustand';
import type { NotificationItem, NotificationNewMessage } from '../types/api';
import { notificationService } from '../services/notificationService';
import { getErrorMessage } from '../utils/error';

interface NotificationState {
  items: NotificationItem[];
  unreadCount: number;
  isLoading: boolean;
  error: string | null;

  fetchList: (unreadOnly?: boolean) => Promise<void>;
  fetchUnreadCount: () => Promise<void>;
  markRead: (id: number) => Promise<void>;
  markAllRead: () => Promise<void>;
  prepend: (message: NotificationNewMessage) => void;
  clearError: () => void;
}

function toItem(message: NotificationNewMessage): NotificationItem {
  return {
    id: message.id,
    type: message.type,
    entity_type: message.entity_type,
    entity_id: message.entity_id,
    project_id: message.project_id ?? null,
    actor_id: message.actor_id ?? null,
    title: message.title,
    body: message.body,
    read_at: null,
    created_at: message.created_at,
  };
}

export const useNotificationStore = create<NotificationState>((set, get) => ({
  items: [],
  unreadCount: 0,
  isLoading: false,
  error: null,

  fetchList: async (unreadOnly = false) => {
    set({ isLoading: true, error: null });
    try {
      const data = await notificationService.list({
        unread_only: unreadOnly,
        page: 1,
        limit: 20,
      });
      set({ items: data.items, isLoading: false });
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, '加载通知失败'), isLoading: false });
    }
  },

  fetchUnreadCount: async () => {
    try {
      const data = await notificationService.unreadCount();
      set({ unreadCount: data.count });
    } catch (error: unknown) {
      // Silent failure — a missing badge is preferable to a thrown error in
      // the global layout.
      set({ error: getErrorMessage(error, '获取未读数量失败') });
    }
  },

  markRead: async (id) => {
    const before = get();
    const target = before.items.find((item) => item.id === id);
    if (!target || target.read_at) {
      return;
    }
    // Optimistic update; revert on failure.
    set({
      items: before.items.map((item) =>
        item.id === id ? { ...item, read_at: new Date().toISOString() } : item
      ),
      unreadCount: Math.max(0, before.unreadCount - 1),
    });
    try {
      await notificationService.markRead(id);
    } catch (error: unknown) {
      set({ items: before.items, unreadCount: before.unreadCount, error: getErrorMessage(error) });
      throw error;
    }
  },

  markAllRead: async () => {
    const before = get();
    if (before.unreadCount === 0) {
      return;
    }
    const now = new Date().toISOString();
    set({
      items: before.items.map((item) => (item.read_at ? item : { ...item, read_at: now })),
      unreadCount: 0,
    });
    try {
      await notificationService.markAllRead();
    } catch (error: unknown) {
      set({ items: before.items, unreadCount: before.unreadCount, error: getErrorMessage(error) });
      throw error;
    }
  },

  prepend: (message) => {
    const { items, unreadCount } = get();
    if (items.some((item) => item.id === message.id)) {
      return;
    }
    set({ items: [toItem(message), ...items].slice(0, 50), unreadCount: unreadCount + 1 });
  },

  clearError: () => set({ error: null }),
}));
