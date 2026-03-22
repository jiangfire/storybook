import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { techLeadService } from '../../services/techLeadService';
import type { Story } from '../../types/models';
import LoadingSpinner from '../../components/ui/LoadingSpinner';
import Button from '../../components/ui/Button';
import Modal from '../../components/ui/Modal';
import { useToast } from '../../components/ui/Toast';
import { getErrorMessage } from '../../utils/error';
import {
  ArchiveIcon,
  BugIcon,
  CheckCircleIcon,
  SparklesIcon,
} from '../../components/ui/AppIcon';

interface ProjectInfo {
  id: number;
  name: string;
  agile_mode: string;
  pending_stories: number;
}

export default function ReviewPage() {
  const navigate = useNavigate();
  const { showError, showSuccess } = useToast();
  const [stories, setStories] = useState<Story[]>([]);
  const [projects, setProjects] = useState<ProjectInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedProject, setSelectedProject] = useState<number | ''>('');
  const [searchQuery, setSearchQuery] = useState('');
  const [reviewModalOpen, setReviewModalOpen] = useState(false);
  const [selectedStory, setSelectedStory] = useState<Story | null>(null);
  const [reviewComment, setReviewComment] = useState('');
  const [reviewAction, setReviewAction] = useState<'approve' | 'reject' | null>(null);
  const [processing, setProcessing] = useState(false);
  const [reviewError, setReviewError] = useState('');

  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      const [storiesRes, projectsRes] = await Promise.all([
        techLeadService.getPendingStories({
          project_id: selectedProject || undefined,
          search: searchQuery || undefined,
        }),
        techLeadService.getMyProjects(),
      ]);
      setStories(storiesRes.stories);
      setProjects(projectsRes.projects);
    } catch (error: unknown) {
      showError(getErrorMessage(error, '审批数据加载失败'));
    } finally {
      setLoading(false);
    }
  }, [searchQuery, selectedProject, showError]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleSearch = () => {
    void loadData();
  };

  const openReviewModal = (story: Story, action: 'approve' | 'reject') => {
    setSelectedStory(story);
    setReviewAction(action);
    setReviewComment('');
    setReviewError('');
    setReviewModalOpen(true);
  };

  const handleReview = async () => {
    if (!selectedStory || !reviewAction) return;

    const comment = reviewComment.trim();
    if (reviewAction === 'reject' && !comment) {
      setReviewError('拒绝审批必须填写原因');
      return;
    }

    try {
      setProcessing(true);
      setReviewError('');
      await techLeadService.reviewStory(selectedStory.id, {
        approved: reviewAction === 'approve',
        comment,
      });
      setReviewModalOpen(false);
      showSuccess(reviewAction === 'approve' ? '审批已通过' : '已拒绝该故事');
      void loadData();
    } catch (error: unknown) {
      showError(getErrorMessage(error, '审批失败，请重试'));
    } finally {
      setProcessing(false);
    }
  };

  const getStatusBadge = (status: string) => {
    const styles: Record<string, string> = {
      pending: 'bg-yellow-100 text-yellow-800',
      backlog: 'bg-secondary-100 text-text',
      ready: 'bg-blue-100 text-blue-800',
      in_progress: 'bg-indigo-100 text-indigo-800',
      test: 'bg-purple-100 text-purple-800',
      done: 'bg-green-100 text-green-800',
    };
    const labels: Record<string, string> = {
      pending: '待审批',
      backlog: '待办',
      ready: '就绪',
      in_progress: '进行中',
      test: '测试中',
      done: '已完成',
    };

    return (
      <span
        className={`rounded-full px-2 py-1 text-xs font-medium ${styles[status] || 'bg-secondary-100'}`}
      >
        {labels[status] || status}
      </span>
    );
  };

  const getStoryTypeBadge = (storyType: string) => {
    if (storyType === 'feature') {
      return {
        label: '功能',
        className: 'bg-blue-50 text-blue-700',
        icon: SparklesIcon,
      };
    }
    if (storyType === 'bug') {
      return {
        label: '缺陷',
        className: 'bg-red-50 text-red-700',
        icon: BugIcon,
      };
    }
    return {
      label: '任务',
      className: 'bg-secondary-100 text-text-light',
      icon: ArchiveIcon,
    };
  };

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <LoadingSpinner />
      </div>
    );
  }

  return (
    <div className="container mx-auto px-4 py-6">
      <div className="mb-6">
        <h1 className="mb-2 text-2xl font-bold text-text">故事审批</h1>
        <p className="text-text-light">审批产品经理创建的用户故事</p>
      </div>

      <div className="mb-6 rounded-lg border border-border bg-white p-4 shadow-sm">
        <div className="grid grid-cols-1 gap-4 md:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]">
          <div className="min-w-0">
            <label className="mb-1 block text-sm font-medium text-text">项目</label>
            <select
              value={selectedProject}
              onChange={(e) => setSelectedProject(e.target.value ? Number(e.target.value) : '')}
              className="w-full rounded-lg border border-border px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <option value="">全部项目</option>
              {projects.map((project) => (
                <option key={project.id} value={project.id}>
                  {project.name} ({project.pending_stories} 待审批)
                </option>
              ))}
            </select>
          </div>

          <div className="min-w-0">
            <label className="mb-1 block text-sm font-medium text-text">搜索</label>
            <div className="flex flex-col gap-2 sm:flex-row">
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="搜索故事标题..."
                className="flex-1 rounded-lg border border-border px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary-500"
                onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              />
              <Button onClick={handleSearch}>搜索</Button>
            </div>
          </div>
        </div>
      </div>

      <div className="rounded-lg border border-border bg-white shadow-sm">
        {stories.length === 0 ? (
          <div className="p-8 text-center text-text-light">
            <div className="mb-3 inline-flex h-14 w-14 items-center justify-center rounded-full bg-secondary-50">
              <CheckCircleIcon size={24} />
            </div>
            <p className="mb-2 text-lg">没有待审批的故事</p>
            <p>所有故事都已审批完成</p>
          </div>
        ) : (
          <div className="divide-y divide-border">
            {stories.map((story) => {
              const typeBadge = getStoryTypeBadge(story.story_type);
              const TypeIcon = typeBadge.icon;

              return (
                <div key={story.id} className="p-4 hover:bg-primary-50">
                  <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
                    <div className="flex-1">
                      <div className="mb-2 flex flex-wrap items-center gap-2">
                        <span className="text-sm text-text-light">#{story.id}</span>
                        {getStatusBadge(story.status)}
                        {story.review_status === 'rejected' && (
                          <span className="rounded-full bg-red-100 px-2 py-1 text-xs font-medium text-red-700">
                            已拒绝
                          </span>
                        )}
                        <span
                          className={`inline-flex items-center gap-1 rounded px-2 py-0.5 text-xs ${typeBadge.className}`}
                        >
                          <TypeIcon size={12} />
                          <span>{typeBadge.label}</span>
                        </span>
                        {story.story_points && (
                          <span className="rounded bg-blue-50 px-2 py-0.5 text-xs text-blue-600">
                            {story.story_points} 点
                          </span>
                        )}
                      </div>

                      <h3
                        className="mb-1 cursor-pointer text-lg font-medium text-text hover:text-primary"
                        onClick={() => navigate(`/stories/${story.id}`)}
                      >
                        {story.title}
                      </h3>

                      <div className="flex flex-wrap items-center gap-4 text-sm text-text-light">
                        <span>项目: {story.project?.name || '-'}</span>
                        <span>创建者: {story.created_by?.email || '-'}</span>
                        <span>创建时间: {new Date(story.created_at).toLocaleDateString()}</span>
                      </div>

                      {story.review_status === 'rejected' && story.review_comment && (
                        <div className="mt-2 text-sm text-red-600">
                          拒绝原因：{story.review_comment}
                        </div>
                      )}
                    </div>

                    <div className="flex flex-wrap items-center gap-2 xl:ml-4 xl:justify-end">
                      <Button
                        variant="secondary"
                        size="sm"
                        onClick={() => navigate(`/stories/${story.id}`)}
                      >
                        查看
                      </Button>
                      <Button
                        variant="primary"
                        size="sm"
                        onClick={() => openReviewModal(story, 'approve')}
                      >
                        通过
                      </Button>
                      <Button
                        variant="danger"
                        size="sm"
                        onClick={() => openReviewModal(story, 'reject')}
                      >
                        拒绝
                      </Button>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      <Modal
        isOpen={reviewModalOpen}
        onClose={() => {
          setReviewModalOpen(false);
          setReviewError('');
        }}
        title={reviewAction === 'approve' ? '审批通过' : '审批拒绝'}
      >
        <div className="space-y-4">
          {selectedStory && (
            <div className="rounded-lg bg-secondary-50 p-3">
              <p className="font-medium text-text">{selectedStory.title}</p>
              <p className="mt-1 text-sm text-text-light">
                项目: {selectedStory.project?.name || '-'}
              </p>
            </div>
          )}

          <div>
            <label className="mb-1 block text-sm font-medium text-text">
              {reviewAction === 'reject' ? '拒绝原因（必填）' : '审批意见（可选）'}
            </label>
            <textarea
              value={reviewComment}
              onChange={(e) => setReviewComment(e.target.value)}
              placeholder={reviewAction === 'reject' ? '请输入拒绝原因...' : '输入审批意见...'}
              rows={3}
              className="w-full rounded-lg border border-border px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          </div>

          {reviewError && (
            <div className="rounded-lg bg-danger-light px-3 py-2 text-sm text-danger">
              {reviewError}
            </div>
          )}

          <div className="flex justify-end gap-2">
            <Button
              variant="secondary"
              onClick={() => {
                setReviewModalOpen(false);
                setReviewError('');
              }}
            >
              取消
            </Button>
            <Button
              variant={reviewAction === 'approve' ? 'primary' : 'danger'}
              onClick={handleReview}
              disabled={processing}
            >
              {processing ? '处理中...' : reviewAction === 'approve' ? '确认通过' : '确认拒绝'}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
