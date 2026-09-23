import { create } from 'zustand';
import { techLeadService } from '../services/techLeadService';

interface PendingReviewState {
  /** 待审批故事总数（技术负责人/管理员侧边栏角标） */
  total: number;
  isLoading: boolean;
  fetch: () => Promise<void>;
}

/**
 * 待审批数量是全局壳层状态：登录后拉一次，收到审批相关实时通知后由
 * Header 的 WebSocket 处理器触发刷新，保证角标不需要轮询也能保持新鲜。
 */
export const usePendingReviewStore = create<PendingReviewState>((set) => ({
  total: 0,
  isLoading: false,
  fetch: async () => {
    set({ isLoading: true });
    try {
      const data = await techLeadService.getPendingStories({ page: 1, limit: 1 });
      set({ total: data.total ?? 0, isLoading: false });
    } catch {
      // 角标获取失败时静默归零，不打扰用户
      set({ total: 0, isLoading: false });
    }
  },
}));
