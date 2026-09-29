import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { searchService } from '../../services/searchService';
import { useToast } from '../../components/ui/Toast';
import { getErrorMessage } from '../../utils/error';
import { PageContainer, PageHero } from '../../components/page/PageLayout';
import Button from '../../components/ui/Button';
import { formatStoryStatus, formatBugStatus } from '../../utils/formatters';
import type { SearchAssigneeCandidate, SearchResponseData } from '../../types/api';

const SEARCH_TYPES = [
  { value: 'all', label: '全部' },
  { value: 'project', label: '项目' },
  { value: 'story', label: '故事' },
  { value: 'bug', label: '缺陷' },
] as const;

const STORY_STATUSES = ['pending', 'backlog', 'ready', 'in_progress', 'test', 'done'];

const STATUS_LABELS: Record<string, string> = {
  pending: '待审批',
  backlog: '待办',
  ready: '就绪',
  in_progress: '进行中',
  test: '测试中',
  done: '已完成',
  open: '打开',
  resolved: '已解决',
  closed: '已关闭',
};

export default function SearchResultPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();
  const { showError } = useToast();

  const q = searchParams.get('q') ?? '';
  const type = (searchParams.get('type') ?? 'all') as (typeof SEARCH_TYPES)[number]['value'];
  const createdFrom = searchParams.get('created_from') ?? '';
  const createdTo = searchParams.get('created_to') ?? '';
  const assigneeParam = searchParams.get('assignee') ?? '';
  const statuses = searchParams.getAll('status');

  const [input, setInput] = useState(q);
  const [data, setData] = useState<SearchResponseData | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [assigneeCandidates, setAssigneeCandidates] = useState<SearchAssigneeCandidate[]>([]);

  useEffect(() => {
    setInput(q);
  }, [q]);

  // 负责人候选加载失败不阻塞搜索，只是下拉为空
  useEffect(() => {
    let cancelled = false;
    searchService
      .listAssignees()
      .then((users) => {
        if (!cancelled) setAssigneeCandidates(users);
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, []);

  const runSearch = useCallback(async () => {
    if (!q.trim()) {
      setData(null);
      return;
    }
    try {
      setIsLoading(true);
      const result = await searchService.search({
        q: q.trim(),
        type,
        limit: 50,
        created_from: createdFrom || undefined,
        created_to: createdTo || undefined,
        status: statuses.length > 0 ? statuses : undefined,
        assignee: assigneeParam ? Number(assigneeParam) : undefined,
      });
      setData(result);
    } catch (error: unknown) {
      showError(getErrorMessage(error, '搜索失败'));
      setData(null);
    } finally {
      setIsLoading(false);
    }
    // statuses 是数组引用，用 join 稳定依赖
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [q, type, createdFrom, createdTo, assigneeParam, statuses.join(',')]);

  useEffect(() => {
    void runSearch();
  }, [runSearch]);

  const updateParam = (key: string, value: string | null) => {
    const next = new URLSearchParams(searchParams);
    if (value === null || value === '') {
      next.delete(key);
    } else {
      next.set(key, value);
    }
    setSearchParams(next, { replace: true });
  };

  const toggleStatus = (status: string) => {
    const next = new URLSearchParams(searchParams);
    const current = next.getAll('status');
    next.delete('status');
    const updated = current.includes(status)
      ? current.filter((s) => s !== status)
      : [...current, status];
    updated.forEach((s) => next.append('status', s));
    setSearchParams(next, { replace: true });
  };

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    const trimmed = input.trim();
    if (!trimmed) {
      showError('请输入搜索关键词');
      return;
    }
    updateParam('q', trimmed);
  };

  const storyCount = data?.stories?.length ?? 0;
  const projectCount = data?.projects?.length ?? 0;
  const bugCount = data?.bugs?.length ?? 0;
  const total = storyCount + projectCount + bugCount;

  return (
    <PageContainer>
      <PageHero>
        <div className="space-y-4 px-5 py-5 sm:px-6 lg:px-7 lg:py-6">
          <div className="space-y-1">
            <h1 className="text-xl font-semibold text-text">搜索结果</h1>
            <p className="text-sm text-text-light">
              {q
                ? isLoading
                  ? '搜索中...'
                  : `「${q}」共 ${total} 条结果（项目 ${projectCount} · 故事 ${storyCount} · 缺陷 ${bugCount}）`
                : '输入关键词搜索项目、故事与缺陷。'}
            </p>
          </div>

          <form onSubmit={handleSubmit} className="flex gap-2">
            <input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder="搜索项目 / 故事 / 缺陷"
              className="flex-1 rounded-xl border border-border bg-secondary-50 px-4 py-2.5 text-sm text-text outline-none transition focus:border-primary-200 focus:bg-white focus:ring-4 focus:ring-primary/10"
            />
            <Button type="submit" isLoading={isLoading}>
              搜索
            </Button>
          </form>

          <div className="flex flex-wrap items-center gap-3 text-sm">
            <div className="flex items-center gap-1">
              {SEARCH_TYPES.map((option) => (
                <button
                  key={option.value}
                  type="button"
                  onClick={() => updateParam('type', option.value === 'all' ? null : option.value)}
                  className={
                    type === option.value
                      ? 'rounded-full bg-primary-50 px-3 py-1 text-primary'
                      : 'rounded-full px-3 py-1 text-text-light transition-colors hover:bg-secondary-100'
                  }
                >
                  {option.label}
                </button>
              ))}
            </div>
            <div className="flex items-center gap-1.5 text-text-light">
              <span>创建于</span>
              <input
                type="date"
                value={createdFrom}
                onChange={(e) => updateParam('created_from', e.target.value)}
                className="rounded-lg border border-border px-2 py-1 text-sm"
                aria-label="创建开始日期"
              />
              <span>至</span>
              <input
                type="date"
                value={createdTo}
                onChange={(e) => updateParam('created_to', e.target.value)}
                className="rounded-lg border border-border px-2 py-1 text-sm"
                aria-label="创建结束日期"
              />
            </div>
            <div className="flex items-center gap-1.5 text-text-light">
              <span>负责人</span>
              <select
                value={assigneeParam}
                onChange={(e) => updateParam('assignee', e.target.value)}
                className="max-w-48 rounded-lg border border-border bg-white px-2 py-1 text-sm text-text"
                aria-label="负责人"
              >
                <option value="">全部</option>
                {assigneeCandidates.map((u) => (
                  <option key={u.id} value={String(u.id)}>
                    {u.email}
                  </option>
                ))}
                {assigneeParam &&
                  !assigneeCandidates.some((u) => String(u.id) === assigneeParam) && (
                    <option value={assigneeParam}>用户 #{assigneeParam}</option>
                  )}
              </select>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-1.5 text-sm">
            <span className="text-text-light">状态：</span>
            {STORY_STATUSES.map((status) => (
              <button
                key={status}
                type="button"
                onClick={() => toggleStatus(status)}
                className={
                  statuses.includes(status)
                    ? 'rounded-full bg-primary-50 px-2.5 py-0.5 text-xs text-primary'
                    : 'rounded-full border border-border px-2.5 py-0.5 text-xs text-text-light transition-colors hover:border-primary-200'
                }
              >
                {STATUS_LABELS[status]}
              </button>
            ))}
          </div>
        </div>
      </PageHero>

      <div className="space-y-4">
        {data && (type === 'all' || type === 'project') && (
          <section className="section-card rounded-[1.8rem] p-4 sm:p-5">
            <h2 className="mb-3 text-base font-semibold text-text">项目（{projectCount}）</h2>
            {projectCount === 0 ? (
              <div className="state-panel state-panel-empty">无匹配项目</div>
            ) : (
              <div className="space-y-2">
                {data.projects?.map((item) => (
                  <Link
                    key={item.id}
                    to={`/projects/${item.id}`}
                    className="section-block block rounded-[1.2rem] px-4 py-3 transition-colors hover:border-primary-200"
                  >
                    <div className="font-medium text-text">{item.name}</div>
                    {item.description && (
                      <div className="mt-1 line-clamp-2 text-sm text-text-light">
                        {item.description}
                      </div>
                    )}
                  </Link>
                ))}
              </div>
            )}
          </section>
        )}

        {data && (type === 'all' || type === 'story') && (
          <section className="section-card rounded-[1.8rem] p-4 sm:p-5">
            <h2 className="mb-3 text-base font-semibold text-text">故事（{storyCount}）</h2>
            {storyCount === 0 ? (
              <div className="state-panel state-panel-empty">无匹配故事</div>
            ) : (
              <div className="space-y-2">
                {data.stories?.map((item) => (
                  <Link
                    key={item.id}
                    to={`/stories/${item.id}`}
                    className="section-block block rounded-[1.2rem] px-4 py-3 transition-colors hover:border-primary-200"
                  >
                    <div className="flex items-center justify-between gap-3">
                      <span className="min-w-0 truncate font-medium text-text">
                        #{item.id} {item.title}
                      </span>
                      <span className="shrink-0 rounded-full bg-secondary-100 px-2 py-0.5 text-xs text-text-light">
                        {formatStoryStatus(item.status)}
                      </span>
                    </div>
                  </Link>
                ))}
              </div>
            )}
          </section>
        )}

        {data && (type === 'all' || type === 'bug') && (
          <section className="section-card rounded-[1.8rem] p-4 sm:p-5">
            <h2 className="mb-3 text-base font-semibold text-text">缺陷（{bugCount}）</h2>
            {bugCount === 0 ? (
              <div className="state-panel state-panel-empty">无匹配缺陷</div>
            ) : (
              <div className="space-y-2">
                {data.bugs?.map((item) => (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => navigate(`/projects/${item.project_id}/bugs?bug=${item.id}`)}
                    className="section-block block w-full rounded-[1.2rem] px-4 py-3 text-left transition-colors hover:border-primary-200"
                  >
                    <div className="flex items-center justify-between gap-3">
                      <span className="min-w-0 truncate font-medium text-text">
                        #{item.id} {item.title}
                      </span>
                      <span className="shrink-0 rounded-full bg-secondary-100 px-2 py-0.5 text-xs text-text-light">
                        {formatBugStatus(item.status)}
                      </span>
                    </div>
                  </button>
                ))}
              </div>
            )}
          </section>
        )}

        {!q && <div className="state-panel state-panel-empty">请输入关键词开始搜索</div>}
      </div>
    </PageContainer>
  );
}
