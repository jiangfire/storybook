import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { techLeadService } from '../../services/techLeadService';
import type { Story } from '../../types/models';
import LoadingSpinner from '../../components/ui/LoadingSpinner';
import Button from '../../components/ui/Button';
import Modal from '../../components/ui/Modal';

interface ProjectInfo {
  id: number;
  name: string;
  agile_mode: string;
  pending_stories: number;
}

export default function ReviewPage() {
  const navigate = useNavigate();
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
    } catch (error) {
      console.error('Failed to load data:', error);
    } finally {
      setLoading(false);
    }
  }, [selectedProject, searchQuery]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleSearch = () => {
    loadData();
  };

  const openReviewModal = (story: Story, action: 'approve' | 'reject') => {
    setSelectedStory(story);
    setReviewAction(action);
    setReviewComment('');
    setReviewModalOpen(true);
  };

  const handleReview = async () => {
    if (!selectedStory || !reviewAction) return;
    const comment = reviewComment.trim();
    if (reviewAction === 'reject' && !comment) {
      alert('拒绝审批必须填写原因');
      return;
    }

    try {
      setProcessing(true);
      await techLeadService.reviewStory(selectedStory.id, {
        approved: reviewAction === 'approve',
        comment,
      });
      setReviewModalOpen(false);
      loadData();
    } catch (error) {
      console.error('Failed to review story:', error);
      alert('审批失败，请重试');
    } finally {
      setProcessing(false);
    }
  };

  const getStatusBadge = (status: string) => {
    const styles: Record<string, string> = {
      pending: 'bg-yellow-100 text-yellow-800',
      backlog: 'bg-gray-100 text-gray-800',
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
        className={`px-2 py-1 rounded-full text-xs font-medium ${styles[status] || 'bg-gray-100'}`}
      >
        {labels[status] || status}
      </span>
    );
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <LoadingSpinner />
      </div>
    );
  }

  return (
    <div className="container mx-auto px-4 py-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-text mb-2">故事审批</h1>
        <p className="text-text-light">审批产品经理创建的用户故事</p>
      </div>

      {/* 筛选栏 */}
      <div className="bg-white rounded-lg shadow-sm border border-border p-4 mb-6">
        <div className="flex flex-wrap gap-4">
          <div className="flex-1 min-w-[200px]">
            <label className="block text-sm font-medium text-text mb-1">项目</label>
            <select
              value={selectedProject}
              onChange={(e) => setSelectedProject(e.target.value ? Number(e.target.value) : '')}
              className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <option value="">全部项目</option>
              {projects.map((project) => (
                <option key={project.id} value={project.id}>
                  {project.name} ({project.pending_stories} 待审批)
                </option>
              ))}
            </select>
          </div>
          <div className="flex-[2] min-w-[300px]">
            <label className="block text-sm font-medium text-text mb-1">搜索</label>
            <div className="flex gap-2">
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="搜索故事标题..."
                className="flex-1 px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
                onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              />
              <Button onClick={handleSearch}>搜索</Button>
            </div>
          </div>
        </div>
      </div>

      {/* 故事列表 */}
      <div className="bg-white rounded-lg shadow-sm border border-border">
        {stories.length === 0 ? (
          <div className="p-8 text-center text-text-light">
            <p className="text-lg mb-2">🎉 没有待审批的故事</p>
            <p>所有故事都已审批完成</p>
          </div>
        ) : (
          <div className="divide-y divide-border">
            {stories.map((story) => (
              <div key={story.id} className="p-4 hover:bg-gray-50">
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-2">
                      <span className="text-sm text-text-light">#{story.id}</span>
                      {getStatusBadge(story.status)}
                      {story.review_status === 'rejected' && (
                        <span className="px-2 py-1 rounded-full text-xs font-medium bg-red-100 text-red-700">
                          已拒绝
                        </span>
                      )}
                      <span className="text-xs px-2 py-0.5 rounded bg-gray-100 text-gray-600">
                        {story.story_type === 'feature'
                          ? '✨ 功能'
                          : story.story_type === 'bug'
                            ? '🐛 缺陷'
                            : '🔧 任务'}
                      </span>
                      {story.story_points && (
                        <span className="text-xs px-2 py-0.5 rounded bg-blue-50 text-blue-600">
                          {story.story_points} 点
                        </span>
                      )}
                    </div>
                    <h3
                      className="text-lg font-medium text-text mb-1 cursor-pointer hover:text-primary"
                      onClick={() => navigate(`/stories/${story.id}`)}
                    >
                      {story.title}
                    </h3>
                    <div className="flex items-center gap-4 text-sm text-text-light">
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
                  <div className="flex items-center gap-2 ml-4">
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
            ))}
          </div>
        )}
      </div>

      {/* 审批弹窗 */}
      <Modal
        isOpen={reviewModalOpen}
        onClose={() => setReviewModalOpen(false)}
        title={reviewAction === 'approve' ? '审批通过' : '审批拒绝'}
      >
        <div className="space-y-4">
          {selectedStory && (
            <div className="bg-gray-50 p-3 rounded-lg">
              <p className="font-medium text-text">{selectedStory.title}</p>
              <p className="text-sm text-text-light mt-1">
                项目: {selectedStory.project?.name || '-'}
              </p>
            </div>
          )}
          <div>
            <label className="block text-sm font-medium text-text mb-1">
              {reviewAction === 'reject' ? '拒绝原因（必填）' : '审批意见（可选）'}
            </label>
            <textarea
              value={reviewComment}
              onChange={(e) => setReviewComment(e.target.value)}
              placeholder={reviewAction === 'reject' ? '请输入拒绝原因...' : '输入审批意见...'}
              rows={3}
              className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setReviewModalOpen(false)}>
              取消
            </Button>
            <Button
              variant={reviewAction === 'approve' ? 'primary' : 'danger'}
              onClick={handleReview}
              disabled={processing || (reviewAction === 'reject' && !reviewComment.trim())}
            >
              {processing ? '处理中...' : reviewAction === 'approve' ? '确认通过' : '确认拒绝'}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
