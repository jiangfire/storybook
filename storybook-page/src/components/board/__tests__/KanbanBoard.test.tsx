import { act, render, screen } from '@testing-library/react';
import type { StoryBoardItem } from '../../../types/models';
import KanbanBoard from '../KanbanBoard';

const fetchBoardData = vi.fn();
const updateStoryStatus = vi.fn();
const showError = vi.fn();

let latestDndContextProps:
  | {
      onDragEnd?: (event: { active: { id: number }; over: { id: string | number } | null }) => unknown;
    }
  | null = null;

let storyStoreState: {
  boardData: Record<string, StoryBoardItem[]>;
  fetchBoardData: typeof fetchBoardData;
  updateStoryStatus: typeof updateStoryStatus;
  isUpdating: boolean;
  isLoading: boolean;
};

vi.mock('@dnd-kit/core', () => ({
  DndContext: ({ children, ...props }: { children: React.ReactNode }) => {
    latestDndContextProps = props;
    return <div data-testid="dnd-context">{children}</div>;
  },
  DragOverlay: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="drag-overlay">{children}</div>
  ),
  PointerSensor: class PointerSensor {},
  useSensor: () => ({}),
  useSensors: () => [],
  closestCorners: () => null,
}));

vi.mock('../../../stores/storyStore', () => ({
  useStoryStore: () => storyStoreState,
}));

vi.mock('../../../hooks/useWebSocket', () => ({
  useWebSocket: () => ({ isConnected: true }),
}));

vi.mock('../../ui/Toast', () => ({
  useToast: () => ({
    showError,
  }),
}));

vi.mock('../BoardColumn', () => ({
  default: ({
    id,
    title,
    count,
    stories,
  }: {
    id: string;
    title: string;
    count: number;
    stories: StoryBoardItem[];
  }) => (
    <section data-testid={`column-${id}`}>
      <h3>{title}</h3>
      <span>{count}</span>
      {stories.map((story) => (
        <div key={story.id}>{story.title}</div>
      ))}
    </section>
  ),
}));

function createStory(id: number, status: StoryBoardItem['status'], position: number): StoryBoardItem {
  return {
    id,
    title: `Story ${id}`,
    story_type: 'feature',
    status,
    priority: 2,
    position,
  };
}

function setStoryStore(boardData: Record<string, StoryBoardItem[]>) {
  storyStoreState = {
    boardData,
    fetchBoardData,
    updateStoryStatus,
    isUpdating: false,
    isLoading: false,
  };
}

describe('KanbanBoard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    latestDndContextProps = null;
    setStoryStore({
      pending: [createStory(1, 'pending', 1024)],
      backlog: [createStory(2, 'backlog', 1024)],
      ready: [],
      in_progress: [],
      test: [],
      done: [],
    });
  });

  it('渲染待审批列并在挂载时拉取看板数据', () => {
    render(<KanbanBoard projectId={7} />);

    expect(fetchBoardData).toHaveBeenCalledWith(7);
    expect(screen.getByTestId('column-pending')).toHaveTextContent('待审批');
    expect(screen.getByText('Story 1')).toBeInTheDocument();
  });

  it('拖拽到或从 pending 列时阻止更新并提示错误', async () => {
    render(<KanbanBoard projectId={7} />);

    await act(async () => {
      await latestDndContextProps?.onDragEnd?.({
        active: { id: 2 },
        over: { id: 'pending' },
      });
    });

    expect(showError).toHaveBeenCalledWith('待审批故事需通过评审流程流转');
    expect(updateStoryStatus).not.toHaveBeenCalled();
  });

  it('普通跨列拖拽会更新状态', async () => {
    render(<KanbanBoard projectId={7} />);

    await act(async () => {
      await latestDndContextProps?.onDragEnd?.({
        active: { id: 2 },
        over: { id: 'ready' },
      });
    });

    expect(updateStoryStatus).toHaveBeenCalledWith(2, {
      status: 'ready',
      position: 0,
    });
  });

  it('同列重排会带上新的 position', async () => {
    setStoryStore({
      pending: [],
      backlog: [createStory(2, 'backlog', 1024), createStory(3, 'backlog', 2048)],
      ready: [],
      in_progress: [],
      test: [],
      done: [],
    });

    render(<KanbanBoard projectId={7} />);

    await act(async () => {
      await latestDndContextProps?.onDragEnd?.({
        active: { id: 2 },
        over: { id: 3 },
      });
    });

    expect(updateStoryStatus).toHaveBeenCalledWith(2, {
      status: 'backlog',
      position: 3072,
    });
  });
});
