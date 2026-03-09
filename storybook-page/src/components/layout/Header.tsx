import { useEffect, useRef, useState, type KeyboardEvent } from 'react';

import { Link, useNavigate } from 'react-router-dom';
import { useAuthStore } from '../../stores/authStore';
import { getUserInitials } from '../../utils/formatters';
import { searchService } from '../../services/searchService';
import { getErrorMessage } from '../../utils/error';
import type { SearchResponseData } from '../../types/api';

export default function Header() {
  const navigate = useNavigate();
  const { user, logout } = useAuthStore();
  const [query, setQuery] = useState('');
  const [searching, setSearching] = useState(false);
  const [searchError, setSearchError] = useState('');
  const [searchData, setSearchData] = useState<SearchResponseData | null>(null);
  const [showResult, setShowResult] = useState(false);
  const searchContainerRef = useRef<HTMLDivElement | null>(null);
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
      const container = searchContainerRef.current;
      if (!container) {
        return;
      }
      if (!container.contains(event.target as Node)) {
        setShowResult(false);
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

  const renderBugLink = (projectID: number, bugID: number) => `/projects/${projectID}/bugs?bug=${bugID}`;

  return (
    <header className="h-14 bg-white border-b border-border flex items-center justify-between px-4 sticky top-0 z-50 glass">
      <div className="flex items-center gap-2">
        {/* Logo */}
        <Link to="/projects" className="flex items-center space-x-3">
          <div className="w-7 h-7 bg-gradient-to-br from-primary to-accent rounded-md flex items-center justify-center">
            <span className="text-white font-bold text-base">S</span>
          </div>
          <h1 className="font-display text-lg font-bold text-primary">Storybook</h1>
        </Link>
      </div>

      <div ref={searchContainerRef} className="relative flex-1 max-w-lg mx-4">
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
            className="w-full pl-3 pr-8 py-1.5 border border-border rounded-lg text-sm"
          />
          <span
            aria-hidden
            className="absolute right-2 top-1/2 -translate-y-1/2 text-xs text-text-light select-none"
          >
            {searching ? '⏳' : '🔍'}
          </span>
        </div>

        {showResult && (
          <div className="absolute top-full left-0 right-0 mt-2 max-h-96 overflow-auto bg-white border border-border rounded-lg shadow-lg p-3 z-50 space-y-3">
            {searchError && <div className="text-sm text-danger">{searchError}</div>}
            {!searchError && !searching && searchData && (
              <>
                <div>
                  <div className="text-xs text-text-light mb-1">项目</div>
                  {searchData.projects && searchData.projects.length > 0 ? (
                    <div className="space-y-1">
                      {searchData.projects.map((item) => (
                        <button
                          type="button"
                          key={`project-${item.id}`}
                          className="w-full text-left text-sm px-2 py-1 rounded hover:bg-primary-50"
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
                  <div className="text-xs text-text-light mb-1">故事</div>
                  {searchData.stories && searchData.stories.length > 0 ? (
                    <div className="space-y-1">
                      {searchData.stories.map((item) => (
                        <button
                          type="button"
                          key={`story-${item.id}`}
                          className="w-full text-left text-sm px-2 py-1 rounded hover:bg-primary-50"
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
                  <div className="text-xs text-text-light mb-1">缺陷</div>
                  {searchData.bugs && searchData.bugs.length > 0 ? (
                    <div className="space-y-1">
                      {searchData.bugs.map((item) => (
                        <button
                          type="button"
                          key={`bug-${item.id}`}
                          className="w-full text-left text-sm px-2 py-1 rounded hover:bg-primary-50"
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
              </>
            )}
          </div>
        )}
      </div>

      {/* 右侧用户菜单 */}
      <div className="flex items-center gap-3">
        <button
          type="button"
          aria-label="返回上一页"
          onClick={handleGoBack}
          className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-md border border-border text-xs text-text hover:bg-primary-50 transition-colors"
        >
          <span aria-hidden>←</span>
          返回
        </button>

        {/* 用户信息 */}
        <div className="flex items-center space-x-3">
          <div className="text-right">
            <div className="text-sm font-medium text-text">{user?.email}</div>
            <div className="text-xs text-text-light">
              {user?.role === 'product' && '产品经理'}
              {user?.role === 'developer' && '开发人员'}
              {user?.role === 'tester' && '测试人员'}
              {user?.role === 'tech_lead' && '技术负责人'}
              {user?.role === 'admin' && '管理员'}
            </div>
          </div>

          {/* 头像 */}
          <div className="relative group">
            <div className="w-10 h-10 bg-primary-100 text-primary rounded-full flex items-center justify-center font-medium cursor-pointer hover:bg-primary-200 transition-colors">
              {user?.email && getUserInitials(user.email)}
            </div>

            {/* 下拉菜单 */}
            <div className="absolute right-0 top-full mt-2 w-48 bg-white rounded-lg shadow-lg border border-border opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all duration-200">
              <div className="py-2">
                <Link
                  to="/dashboard"
                  className="block px-4 py-2 text-sm text-text hover:bg-primary-50 transition-colors"
                >
                  个人工作台
                </Link>
                <button
                  onClick={handleLogout}
                  className="w-full text-left px-4 py-2 text-sm text-danger hover:bg-danger-light transition-colors"
                >
                  退出登录
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </header>
  );
}
