import { act, render } from '@testing-library/react';

// 模块级单例状态会跨用例泄漏，每个用例重置模块并重新导入 hook。
const wsInstances: FakeWebSocket[] = [];

class FakeWebSocket {
  url: string;
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  closed = false;

  constructor(url: string) {
    this.url = url;
    wsInstances.push(this);
  }

  close() {
    if (this.closed) {
      return;
    }
    this.closed = true;
    this.onclose?.();
  }

  simulateOpen() {
    this.onopen?.();
  }

  simulateMessage(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) });
  }
}

// setup.ts 的 beforeAll(server.listen) 会 patch 全局 WebSocket（MSW 拦截），
// 这里在其后执行 stub 覆盖，保证 hook 拿到的是本文件的 Fake。
beforeAll(() => {
  vi.stubGlobal('WebSocket', FakeWebSocket);
});

afterAll(() => {
  vi.unstubAllGlobals();
});

function makeJwt(exp: number): string {
  const encode = (value: unknown) =>
    btoa(JSON.stringify(value)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
  return `${encode({ alg: 'HS256', typ: 'JWT' })}.${encode({ exp })}.sig`;
}

function fakeMessage(type: string, data: unknown) {
  return { type, data, timestamp: new Date().toISOString() };
}

function Subscriber({ handlers }: { handlers?: Record<string, () => void> }) {
  const { useWebSocket } = hooksModule!;
  useWebSocket(handlers && handlers.onStoryCreated ? { onStoryCreated: handlers.onStoryCreated } : {});
  return <div>subscriber</div>;
}

let hooksModule: typeof import('../useWebSocket') | null = null;

async function loadFreshModule() {
  vi.resetModules();
  hooksModule = await import('../useWebSocket');
}

describe('useWebSocket 单例共享', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllTimers();
    wsInstances.length = 0;
    localStorage.clear();
    localStorage.setItem('token', makeJwt(Math.floor(Date.now() / 1000) + 3600));
  });

  afterEach(() => {
    act(() => {
      vi.runOnlyPendingTimers();
    });
    vi.useRealTimers();
  });

  it('多个组件共享同一条连接', async () => {
    await loadFreshModule();

    render(
      <div>
        <Subscriber />
        <Subscriber />
        <Subscriber />
      </div>
    );
    await act(async () => {
      await vi.runOnlyPendingTimersAsync();
    });

    expect(wsInstances).toHaveLength(1);
  });

  it('消息分发给所有订阅者', async () => {
    await loadFreshModule();
    const first = vi.fn();
    const second = vi.fn();

    render(
      <div>
        <Subscriber handlers={{ onStoryCreated: first }} />
        <Subscriber handlers={{ onStoryCreated: second }} />
      </div>
    );
    await act(async () => {
      await vi.runOnlyPendingTimersAsync();
    });

    act(() => {
      wsInstances[0].simulateMessage(
        fakeMessage('story.created', { story_id: 1, project_id: 7, title: 'T', status: 'pending' })
      );
    });

    expect(first).toHaveBeenCalledTimes(1);
    expect(second).toHaveBeenCalledTimes(1);
  });

  it('订阅者归零后延迟关闭连接，重新挂载不再复用旧连接', async () => {
    await loadFreshModule();

    const { unmount } = render(<Subscriber />);
    await act(async () => {
      await vi.runOnlyPendingTimersAsync();
    });
    expect(wsInstances).toHaveLength(1);

    act(() => {
      unmount();
    });
    // 延迟关闭窗口内连接仍在
    act(() => {
      vi.advanceTimersByTime(50);
    });
    expect(wsInstances[0].closed).toBe(false);

    act(() => {
      vi.advanceTimersByTime(100);
    });
    expect(wsInstances[0].closed).toBe(true);

    // 重新挂载会建立新连接
    render(<Subscriber />);
    await act(async () => {
      await vi.runOnlyPendingTimersAsync();
    });
    expect(wsInstances).toHaveLength(2);
    expect(wsInstances[1].closed).toBe(false);
  });

  it('分发 sprint.deleted/reordered 与 project.deleted 事件', async () => {
    await loadFreshModule();
    const onSprintDeleted = vi.fn();
    const onSprintReordered = vi.fn();
    const onProjectDeleted = vi.fn();

    function MultiSubscriber() {
      const { useWebSocket } = hooksModule!;
      useWebSocket({ onSprintDeleted, onSprintReordered, onProjectDeleted });
      return <div>multi</div>;
    }

    render(<MultiSubscriber />);
    await act(async () => {
      await vi.runOnlyPendingTimersAsync();
    });

    act(() => {
      wsInstances[0].simulateMessage(
        fakeMessage('sprint.deleted', { sprint_id: 3, project_id: 7, deleted_by: 1 })
      );
      wsInstances[0].simulateMessage(
        fakeMessage('sprint.reordered', { sprint_id: 3, project_id: 7, orders: [], actor_id: 1 })
      );
      wsInstances[0].simulateMessage(
        fakeMessage('project.deleted', { project_id: 7, deleted_by: 1 })
      );
    });

    expect(onSprintDeleted).toHaveBeenCalledWith(
      expect.objectContaining({ sprint_id: 3, project_id: 7 })
    );
    expect(onSprintReordered).toHaveBeenCalledWith(expect.objectContaining({ sprint_id: 3 }));
    expect(onProjectDeleted).toHaveBeenCalledWith(expect.objectContaining({ project_id: 7 }));
  });
});
