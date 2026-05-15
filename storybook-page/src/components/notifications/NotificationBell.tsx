import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useNotificationStore } from '../../stores/notificationStore';
import { useAuthStore } from '../../stores/authStore';
import { formatRelativeTime } from '../../utils/formatters';
import type { NotificationItem } from '../../types/api';
import { BellIcon } from '../ui/AppIcon';

function notificationLink(item: NotificationItem): string | null {
  switch (item.entity_type) {
    case 'story':
      return `/stories/${item.entity_id}`;
    case 'bug':
      if (item.project_id) {
        return `/projects/${item.project_id}/bugs?bug=${item.entity_id}`;
      }
      return null;
    case 'task':
      // Tasks live under stories; the metadata.story_id is more useful than
      // entity_id alone, but we don't unwrap metadata client-side yet — fall
      // back to the dashboard so the user always lands somewhere relevant.
      return '/dashboard';
    case 'sprint':
      if (item.project_id) {
        return `/projects/${item.project_id}`;
      }
      return null;
    default:
      return null;
  }
}

export default function NotificationBell() {
  const navigate = useNavigate();
  const { isAuthenticated } = useAuthStore();
  const { items, unreadCount, fetchList, fetchUnreadCount, markRead, markAllRead } =
    useNotificationStore();
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement | null>(null);

  // Pull the unread count when the user is authenticated. The WebSocket push
  // keeps it warm afterwards; this initial fetch handles cold loads / hard
  // refreshes where there's no live push yet.
  useEffect(() => {
    if (!isAuthenticated) {
      return;
    }
    void fetchUnreadCount();
  }, [isAuthenticated, fetchUnreadCount]);

  useEffect(() => {
    if (!open) {
      return;
    }
    void fetchList(false);
  }, [open, fetchList]);

  useEffect(() => {
    const handlePointerDown = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', handlePointerDown);
    return () => {
      document.removeEventListener('mousedown', handlePointerDown);
    };
  }, []);

  const handleItemClick = async (item: NotificationItem) => {
    if (!item.read_at) {
      try {
        await markRead(item.id);
      } catch {
        // Optimistic store already reverted; nothing else to do here.
      }
    }
    const link = notificationLink(item);
    setOpen(false);
    if (link) {
      navigate(link);
    }
  };

  if (!isAuthenticated) {
    return null;
  }

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={unreadCount > 0 ? `通知，未读 ${unreadCount} 条` : '通知'}
        onClick={() => setOpen((prev) => !prev)}
        className="relative inline-flex h-9 w-9 items-center justify-center rounded-xl border border-border bg-white text-text-light transition-colors hover:border-primary-200 hover:bg-primary-50 hover:text-primary"
      >
        <BellIcon size={18} />
        {unreadCount > 0 && (
          <span
            aria-hidden
            className="absolute -right-1 -top-1 inline-flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-danger px-1 text-[10px] font-semibold leading-none text-white"
          >
            {unreadCount > 99 ? '99+' : unreadCount}
          </span>
        )}
      </button>

      {open && (
        <div className="surface-card absolute right-0 top-full z-50 mt-1.5 w-80 rounded-[1.25rem] border border-border bg-white shadow-[0_22px_48px_-36px_rgba(16,42,67,0.45)]">
          <div className="flex items-center justify-between border-b border-border px-4 py-2.5">
            <span className="text-sm font-semibold text-text">通知</span>
            <button
              type="button"
              onClick={() => {
                void markAllRead();
              }}
              disabled={unreadCount === 0}
              className="text-xs text-primary transition-colors hover:underline disabled:cursor-not-allowed disabled:text-text-light"
            >
              全部已读
            </button>
          </div>
          <div className="max-h-96 overflow-auto">
            {items.length === 0 ? (
              <div className="px-4 py-6 text-center text-xs text-text-light">暂无通知</div>
            ) : (
              <ul className="divide-y divide-border">
                {items.map((item) => {
                  const unread = !item.read_at;
                  return (
                    <li key={item.id}>
                      <button
                        type="button"
                        onClick={() => {
                          void handleItemClick(item);
                        }}
                        className={`flex w-full items-start gap-2.5 px-4 py-3 text-left transition-colors hover:bg-primary-50 ${
                          unread ? 'bg-primary-50/40' : ''
                        }`}
                      >
                        <span
                          aria-hidden
                          className={`mt-1 inline-block h-2 w-2 shrink-0 rounded-full ${
                            unread ? 'bg-primary' : 'bg-transparent'
                          }`}
                        />
                        <div className="min-w-0 flex-1">
                          <div className="truncate text-sm font-medium text-text">
                            {item.title}
                          </div>
                          {item.body && (
                            <div className="mt-0.5 line-clamp-2 text-xs text-text-light">
                              {item.body}
                            </div>
                          )}
                          <div className="mt-1 text-[10px] uppercase tracking-wide text-text-light">
                            {formatRelativeTime(item.created_at)}
                          </div>
                        </div>
                      </button>
                    </li>
                  );
                })}
              </ul>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
