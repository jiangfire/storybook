import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  DndContext,
  DragEndEvent,
  DragOverlay,
  DragStartEvent,
  PointerSensor,
  useSensor,
  useSensors,
  closestCorners,
} from '@dnd-kit/core';
import { arrayMove } from '@dnd-kit/sortable';
import { useStoryStore } from '../../stores/storyStore';
import { useWebSocket } from '../../hooks/useWebSocket';
import BoardColumn from './BoardColumn';
import { StoryBoardItem } from '../../types/models';
import { StoryStatusChangedMessage } from '../../types/api';
import { useToast } from '../ui/Toast';

const COLUMNS = [
  { id: 'backlog', title: '待办', icon: '📋' },
  { id: 'ready', title: '就绪', icon: '✅' },
  { id: 'in_progress', title: '开发中', icon: '🔨' },
  { id: 'test', title: '测试中', icon: '🔍' },
  { id: 'done', title: '已完成', icon: '✨' },
];

interface KanbanBoardProps {
  projectId: number;
}

export default function KanbanBoard({ projectId }: KanbanBoardProps) {
  const { boardData, fetchBoardData, updateStoryStatus, isUpdating, isLoading } = useStoryStore();
  const [activeId, setActiveId] = useState<number | null>(null);
  const [localBoardData, setLocalBoardData] = useState(boardData);
  const { showError } = useToast();

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8,
      },
    })
  );

  useEffect(() => {
    fetchBoardData(projectId);
  }, [projectId, fetchBoardData]);

  // 同步 boardData 到本地状态
  useEffect(() => {
    setLocalBoardData(boardData);
  }, [boardData]);

  // 处理 WebSocket 实时更新
  const handleStoryStatusChanged = useCallback((message: StoryStatusChangedMessage) => {
    setLocalBoardData((prev: typeof boardData) => {
      const newBoardData = { ...prev };

      // 从旧状态列中移除故事
      Object.keys(newBoardData).forEach((status) => {
        newBoardData[status] = newBoardData[status].filter((s) => s.id !== message.story_id);
      });

      // 找到故事并更新状态
      const story = Object.values(prev)
        .flat()
        .find((s) => s.id === message.story_id);
      if (story) {
        const updatedStory = { ...story, status: message.new_status };
        // 添加到新状态列
        if (newBoardData[message.new_status]) {
          newBoardData[message.new_status].push(updatedStory);
        }
      }

      return newBoardData;
    });
  }, []);

  const findStatusByStoryId = (data: Record<string, StoryBoardItem[]>, storyId: number) => {
    for (const [status, stories] of Object.entries(data)) {
      if (stories.some((item) => item.id === storyId)) {
        return status;
      }
    }
    return null;
  };

  const resolveDropStatus = (overId: string, data: Record<string, StoryBoardItem[]>) => {
    if (COLUMNS.some((column) => column.id === overId)) {
      return overId;
    }

    const overStoryId = Number(overId);
    if (!Number.isNaN(overStoryId)) {
      return findStatusByStoryId(data, overStoryId);
    }

    return null;
  };

  const handleDragStart = (event: DragStartEvent) => {
    setActiveId(event.active.id as number);
  };

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event;
    setActiveId(null);

    if (!over) return;

    const activeId = active.id as number;
    const overId = over.id as string;

    const activeStory = Object.values(localBoardData)
      .flat()
      .find((s) => s.id === activeId);

    if (!activeStory) return;
    const activeStatus = findStatusByStoryId(localBoardData, activeId);
    if (!activeStatus) return;

    const newStatus = resolveDropStatus(overId, localBoardData);
    if (!newStatus) {
      return;
    }

    if (activeStory.status === newStatus) {
      const overStoryId = Number(overId);
      if (Number.isNaN(overStoryId) || overStoryId === activeId) {
        return;
      }

      const stories = localBoardData[activeStatus] || [];
      const oldIndex = stories.findIndex((story) => story.id === activeId);
      const newIndex = stories.findIndex((story) => story.id === overStoryId);
      if (oldIndex < 0 || newIndex < 0 || oldIndex === newIndex) {
        return;
      }

      const reorderedRaw = arrayMove(stories, oldIndex, newIndex);
      const reordered = reorderedRaw.map((story, index) => ({ ...story, position: index + 1 }));

      const prevStory = reorderedRaw[newIndex - 1];
      const nextStory = reorderedRaw[newIndex + 1];
      const movedStory = reorderedRaw[newIndex];
      const prevPos = prevStory?.position;
      const nextPos = nextStory?.position;

      let targetPosition = movedStory?.position || newIndex + 1;
      if (typeof prevPos === 'number' && typeof nextPos === 'number') {
        targetPosition = (prevPos + nextPos) / 2;
      } else if (typeof prevPos === 'number') {
        targetPosition = prevPos + 1024;
      } else if (typeof nextPos === 'number') {
        targetPosition = nextPos - 1024;
      }

      setLocalBoardData((prev) => {
        return { ...prev, [activeStatus]: reordered };
      });

      try {
        await updateStoryStatus(activeId, {
          status: activeStatus,
          position: targetPosition,
        });
      } catch {
        showError('排序更新失败，已回滚');
      }
      return;
    }

    try {
      await updateStoryStatus(activeId, {
        status: newStatus,
        position: 0,
      });
    } catch {
      showError('更新状态失败，已回滚');
    }
  };

  // 获取当前拖拽的故事
  const activeStory = activeId
    ? Object.values(localBoardData)
        .flat()
        .find((s) => s.id === activeId)
    : null;

  // WebSocket 实时更新
  const wsOptions = useMemo(
    () => ({
      onStoryStatusChanged: handleStoryStatusChanged,
    }),
    [handleStoryStatusChanged]
  );
  const { isConnected: wsConnected } = useWebSocket(wsOptions);

  if (isLoading) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center text-text-light">
          <div className="animate-spin rounded-full h-10 w-10 border-b-2 border-primary mx-auto mb-3" />
          <p>加载看板中...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full">
      <DndContext
        sensors={sensors}
        collisionDetection={closestCorners}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
      >
        {/* 看板列 */}
        <div className="flex gap-4 h-full overflow-x-auto pb-4 snap-x snap-mandatory">
          {COLUMNS.map((column) => (
            <BoardColumn
              key={column.id}
              id={column.id}
              stories={localBoardData[column.id] || []}
              title={column.title}
              icon={column.icon}
              count={localBoardData[column.id]?.length || 0}
            />
          ))}
        </div>

        {/* 拖拽预览 */}
        <DragOverlay>
          {activeStory && (
            <div className="w-[280px] opacity-50">
              <div className="bg-white rounded-lg shadow-lg border-2 border-primary p-4">
                <div className="font-medium text-text mb-2">{activeStory.title}</div>
                <div className="text-sm text-text-light">拖拽到新列...</div>
              </div>
            </div>
          )}
        </DragOverlay>
      </DndContext>

      {/* 加载指示器 */}
      {isUpdating && (
        <div className="fixed top-4 right-4 bg-primary text-white px-4 py-2 rounded-lg shadow-lg flex items-center space-x-2">
          <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
          <span className="text-sm">更新中...</span>
        </div>
      )}

      {!wsConnected && (
        <div className="fixed bottom-4 right-4 bg-warning-light text-text px-3 py-2 rounded-lg text-xs border border-warning">
          实时连接已断开
        </div>
      )}
    </div>
  );
}
