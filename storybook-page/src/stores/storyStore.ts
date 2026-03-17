import { create } from 'zustand';
import type { Story, StoryBoardItem, Activity, ACStatus } from '../types/models';
import type { ActivityListParams, BoardData, UpdateStoryStatusRequest } from '../types/api';
import { storyService } from '../services/storyService';
import { getErrorMessage } from '../utils/error';

interface StoryState {
  // 看板数据
  boardData: Record<string, StoryBoardItem[]>;
  // 当前故事
  currentStory: Story | null;
  // 活动历史
  activities: Activity[];
  // 状态
  isLoading: boolean;
  isUpdating: boolean;
  error: string | null;

  // Actions
  fetchBoardData: (projectId: number) => Promise<void>;
  fetchStory: (storyId: number) => Promise<void>;
  fetchActivities: (storyId: number, params?: ActivityListParams) => Promise<void>;
  updateStoryStatus: (storyId: number, data: UpdateStoryStatusRequest) => Promise<void>;
  claimStory: (storyId: number) => Promise<void>;
  releaseStory: (storyId: number) => Promise<void>;
  updateACStatus: (
    storyId: number,
    acId: string,
    status: ACStatus,
    evidence?: string
  ) => Promise<void>;
  clearCurrentStory: () => void;
  clearError: () => void;
}

const emptyBoardData: Record<string, StoryBoardItem[]> = {
  backlog: [],
  ready: [],
  in_progress: [],
  test: [],
  done: [],
};

function normalizeBoardColumns(columns?: BoardData['columns']): Record<string, StoryBoardItem[]> {
  const normalized: Record<string, StoryBoardItem[]> = {
    ...emptyBoardData,
  };

  for (const col of columns || []) {
    const status = col?.status;
    if (!status || !Array.isArray(col?.stories)) {
      continue;
    }

    normalized[status] = col.stories.map((story) => ({
      ...story,
      assigned_to: story.assigned_to || story.assignee,
    }));
  }

  return normalized;
}

function moveStoryToStatus(
  boardData: Record<string, StoryBoardItem[]>,
  storyId: number,
  toStatus: string,
  position?: number
): Record<string, StoryBoardItem[]> {
  const next: Record<string, StoryBoardItem[]> = {};
  let targetStory: StoryBoardItem | null = null;
  let fromStatus = '';

  for (const [status, stories] of Object.entries(boardData)) {
    const found = stories.find((story) => story.id === storyId);
    if (found) {
      targetStory = found;
      fromStatus = status;
    }
    next[status] = stories.filter((story) => story.id !== storyId);
  }

  if (!targetStory || !next[toStatus]) {
    return boardData;
  }

  if (fromStatus === toStatus) {
    if (position === undefined || Number.isNaN(position)) {
      return boardData;
    }
    return patchStoryInBoard(boardData, storyId, { position });
  }

  next[toStatus] = [
    ...next[toStatus],
    {
      ...targetStory,
      status: toStatus as StoryBoardItem['status'],
      ...(position !== undefined ? { position } : {}),
    },
  ];
  return next;
}

function patchStoryInBoard(
  boardData: Record<string, StoryBoardItem[]>,
  storyId: number,
  patch: Partial<StoryBoardItem>
): Record<string, StoryBoardItem[]> {
  const next: Record<string, StoryBoardItem[]> = {};
  for (const [status, stories] of Object.entries(boardData)) {
    next[status] = stories.map((story) => (story.id === storyId ? { ...story, ...patch } : story));
  }
  return next;
}

export const useStoryStore = create<StoryState>((set, get) => ({
  boardData: emptyBoardData,
  currentStory: null,
  activities: [],
  isLoading: false,
  isUpdating: false,
  error: null,

  fetchBoardData: async (projectId) => {
    set({ isLoading: true, error: null });
    try {
      const response = await storyService.getBoardData(projectId);
      set({ boardData: normalizeBoardColumns(response.columns), isLoading: false });
    } catch (error: unknown) {
      set({ error: getErrorMessage(error), isLoading: false });
    }
  },

  fetchStory: async (storyId) => {
    set({ isLoading: true, error: null });
    try {
      const story = await storyService.getStory(storyId);
      set({ currentStory: story, isLoading: false });
    } catch (error: unknown) {
      set({ error: getErrorMessage(error), isLoading: false });
    }
  },

  fetchActivities: async (storyId, params) => {
    try {
      const response = await storyService.getActivities(storyId, params);
      set({ activities: response.activities });
    } catch (error: unknown) {
      set({ error: getErrorMessage(error) });
    }
  },

  updateStoryStatus: async (storyId, data) => {
    set({ isUpdating: true, error: null });
    const previous = get().boardData;
    const optimistic = moveStoryToStatus(previous, storyId, data.status, data.position);
    set({ boardData: optimistic });

    try {
      await storyService.updateStoryStatus(storyId, data);
      set({ isUpdating: false });
    } catch (error: unknown) {
      set({ boardData: previous, error: getErrorMessage(error), isUpdating: false });
      throw error;
    }
  },

  claimStory: async (storyId) => {
    set({ isUpdating: true, error: null });
    try {
      const updatedStory = await storyService.claimStory(storyId);
      const { boardData, currentStory } = get();
      const withAssignee = patchStoryInBoard(boardData, storyId, {
        assigned_to: updatedStory.assigned_to,
      });
      const moved = updatedStory.status
        ? moveStoryToStatus(withAssignee, storyId, updatedStory.status)
        : withAssignee;

      set({
        boardData: moved,
        currentStory:
          currentStory && currentStory.id === storyId
            ? {
                ...currentStory,
                assigned_to: updatedStory.assigned_to,
                status: (updatedStory.status || currentStory.status) as Story['status'],
              }
            : currentStory,
        isUpdating: false,
      });
    } catch (error: unknown) {
      set({ error: getErrorMessage(error), isUpdating: false });
      throw error;
    }
  },

  releaseStory: async (storyId) => {
    set({ isUpdating: true, error: null });
    try {
      const updatedStory = await storyService.releaseStory(storyId);
      const { boardData, currentStory } = get();
      const withoutAssignee = patchStoryInBoard(boardData, storyId, { assigned_to: undefined });
      const moved = updatedStory.status
        ? moveStoryToStatus(withoutAssignee, storyId, updatedStory.status)
        : withoutAssignee;

      set({
        boardData: moved,
        currentStory:
          currentStory && currentStory.id === storyId
            ? {
                ...currentStory,
                assigned_to: undefined,
                status: (updatedStory.status || currentStory.status) as Story['status'],
              }
            : currentStory,
        isUpdating: false,
      });
    } catch (error: unknown) {
      set({ error: getErrorMessage(error), isUpdating: false });
      throw error;
    }
  },

  updateACStatus: async (storyId, acId, status, evidence) => {
    try {
      await storyService.updateACStatus(storyId, acId, { status, evidence });

      // 更新当前故事的AC状态
      const { currentStory } = get();
      if (currentStory && currentStory.id === storyId) {
        const updatedACs = currentStory.acceptance_criteria.map((ac) =>
          ac.id === acId ? { ...ac, status, evidence } : ac
        );
        set({
          currentStory: { ...currentStory, acceptance_criteria: updatedACs },
        });
      }
    } catch (error: unknown) {
      set({ error: getErrorMessage(error) });
      throw error;
    }
  },

  clearCurrentStory: () => {
    set({ currentStory: null, activities: [] });
  },

  clearError: () => set({ error: null }),
}));
