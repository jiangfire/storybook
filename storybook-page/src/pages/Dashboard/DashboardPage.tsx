import { useEffect, useMemo, useState } from 'react';
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
  CheckCircleIcon,
  CrownIcon,
  FolderIcon,
  InboxIcon,
  SparklesIcon,
  StoryIcon,
  UsersIcon,
  WrenchIcon,
} from '../../components/ui/AppIcon';

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
    <div className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
      {/* 头部 */}
      <div className="mb-8">
        <h1 className="mb-2 text-2xl font-bold text-text sm:text-3xl">
          欢迎回来，{user?.email?.split('@')[0]}
        </h1>
        <p className="text-sm text-text-light sm:text-base">
          {user?.role === 'product' && '产品经理'}
          {user?.role === 'developer' && '开发人员'}
          {user?.role === 'tester' && '测试人员'}
          {user?.role === 'tech_lead' && '技术负责人'}
          {user?.role === 'admin' && '管理员'}
        </p>
      </div>

      {isLoading ? (
        <div className="text-center py-12">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-text-light">加载中...</p>
        </div>
      ) : loadError ? (
        <div className="bg-danger-light text-danger px-4 py-3 rounded-lg">{loadError}</div>
      ) : (
        <div className="space-y-8">
          {/* 统计卡片 */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3 sm:gap-6">
            <div className="bg-white rounded-xl border border-border p-5 card-hover sm:p-6">
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

            <div className="bg-white rounded-xl border border-border p-5 card-hover sm:p-6">
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

            <div className="bg-white rounded-xl border border-border p-5 card-hover sm:p-6 sm:col-span-2 xl:col-span-1">
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

          {/* 我的故事 */}
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2 lg:gap-8">
            {/* 我领取的故事 */}
            <div className="bg-white rounded-xl border border-border p-5 sm:p-6">
              <div className="mb-6 flex items-center justify-between gap-3">
                <h2 className="text-lg font-semibold text-text sm:text-xl">我领取的故事</h2>
                <span className="px-3 py-1 bg-primary-100 text-primary rounded-full text-sm font-medium">
                  {dashboardData?.my_stories.assigned.length || 0}
                </span>
              </div>

              {dashboardData?.my_stories.assigned.length === 0 ? (
                <div className="text-center py-8 text-text-light">
                  <div className="mb-2 inline-flex h-14 w-14 items-center justify-center rounded-full bg-secondary-50 text-text-light">
                    <InboxIcon size={26} />
                  </div>
                  <p>还没有领取任何故事</p>
                  <Link
                    to="/projects"
                    className="inline-block mt-4 text-primary hover:text-primary-700"
                  >
                    去看板看看 →
                  </Link>
                </div>
              ) : (
                <div className="space-y-3">
                  {dashboardData?.my_stories.assigned.map((story) => (
                    <Link
                      key={story.id}
                      to={`/stories/${story.id}`}
                      className="block rounded-lg border border-border p-4 transition-all hover:border-primary hover:shadow-md"
                    >
                      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                        <div className="flex-1">
                          <div className="mb-2 flex flex-wrap items-center gap-2">
                            <span
                              className={`text-xs px-2 py-1 rounded-full font-medium ${getStoryTypeColor(story.story_type || 'feature')}`}
                            >
                              {formatStoryType(story.story_type || 'feature')}
                            </span>
                            <span
                              className={`text-xs px-2 py-1 rounded-full font-medium ${getPriorityColor(story.priority)}`}
                            >
                              {formatPriority(story.priority)}
                            </span>
                          </div>
                          <h3 className="font-medium text-text mb-1">{story.title}</h3>
                          <p className="text-sm text-text-light break-words">{story.project}</p>
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
            </div>

            {/* 我创建的故事 */}
            <div className="bg-white rounded-xl border border-border p-5 sm:p-6">
              <div className="mb-6 flex items-center justify-between gap-3">
                <h2 className="text-lg font-semibold text-text sm:text-xl">我创建的故事</h2>
                <span className="px-3 py-1 bg-accent-100 text-accent rounded-full text-sm font-medium">
                  {dashboardData?.my_stories.created.length || 0}
                </span>
              </div>

              {dashboardData?.my_stories.created.length === 0 ? (
                <div className="text-center py-8 text-text-light">
                  <div className="mb-2 inline-flex h-14 w-14 items-center justify-center rounded-full bg-secondary-50 text-text-light">
                    <SparklesIcon size={26} />
                  </div>
                  <p>还没有创建任何故事</p>
                </div>
              ) : (
                <div className="space-y-3">
                  {dashboardData?.my_stories.created.map((story) => (
                    <Link
                      key={story.id}
                      to={`/stories/${story.id}`}
                      className="block rounded-lg border border-border p-4 transition-all hover:border-primary hover:shadow-md"
                    >
                      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                        <div className="flex-1">
                          <div className="mb-2 flex flex-wrap items-center gap-2">
                            <span
                              className={`text-xs px-2 py-1 rounded-full font-medium ${getStoryTypeColor(story.story_type || 'feature')}`}
                            >
                              {formatStoryType(story.story_type || 'feature')}
                            </span>
                            <span
                              className={`text-xs px-2 py-1 rounded-full font-medium ${getPriorityColor(story.priority)}`}
                            >
                              {formatPriority(story.priority)}
                            </span>
                          </div>
                          <h3 className="font-medium text-text mb-1">{story.title}</h3>
                          <p className="text-sm text-text-light break-words">{story.project}</p>
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
            </div>
          </div>

          {/* 快速操作 */}
          <div className="rounded-xl bg-gradient-to-r from-primary to-accent p-5 text-white sm:p-6 lg:p-8">
            <div className="mb-5 flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
              <div>
                <h2 className="mb-1 text-xl font-bold sm:text-2xl">快速开始</h2>
                <p className="text-sm text-white/85 break-words">
                  {quickStartProject
                    ? `以下操作将作用于「${quickStartProject.name}」`
                    : '先创建一个项目，再开始查看详情、进入看板或创建故事'}
                </p>
              </div>
              {projects.length > 1 && quickStartProject && (
                <div className="w-full lg:w-80">
                  <label className="mb-2 block text-xs font-medium text-white/80">目标项目</label>
                  <select
                    value={selectedQuickProjectId ?? ''}
                    onChange={(e) => setSelectedQuickProjectId(Number(e.target.value))}
                    className="w-full rounded-lg border border-white/70 bg-white px-3 py-2 text-sm text-text outline-none focus:ring-2 focus:ring-white/60"
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
              <div className="rounded-lg bg-white/15 p-5 backdrop-blur-sm">
                <div className="mb-4 text-sm text-white/90">当前还没有可操作的项目</div>
                <Link
                  to="/projects"
                  className="inline-flex rounded-lg bg-white px-4 py-2 text-sm font-medium text-primary transition-colors hover:bg-white/90"
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
                  className="rounded-lg bg-white/20 p-4 backdrop-blur-sm transition-all hover:bg-white/30"
                >
                  <div className="mb-2 text-white">
                    <FolderIcon size={24} />
                  </div>
                  <h3 className="font-semibold mb-1">查看项目</h3>
                  <p className="text-sm opacity-90">查看「{quickStartProject.name}」详情</p>
                </Link>

                <Link
                  to={`/projects/${quickStartProject.id}/board`}
                  className="rounded-lg bg-white/20 p-4 backdrop-blur-sm transition-all hover:bg-white/30"
                >
                  <div className="mb-2 text-white">
                    <BoardIcon size={24} />
                  </div>
                  <h3 className="font-semibold mb-1">看板视图</h3>
                  <p className="text-sm opacity-90">进入「{quickStartProject.name}」看板</p>
                </Link>

                {canCreateStory && (
                  <Link
                    to={`/projects/${quickStartProject.id}/stories/new`}
                    className="rounded-lg bg-white/20 p-4 text-left backdrop-blur-sm transition-all hover:bg-white/30"
                  >
                    <div className="mb-2 text-white">
                      <SparklesIcon size={24} />
                    </div>
                    <h3 className="font-semibold mb-1">创建故事</h3>
                    <p className="text-sm opacity-90">
                      在「{quickStartProject.name}」中创建故事
                    </p>
                  </Link>
                )}
              </div>
            )}
          </div>

          {/* 最近项目 */}
          {projects && projects.length > 0 && (
            <div className="bg-white rounded-xl border border-border p-5 sm:p-6">
              <div className="mb-6 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                <h2 className="text-lg font-semibold text-text sm:text-xl">最近项目</h2>
                <div className="inline-flex items-center gap-1.5 text-xs text-text-light">
                  <span className="inline-flex items-center gap-1 rounded-full bg-secondary-50 px-2 py-1">
                    <CrownIcon size={12} />
                    你是 Owner
                  </span>
                </div>
              </div>
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
                {projects.slice(0, 6).map((project) => (
                  <Link
                    key={project.id}
                    to={`/projects/${project.id}/board`}
                    className="rounded-lg border border-border p-4 transition-all hover:border-primary hover:shadow-md"
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
                              title="项目 Owner"
                              aria-label="项目 Owner"
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
            </div>
          )}
        </div>
      )}
    </div>
  );
}
