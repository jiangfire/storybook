import { useEffect, useRef, useCallback, useState } from 'react';
import { WSMessage, StoryACUpdatedMessage, StoryStatusChangedMessage } from '../types/api';
import { authService } from '../services/authService';

interface UseWebSocketOptions {
  onStoryStatusChanged?: (message: StoryStatusChangedMessage) => void;
  onStoryACUpdated?: (message: StoryACUpdatedMessage) => void;
  onStoryCreated?: (message: unknown) => void;
  onStoryUpdated?: (message: unknown) => void;
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

function buildWsUrl(token: string): string {
  const rawBase = (import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080').trim();
  try {
    const base = new URL(rawBase);
    const basePath = base.pathname.replace(/\/+$/, '');
    const normalizedPath = basePath.endsWith('/api') ? basePath.slice(0, -4) : basePath;
    const wsPath = `${normalizedPath || ''}/ws`.replace(/\/{2,}/g, '/');
    const ws = new URL(`${base.protocol}//${base.host}${wsPath}`);
    ws.protocol = ws.protocol === 'https:' ? 'wss:' : 'ws:';
    ws.searchParams.set('token', token);
    return ws.toString();
  } catch {
    return `${rawBase.replace(/\/+$/, '').replace(/\/api$/, '')}/ws?token=${encodeURIComponent(token)}`
      .replace('http://', 'ws://')
      .replace('https://', 'wss://');
  }
}

export function useWebSocket(options: UseWebSocketOptions = {}) {
  const [isConnected, setIsConnected] = useState(false);
  const [connectionError, setConnectionError] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout>>();
  const reconnectAttemptsRef = useRef(0);
  const isManualDisconnectRef = useRef(false);
  const isUnmountedRef = useRef(false);
  const isConnectingRef = useRef(false);
  const connectRef = useRef<() => Promise<void>>(async () => {});
  const maxReconnectAttempts = 20;

  const connect = useCallback(async () => {
    if (isUnmountedRef.current || wsRef.current || isConnectingRef.current) {
      return;
    }
    isConnectingRef.current = true;

    const token = await getUsableAccessToken();
    if (isUnmountedRef.current) {
      isConnectingRef.current = false;
      return;
    }
    if (!token) {
      setConnectionError('未找到认证Token');
      setIsConnected(false);
      isConnectingRef.current = false;
      return;
    }

    const ws = new WebSocket(buildWsUrl(token));
    isManualDisconnectRef.current = false;
    wsRef.current = ws;
    isConnectingRef.current = false;

    ws.onopen = () => {
      if (isUnmountedRef.current || wsRef.current !== ws) {
        return;
      }
      wsLog('connected');
      setIsConnected(true);
      setConnectionError(null);
      reconnectAttemptsRef.current = 0;
    };

    ws.onmessage = (event) => {
      try {
        const message: WSMessage = JSON.parse(event.data) as WSMessage;
        wsLog('message received:', message.type);

        // 根据消息类型分发
          switch (message.type) {
          case 'story.status_changed':
            options.onStoryStatusChanged?.(message.data as StoryStatusChangedMessage);
            break;
          case 'story.ac_updated':
            options.onStoryACUpdated?.(message.data as StoryACUpdatedMessage);
            break;
          case 'story.created':
            options.onStoryCreated?.(message.data);
            break;
          case 'story.updated':
            options.onStoryUpdated?.(message.data);
            break;
          default:
            wsLog('Unknown message type:', message.type);
        }
      } catch (error) {
        wsError('Failed to parse message:', error);
      }
    };

    ws.onerror = (_event) => {
      if (isUnmountedRef.current || wsRef.current !== ws) {
        return;
      }
      // 降低错误日志级别，避免刷屏
      if (reconnectAttemptsRef.current === 0) {
        wsError('连接失败，实时更新功能暂时不可用');
      }
      setConnectionError('连接错误');
    };

    ws.onclose = () => {
      if (isUnmountedRef.current || wsRef.current !== ws) {
        return;
      }
      if (isManualDisconnectRef.current) {
        wsRef.current = null;
        return;
      }
      wsLog('disconnected');
      setIsConnected(false);
      wsRef.current = null;

      // 尝试重连
      if (reconnectAttemptsRef.current < maxReconnectAttempts) {
        reconnectAttemptsRef.current += 1;
        const delay = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000);

        if (reconnectAttemptsRef.current === 1) {
          wsError(`连接断开，${delay / 1000}秒后重连...`);
        }

        reconnectTimeoutRef.current = setTimeout(() => {
          void connectRef.current();
        }, delay);
      } else {
        setConnectionError('实时更新连接失败，请刷新页面');
        wsError('已达到最大重连次数，停止重连');
      }
    };
  }, [options]);

  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
    }
    if (wsRef.current) {
      isManualDisconnectRef.current = true;
      wsRef.current.close();
      wsRef.current = null;
    }
    setIsConnected(false);
  }, []);

  const reconnect = useCallback(() => {
    disconnect();
    reconnectAttemptsRef.current = 0;
    void connectRef.current();
  }, [disconnect]);

  useEffect(() => {
    isUnmountedRef.current = false;
    const connectTimer = setTimeout(() => {
      void connectRef.current();
    }, 0);

    return () => {
      clearTimeout(connectTimer);
      isUnmountedRef.current = true;
      disconnect();
    };
  }, [disconnect]);

  return {
    isConnected,
    connectionError,
    reconnect,
    disconnect,
  };
}
