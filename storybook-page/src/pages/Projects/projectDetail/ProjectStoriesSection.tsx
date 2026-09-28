import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { storyService } from '../../../services/storyService';
import { useWebSocket } from '../../../hooks/useWebSocket';
import {
  formatStoryStatus,
  formatStoryType,
  formatPriority,
  getStoryTypeColor,
  getPriorityColor,
} from '../../../utils/formatters';
import type { Story } from '../../../types/models';

interface ProjectStoriesSectionProps {
  projectId: number;
  canCreateStory: boolean;
}

const STATUS_OPTIONS: Array<{ value: string; label: string }> = [
  { value: '', label: '全部状态' },
  { value: 'pending', label: '待审批' },
  { value: 'backlog', label: '待办' },
  { value: 'ready', label: '就绪' },
  { value: 'in_progress', label: '开发中' },
  { value: 'test', label: '测试中' },
  { value: 'done', label: '已完成' },
];

export function ProjectStoriesSection({ projectId, canCreateStory }: ProjectStoriesSectionProps) {
  const [stories, setStories] = useState<Story[]>([]);
  const [total, setTotal] = useState(0);
  const [statusFilter, setStatusFilter] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  const loadStories = useCallback(async () => {
    try {
      setIsLoading(true);
      setError('');
      const data = await storyService.getStories(projectId, {
        status: statusFilter || undefined,
        sort_by: 'priority',
        order: 'desc',
      });
      setStories(data.stories || []);
      setTotal(data.total ?? 0);
    } catch {
      setStories([]);
      setTotal(0);
      setError('需求列表加载失败');
    } finally {
      setIsLoading(false);
    }
  }, [projectId, statusFilter]);

  useEffect(() => {
    void loadStories();
  }, [loadStories]);

  // 冲刺完成/取消会把故事退回待办池，监听收尾事件保持列表新鲜（忽略其他项目）
  useWebSocket(
    useMemo(
      () => ({
        onSprintTerminated: (message) => {
          if (message.project_id === projectId) {
            void loadStories();
          }
        },
      }),
      [projectId, loadStories]
    )
  );

  return (
    <section className="section-card rounded-[1.8rem] p-4 sm:p-5">
      <div className="mb-4 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h2 className="text-lg font-semibold text-text">需求列表</h2>
          <p className="mt-1 text-sm text-text-light">项目内全部用户故事，按优先级排序，点击进入详情。</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <label htmlFor="project-stories-status" className="text-xs text-text-light">
            状态
          </label>
          <select
            id="project-stories-status"
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="field-control w-32"
          >
            {STATUS_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
          <Link
            to={`/projects/${projectId}/board`}
            className="rounded-xl border border-border bg-white px-3 py-2 text-sm text-text-light transition-colors hover:border-primary-200 hover:text-primary"
          >
            查看看板
          </Link>
        </div>
      </div>

      {isLoading ? (
        <div className="state-panel state-panel-loading py-8">需求列表加载中...</div>
      ) : error ? (
        <div className="state-panel state-panel-error">{error}</div>
      ) : stories.length === 0 ? (
        <div className="state-panel state-panel-empty">
          <p>还没有故事</p>
          {canCreateStory && (
            <Link
              to={`/projects/${projectId}/stories/new`}
              className="mt-3 inline-flex rounded-xl bg-primary px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-700"
            >
              创建第一个故事
            </Link>
          )}
        </div>
      ) : (
        <div className="space-y-2">
          {stories.map((story) => (
            <Link
              key={story.id}
              to={`/stories/${story.id}`}
              className="section-block block rounded-2xl p-4 transition-colors hover:border-primary-200 hover:bg-white"
            >
              <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                <div className="min-w-0 flex-1">
                  <div className="mb-2 flex flex-wrap items-center gap-1.5">
                    <span
                      className={`rounded-full px-2 py-1 text-xs font-medium ${getStoryTypeColor(story.story_type)}`}
                    >
                      {formatStoryType(story.story_type)}
                    </span>
                    <span
                      className={`rounded-full px-2 py-1 text-xs font-medium ${getPriorityColor(story.priority)}`}
                    >
                      {formatPriority(story.priority)}
                    </span>
                    <span className="rounded-full bg-secondary-100 px-2 py-1 text-xs text-text">
                      {formatStoryStatus(story.status)}
                    </span>
                    {story.status === 'pending' && story.review_status === 'rejected' && (
                      <span className="rounded-full bg-danger-light px-2 py-1 text-xs font-medium text-danger">
                        被驳回
                      </span>
                    )}
                    {story.status === 'pending' && story.review_status !== 'rejected' && (
                      <span className="rounded-full bg-warning-light px-2 py-1 text-xs font-medium text-warning">
                        待审批
                      </span>
                    )}
                  </div>
                  <h3 className="truncate font-medium text-text">{story.title}</h3>
                </div>
                <div className="flex items-center gap-3 text-xs text-text-light lg:shrink-0">
                  {story.assigned_to && <span>{story.assigned_to.email}</span>}
                  {story.story_points ? <span>{story.story_points} 点</span> : null}
                </div>
              </div>
            </Link>
          ))}
          {total > stories.length && (
            <p className="pt-1 text-center text-xs text-text-light">
              仅显示前 {stories.length} 条（共 {total} 条），可按状态筛选缩小范围
            </p>
          )}
        </div>
      )}
    </section>
  );
}

export default ProjectStoriesSection;
