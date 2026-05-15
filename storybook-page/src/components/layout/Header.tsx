import { useEffect, useRef, useState, type KeyboardEvent, type ReactNode } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuthStore } from '../../stores/authStore';
import { useNotificationStore } from '../../stores/notificationStore';
import { getUserInitials } from '../../utils/formatters';
import { searchService } from '../../services/searchService';
import { getErrorMessage } from '../../utils/error';
import { getUserRoleLabel } from '../../utils/roleLabel';
import { useWebSocket } from '../../hooks/useWebSocket';
import { useToast } from '../ui/Toast';
import NotificationBell from '../notifications/NotificationBell';
import type {
  NotificationNewMessage,
  SearchResponseData,
  SemanticStorySearchResponse,
} from '../../types/api';
import { BugIcon, CompassIcon, FolderIcon, SearchIcon, StoryIcon } from '../ui/AppIcon';

type SemanticSearchState = 'idle' | 'available' | 'unavailable' | 'error';

type SemanticCapabilityState = 'enabled' | 'disabled' | 'unknown';

function SemanticCapabilityIcon({ enabled }: { enabled: boolean }) {
  return (
    <svg
      viewBox="0 0 20 20"
      fill="none"
      width={16}
      height={16}
      aria-hidden="true"
      className="shrink-0"
    >
      <circle
        cx="10"
        cy="10"
        r="6.5"
        stroke="currentColor"
        strokeWidth="1.6"
        className={enabled ? 'text-emerald-600' : 'text-slate-300'}
      />
      <circle
        cx="7"
        cy="10"
        r="1"
        fill="currentColor"
        className={enabled ? 'text-emerald-600' : 'text-slate-300'}
      />
      <circle
        cx="13"
        cy="8"
        r="1"
        fill="currentColor"
        className={enabled ? 'text-emerald-600' : 'text-slate-300'}
      />
      <circle
        cx="12"
        cy="13"
        r="1"
        fill="currentColor"
        className={enabled ? 'text-emerald-600' : 'text-slate-300'}
      />
      <path
        d="M7.8 9.6 11.9 8.4 11.2 12"
        stroke="currentColor"
        strokeWidth="1.4"
        strokeLinecap="round"
        strokeLinejoin="round"
        className={enabled ? 'text-emerald-600' : 'text-slate-300'}
      />
      {!enabled && (
        <path
          d="M4.6 15.4 15.4 4.6"
          stroke="currentColor"
          strokeWidth="1.8"
          strokeLinecap="round"
          className="text-slate-400"
        />
      )}
    </svg>
  );
}

function SearchSectionIcon({
  title,
  tone = 'neutral',
  children,
}: {
  title: string;
  tone?: 'neutral' | 'semantic';
  children: ReactNode;
}) {
  return (
    <span
      aria-label={title}
      title={title}
      className={`inline-flex h-6 w-6 items-center justify-center rounded-full ${
        tone === 'semantic' ? 'bg-primary-50 text-primary' : 'bg-secondary-50 text-text-light'
      }`}
    >
      {children}
    </span>
  );
}

function SearchSectionPlaceholder({
  title,
  tone = 'neutral',
  children,
}: {
  title: string;
  tone?: 'neutral' | 'semantic';
  children: ReactNode;
}) {
  return (
    <div className="px-3 py-2">
      <span
        aria-label={title}
        title={title}
        className={`inline-flex h-7 w-7 items-center justify-center rounded-full border ${
          tone === 'semantic'
            ? 'border-primary-100 bg-primary-50/60 text-primary/45'
            : 'border-border bg-secondary-50 text-text-light/45'
        }`}
      >
        {children}
      </span>
    </div>
  );
}

export default function Header() {
  const navigate = useNavigate();
  const { user, logout, isAuthenticated } = useAuthStore();
  const { prepend, fetchUnreadCount } = useNotificationStore();
  const { showInfo } = useToast();
  const [query, setQuery] = useState('');
  const [searching, setSearching] = useState(false);
  const [searchError, setSearchError] = useState('');
  const [searchData, setSearchData] = useState<SearchResponseData | null>(null);
  const [semanticData, setSemanticData] = useState<SemanticStorySearchResponse | null>(null);
  const [semanticState, setSemanticState] = useState<SemanticSearchState>('idle');
  const [semanticError, setSemanticError] = useState('');
  const [semanticCapability, setSemanticCapability] = useState<SemanticCapabilityState>('unknown');
  const [showResult, setShowResult] = useState(false);
  const [isUserMenuOpen, setIsUserMenuOpen] = useState(false);
  const searchContainerRef = useRef<HTMLDivElement | null>(null);
  const userMenuRef = useRef<HTMLDivElement | null>(null);
  const searchRequestIDRef = useRef(0);

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const handleGoBack = () => {
    if (window.history.length > 1) {
      navigate(-1);
      return;
    }
    navigate('/projects');
  };

  useEffect(() => {
    let cancelled = false;

    const loadCapabilities = async () => {
      try {
        const data = await searchService.getCapabilities();
        if (cancelled) {
          return;
        }
        setSemanticCapability(data.semantic_enabled ? 'enabled' : 'disabled');
      } catch {
        if (!cancelled) {
          setSemanticCapability('disabled');
        }
      }
    };

    void loadCapabilities();

    return () => {
      cancelled = true;
    };
  }, []);

  // Subscribe to notification.new from the global WS so the bell badge and
  // toast notifications stay live while the user navigates. Kept in Header
  // (rather than MainLayout) so it follows the always-mounted top-bar lifecycle.
  useWebSocket({
    onNotificationNew: (message: NotificationNewMessage) => {
      prepend(message);
      showInfo(message.title);
    },
  });

  // Re-sync the unread badge on auth state changes (login / logout) so the
  // dot vanishes immediately on logout and reappears on a fresh login.
  useEffect(() => {
    if (!isAuthenticated) {
      return;
    }
    void fetchUnreadCount();
  }, [isAuthenticated, fetchUnreadCount]);

  useEffect(() => {
    const q = query.trim();
    if (!q) {
      searchRequestIDRef.current += 1;
      setSearching(false);
      setSearchError('');
      setSearchData(null);
      setSemanticData(null);
      setSemanticState('idle');
      setSemanticError('');
      setShowResult(false);
      return;
    }

    const requestID = searchRequestIDRef.current + 1;
    searchRequestIDRef.current = requestID;
    const timer = window.setTimeout(async () => {
      try {
        setSearching(true);
        setSearchError('');
        setSemanticError('');
        const [keywordResult, semanticResult] = await Promise.allSettled([
          searchService.search({ q, type: 'all', limit: 8 }),
          searchService.searchSemanticStories(q, 5),
        ]);
        if (searchRequestIDRef.current !== requestID) {
          return;
        }

        if (keywordResult.status === 'fulfilled') {
          setSearchData(keywordResult.value);
        } else {
          setSearchData(null);
          setSearchError(getErrorMessage(keywordResult.reason, '搜索失败'));
        }

        if (semanticResult.status === 'fulfilled') {
          setSemanticData(semanticResult.value);
          setSemanticState('available');
          setSemanticCapability('enabled');
        } else {
          const message = getErrorMessage(semanticResult.reason, '');
          const lowerMessage = message.toLowerCase();
          const disabled =
            message.includes('向量搜索服务未启用') ||
            message.includes('postgres') ||
            message.includes('embedding') ||
            lowerMessage.includes('vector search');
          setSemanticData(null);
          setSemanticState(disabled ? 'unavailable' : 'error');
          setSemanticError(disabled ? '' : message || '语义搜索失败');
          if (disabled) {
            setSemanticCapability('disabled');
          }
        }

        setShowResult(true);
      } catch (err: unknown) {
        if (searchRequestIDRef.current !== requestID) {
          return;
        }
        setSearchData(null);
        setSemanticData(null);
        setSemanticState('error');
        setSemanticError('');
        setShowResult(true);
        setSearchError(getErrorMessage(err, '搜索失败'));
      } finally {
        if (searchRequestIDRef.current === requestID) {
          setSearching(false);
        }
      }
    }, 300);

    return () => {
      window.clearTimeout(timer);
    };
  }, [query]);

  useEffect(() => {
    const handlePointerDown = (event: MouseEvent) => {
      const target = event.target as Node;

      if (searchContainerRef.current && !searchContainerRef.current.contains(target)) {
        setShowResult(false);
      }

      if (userMenuRef.current && !userMenuRef.current.contains(target)) {
        setIsUserMenuOpen(false);
      }
    };

    document.addEventListener('mousedown', handlePointerDown);
    return () => {
      document.removeEventListener('mousedown', handlePointerDown);
    };
  }, []);

  const handleSearchBlur = () => {
    window.setTimeout(() => {
      const container = searchContainerRef.current;
      if (!container) {
        return;
      }
      const activeElement = document.activeElement;
      if (activeElement && !container.contains(activeElement)) {
        setShowResult(false);
      }
    }, 0);
  };

  const handleSearchKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Escape') {
      setShowResult(false);
      event.currentTarget.blur();
    }
  };

  const renderBugLink = (projectID: number, bugID: number) =>
    `/projects/${projectID}/bugs?bug=${bugID}`;

  const roleLabel = getUserRoleLabel(user?.role);

  return (
    <header className="relative z-20 px-2 pt-2 sm:px-3 lg:px-4">
      <div className="mx-auto max-w-[1600px]">
        <div className="section-card rounded-[1.25rem] px-3 py-2.5 sm:px-4">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:gap-4">
            <div className="flex items-center justify-between gap-3 lg:min-w-[280px]">
              <Link to="/projects" className="flex min-w-0 items-center gap-2.5">
                <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-primary text-white">
                  <span className="text-sm font-bold">S</span>
                </div>
                <div className="min-w-0">
                  <h1 className="font-display text-base font-bold text-primary sm:text-lg">Storybook</h1>
                </div>
              </Link>

              <button
                type="button"
                aria-label="返回上一页"
                onClick={handleGoBack}
                className="inline-flex items-center gap-1 rounded-full border border-transparent px-2.5 py-2 text-xs text-text-light transition-colors hover:border-border hover:bg-secondary-50 hover:text-text"
              >
                <span aria-hidden>←</span>
                <span className="hidden sm:inline">返回</span>
              </button>
            </div>

            <div ref={searchContainerRef} className="relative order-3 w-full lg:order-none lg:max-w-xl lg:flex-1">
              <div className="relative">
                <input
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  onFocus={() => {
                    if (query.trim() && (searchData || searchError || searching)) {
                      setShowResult(true);
                      return;
                    }
                    if (query.trim() && (semanticData || semanticError || semanticState === 'available')) {
                      setShowResult(true);
                    }
                  }}
                  onBlur={handleSearchBlur}
                  onKeyDown={handleSearchKeyDown}
                  placeholder="搜索项目 / 故事 / 缺陷"
                  className="w-full rounded-xl border border-border bg-secondary-50 py-2.5 pl-4 pr-16 text-sm text-text outline-none transition focus:border-primary-200 focus:bg-white focus:ring-4 focus:ring-primary/10"
                />
                {semanticCapability !== 'unknown' && (
                  <span
                    aria-label={
                      semanticCapability === 'enabled' ? '语义搜索已启用' : '语义搜索未启用'
                    }
                    title={semanticCapability === 'enabled' ? '语义搜索已启用' : '语义搜索未启用'}
                    className="absolute right-9 top-1/2 inline-flex -translate-y-1/2 items-center justify-center"
                  >
                    <SemanticCapabilityIcon enabled={semanticCapability === 'enabled'} />
                  </span>
                )}
                <span
                  aria-hidden
                  className="absolute right-3 top-1/2 inline-flex -translate-y-1/2 items-center justify-center text-text-light"
                >
                  {searching ? (
                    <span className="h-4 w-4 animate-spin rounded-full border-2 border-border border-t-primary" />
                  ) : (
                    <SearchIcon size={16} />
                  )}
                </span>
              </div>

              {showResult && (
                <div className="surface-card absolute left-0 right-0 top-full z-50 mt-1.5 max-h-96 overflow-auto rounded-[1.25rem] border border-border bg-white p-2.5 shadow-[0_22px_48px_-36px_rgba(16,42,67,0.45)]">
                  {searchError && <div className="text-sm text-danger">{searchError}</div>}
                  {!searchError && !searching && searchData && (
                    <div className="space-y-3">
                      <div>
                        <div className="mb-1">
                          <SearchSectionIcon title="项目结果">
                            <FolderIcon size={12} />
                          </SearchSectionIcon>
                        </div>
                        {searchData.projects && searchData.projects.length > 0 ? (
                          <div className="space-y-1">
                            {searchData.projects.map((item) => (
                              <button
                                type="button"
                                key={`project-${item.id}`}
                                className="w-full rounded-xl px-3 py-2 text-left text-sm transition-colors hover:bg-primary-50"
                                onClick={() => {
                                  setShowResult(false);
                                  navigate(`/projects/${item.id}`);
                                }}
                              >
                                {item.name}
                              </button>
                            ))}
                          </div>
                        ) : (
                          <SearchSectionPlaceholder title="无项目结果">
                            <FolderIcon size={12} />
                          </SearchSectionPlaceholder>
                        )}
                      </div>

                      <div>
                        <div className="mb-1">
                          <SearchSectionIcon title="故事结果">
                            <StoryIcon size={12} />
                          </SearchSectionIcon>
                        </div>
                        {searchData.stories && searchData.stories.length > 0 ? (
                          <div className="space-y-1">
                            {searchData.stories.map((item) => (
                              <button
                                type="button"
                                key={`story-${item.id}`}
                                className="w-full rounded-xl px-3 py-2 text-left text-sm transition-colors hover:bg-primary-50"
                                onClick={() => {
                                  setShowResult(false);
                                  navigate(`/stories/${item.id}`);
                                }}
                              >
                                #{item.id} {item.title}
                              </button>
                            ))}
                          </div>
                        ) : (
                          <SearchSectionPlaceholder title="无故事结果">
                            <StoryIcon size={12} />
                          </SearchSectionPlaceholder>
                        )}
                      </div>

                      {semanticState !== 'idle' && semanticState !== 'unavailable' && (
                        <div>
                          <div className="mb-1">
                            <SearchSectionIcon title="语义搜索结果" tone="semantic">
                              <CompassIcon size={12} />
                            </SearchSectionIcon>
                          </div>
                          {semanticState === 'error' ? (
                            <SearchSectionPlaceholder
                              title={semanticError || '语义搜索暂时不可用'}
                              tone="semantic"
                            >
                              <SemanticCapabilityIcon enabled={false} />
                            </SearchSectionPlaceholder>
                          ) : semanticData && semanticData.stories.length > 0 ? (
                            <div className="space-y-1">
                              {semanticData.stories.map((item) => (
                                <button
                                  type="button"
                                  key={`semantic-story-${item.id}`}
                                  className="w-full rounded-xl px-3 py-2 text-left text-sm transition-colors hover:bg-primary-50"
                                  onClick={() => {
                                    setShowResult(false);
                                    navigate(`/stories/${item.id}`);
                                  }}
                                >
                                  <div className="flex items-start justify-between gap-3">
                                    <span className="min-w-0 truncate">
                                      #{item.id} {item.title}
                                    </span>
                                    <span className="shrink-0 text-xs text-primary">
                                      {(item.similarity * 100).toFixed(0)}%
                                    </span>
                                  </div>
                                </button>
                              ))}
                            </div>
                          ) : (
                            <SearchSectionPlaceholder title="无语义结果" tone="semantic">
                              <CompassIcon size={12} />
                            </SearchSectionPlaceholder>
                          )}
                        </div>
                      )}

                      <div>
                        <div className="mb-1">
                          <SearchSectionIcon title="缺陷结果">
                            <BugIcon size={12} />
                          </SearchSectionIcon>
                        </div>
                        {searchData.bugs && searchData.bugs.length > 0 ? (
                          <div className="space-y-1">
                            {searchData.bugs.map((item) => (
                              <button
                                type="button"
                                key={`bug-${item.id}`}
                                className="w-full rounded-xl px-3 py-2 text-left text-sm transition-colors hover:bg-primary-50"
                                onClick={() => {
                                  setShowResult(false);
                                  navigate(renderBugLink(item.project_id, item.id));
                                }}
                              >
                                #{item.id} {item.title}
                              </button>
                            ))}
                          </div>
                        ) : (
                          <SearchSectionPlaceholder title="无缺陷结果">
                            <BugIcon size={12} />
                          </SearchSectionPlaceholder>
                        )}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>

            <div className="flex items-center justify-end gap-2 sm:gap-2.5 lg:ml-auto">
              <NotificationBell />
              <div ref={userMenuRef} className="relative">
                <button
                  type="button"
                  aria-haspopup="menu"
                  aria-expanded={isUserMenuOpen}
                  onClick={() => setIsUserMenuOpen((prev) => !prev)}
                  className="flex items-center justify-end gap-2 rounded-xl border border-border bg-white px-2 py-1.5 text-right transition-colors hover:border-primary-200 hover:bg-primary-50 sm:px-2.5"
                >
                  <div className="hidden text-right lg:block">
                    <div className="max-w-[180px] truncate text-sm font-medium text-text">
                      {user?.email}
                    </div>
                    <div className="text-[11px] text-text-light">{roleLabel}</div>
                  </div>
                  <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary-100 text-sm font-medium text-primary">
                    {user?.email ? getUserInitials(user.email) : '?'}
                  </div>
                </button>

                {isUserMenuOpen && (
                  <div className="surface-card absolute right-0 top-full z-50 mt-1.5 w-56 rounded-[1.25rem] border border-border bg-white shadow-[0_22px_48px_-36px_rgba(16,42,67,0.45)]">
                    <div className="border-b border-border px-4 py-3 lg:hidden">
                      <div className="truncate text-sm font-medium text-text">{user?.email}</div>
                      <div className="mt-1 text-xs text-text-light">{roleLabel}</div>
                    </div>
                    <div className="py-2">
                      <Link
                        to="/dashboard"
                        onClick={() => setIsUserMenuOpen(false)}
                        className="block px-4 py-2 text-sm text-text transition-colors hover:bg-primary-50"
                      >
                        个人工作台
                      </Link>
                      <button
                        type="button"
                        onClick={handleLogout}
                        className="w-full px-4 py-2 text-left text-sm text-danger transition-colors hover:bg-danger-light"
                      >
                        退出登录
                      </button>
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    </header>
  );
}
