import { useEffect, useRef, useState, type KeyboardEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuthStore } from '../../stores/authStore';
import { getUserInitials } from '../../utils/formatters';
import { searchService } from '../../services/searchService';
import { getErrorMessage } from '../../utils/error';
import type { SearchResponseData } from '../../types/api';
import { SearchIcon } from '../ui/AppIcon';

export default function Header() {
  const navigate = useNavigate();
  const { user, logout } = useAuthStore();
  const [query, setQuery] = useState('');
  const [searching, setSearching] = useState(false);
  const [searchError, setSearchError] = useState('');
  const [searchData, setSearchData] = useState<SearchResponseData | null>(null);
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
    const q = query.trim();
    if (!q) {
      searchRequestIDRef.current += 1;
      setSearching(false);
      setSearchError('');
      setSearchData(null);
      setShowResult(false);
      return;
    }

    const requestID = searchRequestIDRef.current + 1;
    searchRequestIDRef.current = requestID;
    const timer = window.setTimeout(async () => {
      try {
        setSearching(true);
        setSearchError('');
        const data = await searchService.search({ q, type: 'all', limit: 8 });
        if (searchRequestIDRef.current !== requestID) {
          return;
        }
        setSearchData(data);
        setShowResult(true);
      } catch (err: unknown) {
        if (searchRequestIDRef.current !== requestID) {
          return;
        }
        setSearchData(null);
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

  const roleLabel =
    (user?.role === 'product' && '产品经理') ||
    (user?.role === 'developer' && '开发人员') ||
    (user?.role === 'tester' && '测试人员') ||
    (user?.role === 'tech_lead' && '技术负责人') ||
    (user?.role === 'admin' && '管理员') ||
    '协作成员';

  return (
    <header className="relative z-20 px-3 pt-3 sm:px-4 lg:px-6">
      <div className="mx-auto max-w-[1600px]">
        <div className="surface-card rounded-[1rem] px-3 py-2.5 sm:px-4">
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
                    }
                  }}
                  onBlur={handleSearchBlur}
                  onKeyDown={handleSearchKeyDown}
                  placeholder="搜索项目 / 故事 / 缺陷"
                  className="w-full rounded-xl border border-border bg-secondary-50 py-2.5 pl-4 pr-10 text-sm text-text outline-none transition focus:border-primary-200 focus:bg-white focus:ring-4 focus:ring-primary/10"
                />
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
                        <div className="mb-1 text-xs font-medium text-text-light">项目</div>
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
                          <div className="text-xs text-text-light">无项目结果</div>
                        )}
                      </div>

                      <div>
                        <div className="mb-1 text-xs font-medium text-text-light">故事</div>
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
                          <div className="text-xs text-text-light">无故事结果</div>
                        )}
                      </div>

                      <div>
                        <div className="mb-1 text-xs font-medium text-text-light">缺陷</div>
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
                          <div className="text-xs text-text-light">无缺陷结果</div>
                        )}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>

            <div className="flex items-center justify-end gap-2 sm:gap-2.5">
              <div ref={userMenuRef} className="relative">
                <button
                  type="button"
                  aria-haspopup="menu"
                  aria-expanded={isUserMenuOpen}
                  onClick={() => setIsUserMenuOpen((prev) => !prev)}
                  className="flex items-center gap-2 rounded-xl border border-border bg-white px-2 py-1.5 transition-colors hover:border-primary-200 hover:bg-primary-50 sm:px-2.5"
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
