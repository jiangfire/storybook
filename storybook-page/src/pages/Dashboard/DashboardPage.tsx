import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { useAuthStore } from '../../stores/authStore';
import { useProjectStore } from '../../stores/projectStore';
import apiClient from '../../services/api';
import {
  formatStoryStatus,
  formatStoryType,
  getStoryTypeColor,
  formatPriority,
  getPriorityColor,
} from '../../utils/formatters';
import type { ApiResponse, DashboardData } from '../../types/api';
import { resolveQuickStartProject } from './quickStart';
import {
  BoardIcon,
  ChartIcon,
  CheckCircleIcon,
  CrownIcon,
  FolderIcon,
  InboxIcon,
  SparklesIcon,
  StoryIcon,
  UsersIcon,
  WrenchIcon,
} from '../../components/ui/AppIcon';

type DashboardStoryItem = DashboardData['my_stories']['assigned'][number];

function StoryListSection({
  title,
  count,
  badgeClass,
  emptyIcon,
  emptyTitle,
  emptyAction,
  items,
}: {
  title: string;
  count: number;
  badgeClass: string;
  emptyIcon: ReactNode;
  emptyTitle: string;
  emptyAction?: ReactNode;
  items: DashboardStoryItem[];
}) {
  return (
    <section className="surface-card rounded-[1.7rem] p-5 sm:p-6">
      <div className="mb-6 flex items-center justify-between gap-3">
        <h2 className="text-lg font-semibold text-text sm:text-xl">{title}</h2>
        <span className={`rounded-full px-3 py-1 text-sm font-medium ${badgeClass}`}>{count}</span>
      </div>

      {items.length === 0 ? (
        <div className="py-8 text-center text-text-light">
          <div className="mb-2 inline-flex h-14 w-14 items-center justify-center rounded-full bg-secondary-50 text-text-light">
            {emptyIcon}
          </div>
          <p>{emptyTitle}</p>
          {emptyAction}
        </div>
      ) : (
        <div className="space-y-3">
          {items.map((story) => (
            <Link
              key={story.id}
              to={`/stories/${story.id}`}
              className="card-hover block rounded-2xl border border-border bg-white p-4 transition-all hover:border-primary-200"
            >
              <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                <div className="flex-1">
                  <div className="mb-2 flex flex-wrap items-center gap-2">
                    <span
                      className={`rounded-full px-2 py-1 text-xs font-medium ${getStoryTypeColor(story.story_type || 'feature')}`}
                    >
                      {formatStoryType(story.story_type || 'feature')}
                    </span>
                    <span
                      className={`rounded-full px-2 py-1 text-xs font-medium ${getPriorityColor(story.priority)}`}
                    >
                      {formatPriority(story.priority)}
                    </span>
                  </div>
                  <h3 className="mb-1 font-medium text-text">{story.title}</h3>
                  <p className="break-words text-sm text-text-light">{story.project}</p>
                </div>
                <div className="sm:ml-4">
                  <span className="inline-flex rounded-lg bg-secondary-100 px-3 py-1 text-sm text-text">
                    {formatStoryStatus(story.status)}
                  </span>
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}
    </section>
  );
}

export default function DashboardPage() {
  const { user } = useAuthStore();
  const { projects, fetchProjects } = useProjectStore();
  const [dashboardData, setDashboardData] = useState<DashboardData | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const [selectedQuickProjectId, setSelectedQuickProjectId] = useState<number | null>(null);
  const canCreateStory = user?.role === 'product' || user?.role === 'admin';

  const quickStartProject = useMemo(
    () => resolveQuickStartProject(projects, selectedQuickProjectId),
    [projects, selectedQuickProjectId]
  );
  const roleLabel =
    (user?.role === 'product' && '产品经理') ||
    (user?.role === 'developer' && '开发人员') ||
    (user?.role === 'tester' && '测试人员') ||
    (user?.role === 'tech_lead' && '技术负责人') ||
    (user?.role === 'admin' && '管理员') ||
    '协作成员';

  useEffect(() => {
    if (!projects || projects.length === 0) {
      if (selectedQuickProjectId !== null) {
        setSelectedQuickProjectId(null);
      }
      return;
    }

    if (
      selectedQuickProjectId === null ||
      !projects.some((project) => project.id === selectedQuickProjectId)
    ) {
      setSelectedQuickProjectId(projects[0].id);
    }
  }, [projects, selectedQuickProjectId]);

  useEffect(() => {
    loadDashboardData();
    fetchProjects();
  }, [fetchProjects]);

  const loadDashboardData = async () => {
    setIsLoading(true);
    setLoadError('');
    try {
      const response = await apiClient.get<ApiResponse<DashboardData>>('/api/me/dashboard');
      setDashboardData(response.data.data);
    } catch {
      setLoadError('工作台数据加载失败，请稍后重试');
      setDashboardData(null);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="mx-auto max-w-7xl space-y-6 px-4 py-6 sm:px-6 lg:px-8">
      <section className="surface-card overflow-hidden rounded-[2rem]">
        <div className="grid gap-6 px-5 py-6 sm:px-6 lg:grid-cols-[minmax(0,1fr)_320px] lg:px-8 lg:py-8">
          <div className="space-y-5">
            <div className="space-y-3">
              <div className="inline-flex items-center gap-2 rounded-full bg-primary-50 px-3 py-1 text-xs font-medium text-primary">
                <ChartIcon size={14} />
                个人工作台
              </div>
              <div>
                <h1 className="text-3xl font-bold tracking-tight text-text sm:text-4xl">
                  欢迎回来，{user?.email?.split('@')[0]}
                </h1>
                <p className="mt-2 max-w-2xl text-sm leading-6 text-text-light sm:text-base">
                  今天先从最重要的事情开始。这里汇总了你的当前交付、进展状态和快速入口。
                </p>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
              <div className="rounded-2xl border border-border bg-white p-4">
                <div className="text-xs font-medium text-text-light">当前角色</div>
                <div className="mt-2 text-lg font-semibold text-text">{roleLabel}</div>
              </div>
              <div className="rounded-2xl border border-border bg-white p-4">
                <div className="text-xs font-medium text-text-light">可见项目</div>
                <div className="mt-2 text-lg font-semibold text-text">{projects.length}</div>
              </div>
              <div className="rounded-2xl border border-border bg-white p-4">
                <div className="text-xs font-medium text-text-light">我的创建权限</div>
                <div className="mt-2 text-lg font-semibold text-text">
                  {canCreateStory ? '可创建故事' : '浏览与协作'}
                </div>
              </div>
            </div>
          </div>

          <div className="rounded-[1.6rem] bg-gradient-to-br from-primary to-primary-700 p-5 text-white shadow-md">
            <div className="text-xs font-medium text-white/70">今日重点</div>
            <div className="mt-3 text-3xl font-semibold">
              {dashboardData?.statistics.in_progress || 0}
            </div>
            <p className="mt-1 text-sm text-white/80">个故事正在推进中</p>
            <div className="mt-5 space-y-2 text-sm text-white/85">
              <div className="flex items-center justify-between rounded-xl bg-white/10 px-3 py-2">
                <span>总领取</span>
                <span className="font-semibold">{dashboardData?.statistics.total_assigned || 0}</span>
              </div>
              <div className="flex items-center justify-between rounded-xl bg-white/10 px-3 py-2">
                <span>已完成</span>
                <span className="font-semibold">{dashboardData?.statistics.completed || 0}</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      {isLoading ? (
        <div className="state-panel state-panel-loading py-12">工作台加载中...</div>
      ) : loadError ? (
        <div className="state-panel state-panel-error">{loadError}</div>
      ) : (
        <div className="space-y-8">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3 sm:gap-6">
            <div className="surface-card card-hover rounded-[1.7rem] p-5 sm:p-6">
              <div className="mb-4 flex items-center justify-between">
                <h3 className="text-base font-semibold text-text sm:text-lg">总领取</h3>
                <div className="flex h-11 w-11 items-center justify-center rounded-lg bg-primary-100 text-primary sm:h-12 sm:w-12">
                  <StoryIcon size={22} />
                </div>
              </div>
              <p className="mb-1 text-3xl font-bold text-primary sm:mb-2 sm:text-4xl">
                {dashboardData?.statistics.total_assigned || 0}
              </p>
              <p className="text-sm text-text-light">个故事</p>
            </div>

            <div className="surface-card card-hover rounded-[1.7rem] p-5 sm:p-6">
              <div className="mb-4 flex items-center justify-between">
                <h3 className="text-base font-semibold text-text sm:text-lg">进行中</h3>
                <div className="flex h-11 w-11 items-center justify-center rounded-lg bg-warning-light text-warning sm:h-12 sm:w-12">
                  <WrenchIcon size={22} />
                </div>
              </div>
              <p className="mb-1 text-3xl font-bold text-warning sm:mb-2 sm:text-4xl">
                {dashboardData?.statistics.in_progress || 0}
              </p>
              <p className="text-sm text-text-light">个故事</p>
            </div>

            <div className="surface-card card-hover rounded-[1.7rem] p-5 sm:col-span-2 sm:p-6 xl:col-span-1">
              <div className="mb-4 flex items-center justify-between">
                <h3 className="text-base font-semibold text-text sm:text-lg">已完成</h3>
                <div className="flex h-11 w-11 items-center justify-center rounded-lg bg-success text-white sm:h-12 sm:w-12">
                  <CheckCircleIcon size={22} />
                </div>
              </div>
              <p className="mb-1 text-3xl font-bold text-success sm:mb-2 sm:text-4xl">
                {dashboardData?.statistics.completed || 0}
              </p>
              <p className="text-sm text-text-light">个故事</p>
            </div>
          </div>

          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2 lg:gap-8">
            <StoryListSection
              title="我领取的故事"
              count={dashboardData?.my_stories.assigned.length || 0}
              badgeClass="bg-primary-100 text-primary"
              emptyIcon={<InboxIcon size={26} />}
              emptyTitle="还没有领取任何故事"
              emptyAction={
                <Link to="/projects" className="mt-4 inline-block text-primary hover:text-primary-700">
                  去看板看看 →
                </Link>
              }
              items={dashboardData?.my_stories.assigned || []}
            />

            <StoryListSection
              title="我创建的故事"
              count={dashboardData?.my_stories.created.length || 0}
              badgeClass="bg-accent-100 text-accent"
              emptyIcon={<SparklesIcon size={26} />}
              emptyTitle="还没有创建任何故事"
              items={dashboardData?.my_stories.created || []}
            />
          </div>

          <section className="surface-card rounded-[1.8rem] p-5 sm:p-6">
            <div className="mb-5 flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
              <div>
                <h2 className="mb-1 text-lg font-semibold text-text sm:text-xl">快速开始</h2>
                <p className="break-words text-sm text-text-light">
                  {quickStartProject
                    ? `以下操作将作用于「${quickStartProject.name}」`
                    : '先创建一个项目，再开始查看详情、进入看板或创建故事'}
                </p>
              </div>
              {projects.length > 1 && quickStartProject && (
                <div className="w-full lg:w-80">
                  <label className="mb-2 block text-xs font-medium text-text-light">目标项目</label>
                  <select
                    value={selectedQuickProjectId ?? ''}
                    onChange={(e) => setSelectedQuickProjectId(Number(e.target.value))}
                    className="field-control"
                  >
                    {projects.map((project) => (
                      <option key={project.id} value={project.id} className="text-text">
                        {project.name}
                      </option>
                    ))}
                  </select>
                </div>
              )}
            </div>

            {!quickStartProject ? (
              <div className="rounded-2xl border border-dashed border-border bg-secondary-50 p-5">
                <div className="mb-4 text-sm text-text-light">当前还没有可操作的项目</div>
                <Link
                  to="/projects"
                  className="inline-flex rounded-xl border border-border bg-white px-4 py-2 text-sm font-medium text-primary transition-colors hover:border-primary-200 hover:bg-primary-50"
                >
                  去创建项目
                </Link>
              </div>
            ) : (
              <div
                className={`grid grid-cols-1 gap-4 ${
                  canCreateStory ? 'md:grid-cols-3' : 'md:grid-cols-2'
                }`}
              >
                <Link
                  to={`/projects/${quickStartProject.id}`}
                  className="rounded-2xl border border-border bg-secondary-50 p-4 transition-colors hover:border-primary-200 hover:bg-white"
                >
                  <div className="mb-2 text-primary">
                    <FolderIcon size={24} />
                  </div>
                  <h3 className="mb-1 font-semibold text-text">查看项目</h3>
                  <p className="text-sm text-text-light">查看「{quickStartProject.name}」详情</p>
                </Link>

                <Link
                  to={`/projects/${quickStartProject.id}/board`}
                  className="rounded-2xl border border-border bg-secondary-50 p-4 transition-colors hover:border-primary-200 hover:bg-white"
                >
                  <div className="mb-2 text-primary">
                    <BoardIcon size={24} />
                  </div>
                  <h3 className="mb-1 font-semibold text-text">看板视图</h3>
                  <p className="text-sm text-text-light">进入「{quickStartProject.name}」看板</p>
                </Link>

                {canCreateStory && (
                  <Link
                    to={`/projects/${quickStartProject.id}/stories/new`}
                    className="rounded-2xl border border-border bg-secondary-50 p-4 text-left transition-colors hover:border-primary-200 hover:bg-white"
                  >
                    <div className="mb-2 text-primary">
                      <SparklesIcon size={24} />
                    </div>
                    <h3 className="mb-1 font-semibold text-text">创建故事</h3>
                    <p className="text-sm text-text-light">
                      在「{quickStartProject.name}」中创建故事
                    </p>
                  </Link>
                )}
              </div>
            )}
          </section>

          {projects && projects.length > 0 && (
            <section className="surface-card rounded-[1.8rem] p-5 sm:p-6">
              <div className="mb-6 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                <h2 className="text-lg font-semibold text-text sm:text-xl">最近项目</h2>
                <div className="inline-flex items-center gap-1.5 text-xs text-text-light">
                  <span className="inline-flex items-center gap-1 rounded-full bg-secondary-50 px-2 py-1">
                    <CrownIcon size={12} />
                    你是负责人
                  </span>
                </div>
              </div>
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
                {projects.slice(0, 6).map((project) => (
                  <Link
                    key={project.id}
                    to={`/projects/${project.id}`}
                    className="card-hover rounded-2xl border border-border bg-white p-4 transition-all hover:border-primary-200"
                  >
                    <div className="mb-2 flex items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-100">
                        <span className="text-primary font-bold">{project.name[0]}</span>
                      </div>
                      <div className="flex-1 min-w-0">
                        <h3 className="font-medium text-text truncate">{project.name}</h3>
                        <div className="mt-1 flex items-center gap-2 text-xs text-text-light">
                          <span
                            className="inline-flex items-center gap-1 rounded-full bg-secondary-50 px-2 py-1"
                            title={`${project.member_count ?? 0} 名成员`}
                          >
                            <UsersIcon size={12} />
                            <span>{project.member_count ?? 0}</span>
                          </span>
                          <span
                            className="inline-flex items-center gap-1 rounded-full bg-secondary-50 px-2 py-1"
                            title={`${project.story_count ?? 0} 个故事`}
                          >
                            <StoryIcon size={12} />
                            <span>{project.story_count ?? 0}</span>
                          </span>
                          {project.is_owner && (
                            <span
                              className="inline-flex items-center rounded-full bg-amber-50 px-2 py-1 text-amber-700"
                              title="项目负责人"
                              aria-label="项目负责人"
                            >
                              <CrownIcon size={12} />
                            </span>
                          )}
                        </div>
                      </div>
                    </div>
                  </Link>
                ))}
              </div>
            </section>
          )}
        </div>
      )}
    </div>
  );
}
