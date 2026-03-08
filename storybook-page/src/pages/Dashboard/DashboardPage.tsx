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
import { ApiResponse, DashboardData } from '../../types/api';

export default function DashboardPage() {
  const { user } = useAuthStore();
  const { projects, fetchProjects } = useProjectStore();
  const [dashboardData, setDashboardData] = useState<DashboardData | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState('');

  const preferredStoryCreatePath = useMemo(() => {
    if (!projects || projects.length === 0) {
      return '/projects';
    }
    return `/projects/${projects[0].id}/stories/new`;
  }, [projects]);

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
    <div className="p-8 max-w-7xl mx-auto">
      {/* 头部 */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-text mb-2">
          欢迎回来，{user?.email?.split('@')[0]}! 👋
        </h1>
        <p className="text-text-light">
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
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="bg-white rounded-xl border border-border p-6 card-hover">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-text">总领取</h3>
                <div className="w-12 h-12 bg-primary-100 rounded-lg flex items-center justify-center">
                  <span className="text-2xl">📝</span>
                </div>
              </div>
              <p className="text-4xl font-bold text-primary mb-2">
                {dashboardData?.statistics.total_assigned || 0}
              </p>
              <p className="text-sm text-text-light">个故事</p>
            </div>

            <div className="bg-white rounded-xl border border-border p-6 card-hover">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-text">进行中</h3>
                <div className="w-12 h-12 bg-yellow-100 rounded-lg flex items-center justify-center">
                  <span className="text-2xl">🔨</span>
                </div>
              </div>
              <p className="text-4xl font-bold text-warning mb-2">
                {dashboardData?.statistics.in_progress || 0}
              </p>
              <p className="text-sm text-text-light">个故事</p>
            </div>

            <div className="bg-white rounded-xl border border-border p-6 card-hover">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-text">已完成</h3>
                <div className="w-12 h-12 bg-success rounded-lg flex items-center justify-center">
                  <span className="text-2xl">✅</span>
                </div>
              </div>
              <p className="text-4xl font-bold text-success mb-2">
                {dashboardData?.statistics.completed || 0}
              </p>
              <p className="text-sm text-text-light">个故事</p>
            </div>
          </div>

          {/* 我的故事 */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
            {/* 我领取的故事 */}
            <div className="bg-white rounded-xl border border-border p-6">
              <div className="flex items-center justify-between mb-6">
                <h2 className="text-xl font-semibold text-text">我领取的故事</h2>
                <span className="px-3 py-1 bg-primary-100 text-primary rounded-full text-sm font-medium">
                  {dashboardData?.my_stories.assigned.length || 0}
                </span>
              </div>

              {dashboardData?.my_stories.assigned.length === 0 ? (
                <div className="text-center py-8 text-text-light">
                  <div className="text-4xl mb-2">📭</div>
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
                      className="block p-4 rounded-lg border border-border hover:border-primary hover:shadow-md transition-all"
                    >
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="flex items-center space-x-2 mb-2">
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
                          <p className="text-sm text-text-light">{story.project}</p>
                        </div>
                        <div className="ml-4">
                          <span className="px-3 py-1 bg-gray-100 text-text rounded-lg text-sm">
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
            <div className="bg-white rounded-xl border border-border p-6">
              <div className="flex items-center justify-between mb-6">
                <h2 className="text-xl font-semibold text-text">我创建的故事</h2>
                <span className="px-3 py-1 bg-accent-100 text-accent rounded-full text-sm font-medium">
                  {dashboardData?.my_stories.created.length || 0}
                </span>
              </div>

              {dashboardData?.my_stories.created.length === 0 ? (
                <div className="text-center py-8 text-text-light">
                  <div className="text-4xl mb-2">✨</div>
                  <p>还没有创建任何故事</p>
                </div>
              ) : (
                <div className="space-y-3">
                  {dashboardData?.my_stories.created.map((story) => (
                    <Link
                      key={story.id}
                      to={`/stories/${story.id}`}
                      className="block p-4 rounded-lg border border-border hover:border-primary hover:shadow-md transition-all"
                    >
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="flex items-center space-x-2 mb-2">
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
                          <p className="text-sm text-text-light">{story.project}</p>
                        </div>
                        <div className="ml-4">
                          <span className="px-3 py-1 bg-gray-100 text-text rounded-lg text-sm">
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
          <div className="bg-gradient-to-r from-primary to-accent rounded-xl p-8 text-white">
            <h2 className="text-2xl font-bold mb-4">快速开始</h2>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <Link
                to="/projects"
                className="bg-white/20 backdrop-blur-sm rounded-lg p-4 hover:bg-white/30 transition-all"
              >
                <div className="text-3xl mb-2">📁</div>
                <h3 className="font-semibold mb-1">查看项目</h3>
                <p className="text-sm opacity-90">浏览所有项目</p>
              </Link>

              <Link
                to="/projects"
                className="bg-white/20 backdrop-blur-sm rounded-lg p-4 hover:bg-white/30 transition-all"
              >
                <div className="text-3xl mb-2">📋</div>
                <h3 className="font-semibold mb-1">看板视图</h3>
                <p className="text-sm opacity-90">管理故事进度</p>
              </Link>

              <Link
                to={preferredStoryCreatePath}
                className="bg-white/20 backdrop-blur-sm rounded-lg p-4 hover:bg-white/30 transition-all text-left"
              >
                <div className="text-3xl mb-2">✨</div>
                <h3 className="font-semibold mb-1">创建故事</h3>
                <p className="text-sm opacity-90">添加新故事</p>
              </Link>
            </div>
          </div>

          {/* 最近项目 */}
          {projects && projects.length > 0 && (
            <div className="bg-white rounded-xl border border-border p-6">
              <h2 className="text-xl font-semibold text-text mb-6">最近项目</h2>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {projects.slice(0, 6).map((project) => (
                  <Link
                    key={project.id}
                    to={`/projects/${project.id}/board`}
                    className="p-4 rounded-lg border border-border hover:border-primary hover:shadow-md transition-all"
                  >
                    <div className="flex items-center space-x-3 mb-2">
                      <div className="w-10 h-10 bg-primary-100 rounded-lg flex items-center justify-center">
                        <span className="text-primary font-bold">{project.name[0]}</span>
                      </div>
                      <div className="flex-1 min-w-0">
                        <h3 className="font-medium text-text truncate">{project.name}</h3>
                        <p className="text-xs text-text-light">
                          {project.member_count} 成员 · {project.story_count} 故事
                        </p>
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
