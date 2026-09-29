import { useEffect, useRef, useState, useCallback } from 'react';
import type {
  WSMessage,
  StoryACUpdatedMessage,
  StoryCreatedMessage,
  StoryStatusChangedMessage,
  SprintTerminatedMessage,
  SprintDeletedMessage,
  SprintReorderedMessage,
  ProjectDeletedMessage,
  BugCreatedMessage,
  NotificationNewMessage,
} from '../types/api';
import { authService } from '../services/authService';

interface UseWebSocketOptions {
  onStoryStatusChanged?: (message: StoryStatusChangedMessage) => void;
  onStoryACUpdated?: (message: StoryACUpdatedMessage) => void;
  onStoryCreated?: (message: StoryCreatedMessage) => void;
  onStoryUpdated?: (message: StoryCreatedMessage) => void;
  onSprintTerminated?: (message: SprintTerminatedMessage) => void;
  onSprintDeleted?: (message: SprintDeletedMessage) => void;
  onSprintReordered?: (message: SprintReorderedMessage) => void;
  onProjectDeleted?: (message: ProjectDeletedMessage) => void;
  onBugCreated?: (message: BugCreatedMessage) => void;
  onNotificationNew?: (message: NotificationNewMessage) => void;
}

const isDev = import.meta.env.DEV;

function wsLog(message: string, ...args: unknown[]) {
  // 只在开发环境且非错误时输出
  if (isDev && !message.includes('error') && !message.includes('Error')) {
    console.log(`[WebSocket] ${message}`, ...args);
  }
}

function wsError(message: string, ...args: unknown[]) {
  // 错误总是输出，但添加前缀方便过滤
  console.warn(`[WebSocket] ${message}`, ...args);
}

function parseJwtExpiry(token: string): number | null {
  try {
    const payloadPart = token.split('.')[1];
    if (!payloadPart) {
      return null;
    }
    const normalized = payloadPart.replace(/-/g, '+').replace(/_/g, '/');
    const padded = normalized + '='.repeat((4 - (normalized.length % 4 || 4)) % 4);
    const payload = JSON.parse(atob(padded));
    return typeof payload?.exp === 'number' ? payload.exp : null;
  } catch {
    return null;
  }
}

async function getUsableAccessToken(): Promise<string | null> {
  const token = localStorage.getItem('token');
  if (!token) {
    return null;
  }

  const exp = parseJwtExpiry(token);
  const now = Math.floor(Date.now() / 1000);
  // 提前30秒刷新，避免连接刚建立就因 token 过期被断开。
  if (!exp || exp - now > 30) {
    return token;
  }

  const refreshToken = localStorage.getItem('refresh_token');
  if (!refreshToken) {
    return token;
  }

  try {
    const data = await authService.refreshToken({ refresh_token: refreshToken });
    localStorage.setItem('token', data.token);
    return data.token;
  } catch {
    return token;
  }
}

function buildWsUrl(): string {
  const rawBase = (
    import.meta.env.VITE_API_BASE_URL ||
    (import.meta.env.DEV ? 'http://localhost:8080' : window.location.origin)
  ).trim();
  try {
    const base = new URL(rawBase);
    const basePath = base.pathname.replace(/\/+$/, '');
    const normalizedPath = basePath.endsWith('/api') ? basePath.slice(0, -4) : basePath;
    const wsPath = `${normalizedPath || ''}/ws`.replace(/\/{2,}/g, '/');
    const ws = new URL(`${base.protocol}//${base.host}${wsPath}`);
    ws.protocol = ws.protocol === 'https:' ? 'wss:' : 'ws:';
    return ws.toString();
  } catch {
    return `${rawBase.replace(/\/+$/, '').replace(/\/api$/, '')}/ws`
      .replace('http://', 'ws://')
      .replace('https://', 'wss://');
  }
}

// ===== 模块级共享连接 =====
// 同屏多个组件（Header、看板、故事/缺陷页等）复用一条 WebSocket，
// 订阅者归零后延迟关闭，避免页面切换时的连接抖动。

interface Subscriber {
  /** ref 形式持有 options，消息分发时读取最新回调，避免陈旧闭包 */
  optionsRef: { current: UseWebSocketOptions };
  onStateChange: (state: SharedConnectionState) => void;
}

interface SharedConnectionState {
  isConnected: boolean;
  connectionError: string | null;
}

let sharedState: SharedConnectionState = { isConnected: false, connectionError: null };
let sharedWs: WebSocket | null = null;
const subscribers = new Set<Subscriber>();
let isConnecting = false;
let isManualDisconnect = false;
let reconnectAttempts = 0;
let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
let closeTimer: ReturnType<typeof setTimeout> | undefined;
const maxReconnectAttempts = 20;

function notifyState() {
  subscribers.forEach((subscriber) => subscriber.onStateChange(sharedState));
}

function setState(patch: Partial<SharedConnectionState>) {
  sharedState = { ...sharedState, ...patch };
  notifyState();
}

function dispatchMessage(options: UseWebSocketOptions, message: WSMessage) {
  switch (message.type) {
    case 'story.status_changed':
      options.onStoryStatusChanged?.(message.data as StoryStatusChangedMessage);
      break;
    case 'story.ac_updated':
      options.onStoryACUpdated?.(message.data as StoryACUpdatedMessage);
      break;
    case 'story.ac_added':
    case 'story.ac_edited':
    case 'story.ac_removed':
      // 三类事件与 ac_updated 载荷一致（story_id/ac_id/actor），统一走 AC 同步回调
      options.onStoryACUpdated?.(message.data as StoryACUpdatedMessage);
      break;
    case 'story.created':
      options.onStoryCreated?.(message.data as StoryCreatedMessage);
      break;
    case 'story.updated':
      options.onStoryUpdated?.(message.data as StoryCreatedMessage);
      break;
    case 'sprint.closed':
    case 'sprint.cancelled':
      // close/cancel 载荷一致（sprint_id/project_id/status/actor_id）
      options.onSprintTerminated?.(message.data as SprintTerminatedMessage);
      break;
    case 'sprint.deleted':
      options.onSprintDeleted?.(message.data as SprintDeletedMessage);
      break;
    case 'sprint.reordered':
      options.onSprintReordered?.(message.data as SprintReorderedMessage);
      break;
    case 'project.deleted':
      options.onProjectDeleted?.(message.data as ProjectDeletedMessage);
      break;
    case 'bug.created':
      options.onBugCreated?.(message.data as BugCreatedMessage);
      break;
    case 'notification.new':
      options.onNotificationNew?.(message.data as NotificationNewMessage);
      break;
    default:
      wsLog('Unknown message type:', message.type);
  }
}

function scheduleReconnect() {
  if (reconnectAttempts >= maxReconnectAttempts) {
    setState({ isConnected: false, connectionError: '实时更新连接失败，请刷新页面' });
    wsError('已达到最大重连次数，停止重连');
    return;
  }
  reconnectAttempts += 1;
  const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
  if (reconnectAttempts === 1) {
    wsError(`连接断开，${delay / 1000}秒后重连...`);
  }
  reconnectTimer = setTimeout(() => {
    void ensureConnection();
  }, delay);
}

async function ensureConnection(): Promise<void> {
  if (sharedWs || isConnecting) {
    return;
  }
  if (subscribers.size === 0) {
    return;
  }

  isConnecting = true;
  if (closeTimer) {
    clearTimeout(closeTimer);
    closeTimer = undefined;
  }

  const token = await getUsableAccessToken();
  isConnecting = false;
  if (!token) {
    setState({ isConnected: false, connectionError: '未找到认证Token' });
    return;
  }
  if (subscribers.size === 0 || sharedWs) {
    return;
  }

  const ws = new WebSocket(buildWsUrl(), ['storybook-token', token]);
  isManualDisconnect = false;
  sharedWs = ws;

  ws.onopen = () => {
    if (sharedWs !== ws) {
      return;
    }
    wsLog('connected');
    reconnectAttempts = 0;
    setState({ isConnected: true, connectionError: null });
  };

  ws.onmessage = (event) => {
    if (sharedWs !== ws) {
      return;
    }
    try {
      const message = JSON.parse(event.data) as WSMessage;
      wsLog('message received:', message.type);
      subscribers.forEach((subscriber) => {
        dispatchMessage(subscriber.optionsRef.current, message);
      });
    } catch (error) {
      wsError('Failed to parse message:', error);
    }
  };

  ws.onerror = () => {
    if (sharedWs !== ws) {
      return;
    }
    // 降低错误日志级别，避免刷屏
    if (reconnectAttempts === 0) {
      wsError('连接失败，实时更新功能暂时不可用');
    }
    setState({ connectionError: '连接错误' });
  };

  ws.onclose = () => {
    if (sharedWs !== ws) {
      return;
    }
    sharedWs = null;
    setState({ isConnected: false });
    if (isManualDisconnect) {
      return;
    }
    wsLog('disconnected');
    if (subscribers.size > 0) {
      scheduleReconnect();
    }
  };
}

function forceDisconnect() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = undefined;
  }
  if (sharedWs) {
    isManualDisconnect = true;
    sharedWs.close();
    sharedWs = null;
  }
  setState({ isConnected: false });
}

function releaseConnection() {
  if (subscribers.size > 0) {
    return;
  }
  // 延迟关闭：容忍 StrictMode 双挂载/页面快速切换造成的瞬时零订阅
  if (closeTimer) {
    clearTimeout(closeTimer);
  }
  closeTimer = setTimeout(() => {
    closeTimer = undefined;
    if (subscribers.size === 0) {
      forceDisconnect();
    }
  }, 100);
}

export function useWebSocket(options: UseWebSocketOptions = {}) {
  const [isConnected, setIsConnected] = useState(sharedState.isConnected);
  const [connectionError, setConnectionError] = useState<string | null>(sharedState.connectionError);
  const optionsRef = useRef(options);
  // 渲染后同步最新回调，消息分发时读取，避免陈旧闭包（不在渲染期写 ref）
  useEffect(() => {
    optionsRef.current = options;
  }, [options]);

  useEffect(() => {
    const subscriber: Subscriber = {
      optionsRef,
      onStateChange: (state) => {
        setIsConnected(state.isConnected);
        setConnectionError(state.connectionError);
      },
    };
    subscribers.add(subscriber);
    notifyState();
    void ensureConnection();

    return () => {
      subscribers.delete(subscriber);
      releaseConnection();
    };
  }, []);

  const reconnect = useCallback(() => {
    forceDisconnect();
    reconnectAttempts = 0;
    void ensureConnection();
  }, []);

  const disconnect = useCallback(() => {
    forceDisconnect();
  }, []);

  return {
    isConnected,
    connectionError,
    reconnect,
    disconnect,
  };
}
