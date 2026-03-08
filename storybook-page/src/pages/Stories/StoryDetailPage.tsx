import { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useStoryStore } from '../../stores/storyStore';
import { useAuthStore } from '../../stores/authStore';
import { useToast } from '../../components/ui/Toast';
import AcceptanceCriteriaList from '../../components/story/AcceptanceCriteriaList';
import ActivityTimeline from '../../components/story/ActivityTimeline';
import StoryForm from '../../components/story/StoryForm';
import Button from '../../components/ui/Button';
import { StoryDetailSkeleton } from '../../components/ui/Skeleton';
import { projectService } from '../../services/projectService';
import { storyService } from '../../services/storyService';
import { SprintSummary } from '../../types/api';
import { getErrorMessage } from '../../utils/error';
import {
  formatStoryType,
  getStoryTypeColor,
  formatPriority,
  getPriorityColor,
  formatStoryStatus,
  formatSprintStatus,
  getUserInitials,
} from '../../utils/formatters';

interface MemberItem {
  id: number;
  user_id: number;
  role_in_project: string;
  user?: {
    id: number;
    email: string;
  };
}

export default function StoryDetailPage() {
  const { id } = useParams<{ id: string }>();
  const {
    currentStory,
    fetchStory,
    fetchActivities,
    activities,
    claimStory,
    releaseStory,
    isUpdating,
    clearCurrentStory,
  } = useStoryStore();
  const { user } = useAuthStore();
  const { showError, showSuccess } = useToast();
  const [sprints, setSprints] = useState<SprintSummary[]>([]);
  const [isSprintsLoading, setIsSprintsLoading] = useState(false);
  const [isSprintPlanning, setIsSprintPlanning] = useState(false);
  const [sprintLoadError, setSprintLoadError] = useState('');
  const [selectedSprintID, setSelectedSprintID] = useState('');
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [members, setMembers] = useState<MemberItem[]>([]);
  const [isMembersLoading, setIsMembersLoading] = useState(false);
  const [memberLoadError, setMemberLoadError] = useState('');
  const [selectedAssigneeID, setSelectedAssigneeID] = useState('');
  const [isAssigning, setIsAssigning] = useState(false);

  useEffect(() => {
    clearCurrentStory();
    if (id) {
      fetchStory(parseInt(id));
      fetchActivities(parseInt(id));
    }
    return () => {
      clearCurrentStory();
    };
  }, [id, fetchStory, fetchActivities, clearCurrentStory]);

  useEffect(() => {
    if (!currentStory?.project_id) {
      setSprints([]);
      setSprintLoadError('');
      return;
    }

    const loadSprints = async () => {
      try {
        setIsSprintsLoading(true);
        setSprintLoadError('');
        const data = await projectService.getSprints(currentStory.project_id);
        setSprints(data.sprints || []);
      } catch {
        setSprints([]);
        setSprintLoadError('冲刺列表加载失败');
      } finally {
        setIsSprintsLoading(false);
      }
    };

    loadSprints();
  }, [currentStory?.project_id]);

  useEffect(() => {
    const sprintID = currentStory?.sprint_id;
    setSelectedSprintID(sprintID ? String(sprintID) : '');
  }, [currentStory?.sprint_id]);

  useEffect(() => {
    setSelectedAssigneeID(currentStory?.assigned_to?.id ? String(currentStory.assigned_to.id) : '');
  }, [currentStory?.assigned_to?.id]);

  const canManageAssignee = user?.role === 'product' || user?.role === 'admin';

  useEffect(() => {
    if (!currentStory?.project_id || !canManageAssignee) {
      setMembers([]);
      setMemberLoadError('');
      return;
    }

    const loadMembers = async () => {
      try {
        setIsMembersLoading(true);
        setMemberLoadError('');
        const data = await projectService.getProjectMembers(currentStory.project_id);
        setMembers(data.members || []);
      } catch {
        setMembers([]);
        setMemberLoadError('成员列表加载失败');
      } finally {
        setIsMembersLoading(false);
      }
    };

    loadMembers();
  }, [currentStory?.project_id, canManageAssignee]);

  if (!currentStory) {
    return <StoryDetailSkeleton />;
  }

  const handleClaim = async () => {
    try {
      await claimStory(currentStory.id);
      showSuccess('故事领取成功');
    } catch {
      showError('领取失败');
    }
  };

  const handleRelease = async () => {
    try {
      await releaseStory(currentStory.id);
      showSuccess('故事已释放');
    } catch {
      showError('释放失败');
    }
  };

  const canPlanSprint = user?.role === 'product' || user?.role === 'admin';
  const canClaimStory = user?.role === 'developer';
  const canEditStory =
    user?.role === 'product' || user?.role === 'admin' || user?.id === currentStory.created_by.id;
  const developerMembers = members.filter((m) => m.role_in_project === 'developer');

  const handleSprintChange = async (value: string) => {
    if (!currentStory) {
      return;
    }
    if (!canPlanSprint) {
      showError('仅产品经理可规划冲刺');
      return;
    }
    if (value === selectedSprintID) {
      return;
    }

    const targetSprintID = value ? Number(value) : null;
    setSelectedSprintID(value);

    try {
      setIsSprintPlanning(true);
      await storyService.planToSprint(currentStory.id, { sprint_id: targetSprintID });
      await fetchStory(currentStory.id);
      showSuccess(targetSprintID ? '故事已加入冲刺' : '故事已移出冲刺');
    } catch (error: unknown) {
      setSelectedSprintID(currentStory.sprint_id ? String(currentStory.sprint_id) : '');
      showError(getErrorMessage(error, '冲刺规划失败'));
    } finally {
      setIsSprintPlanning(false);
    }
  };

  const handleAssignStory = async () => {
    if (!canManageAssignee) {
      showError('仅产品经理可分配故事');
      return;
    }
    try {
      setIsAssigning(true);
      await storyService.assignStory(currentStory.id, {
        assigned_to: selectedAssigneeID ? Number(selectedAssigneeID) : null,
      });
      await fetchStory(currentStory.id);
      showSuccess(selectedAssigneeID ? '负责人分配成功' : '已清除负责人');
    } catch (error: unknown) {
      showError(getErrorMessage(error, '分配失败'));
    } finally {
      setIsAssigning(false);
    }
  };

  return (
    <div className="max-w-6xl mx-auto p-8">
      {/* 头部 */}
      <div className="mb-8">
        <div className="flex items-center space-x-2 text-sm text-text-light mb-4">
          <Link to={`/projects/${currentStory.project_id}`} className="hover:text-primary">
            项目
          </Link>
          <span>›</span>
          <Link to={`/projects/${currentStory.project_id}/board`} className="hover:text-primary">
            看板
          </Link>
          <span>›</span>
          <span className="text-text">故事详情</span>
        </div>

        <div className="flex items-start justify-between">
          <div className="flex-1">
            <div className="flex items-center space-x-3 mb-4">
              <h1 className="text-3xl font-bold text-text">{currentStory.title}</h1>
              <span
                className={`px-3 py-1 rounded-full text-sm font-medium ${getStoryTypeColor(currentStory.story_type)}`}
              >
                {formatStoryType(currentStory.story_type)}
              </span>
            </div>

            {/* 元信息 */}
            <div className="flex items-center space-x-6 text-sm text-text-light">
              <div className="flex items-center space-x-2">
                <span>优先级:</span>
                <span
                  className={`px-2 py-1 rounded font-medium ${getPriorityColor(currentStory.priority)}`}
                >
                  {formatPriority(currentStory.priority)}
                </span>
              </div>
              {currentStory.story_points && (
                <div className="flex items-center space-x-2">
                  <span>故事点:</span>
                  <span className="font-medium text-text">{currentStory.story_points}</span>
                </div>
              )}
              <div className="flex items-center space-x-2">
                <span>状态:</span>
                <span className="font-medium text-text">
                  {formatStoryStatus(currentStory.status)}
                </span>
              </div>
            </div>
          </div>

          {/* 操作按钮 */}
          <div className="flex items-center gap-3 flex-wrap justify-end">
            {canEditStory && (
              <Button variant="secondary" size="sm" onClick={() => setIsEditOpen(true)}>
                编辑故事
              </Button>
            )}

            {canManageAssignee ? (
              <div className="flex items-center gap-2">
                <select
                  value={selectedAssigneeID}
                  onChange={(e) => setSelectedAssigneeID(e.target.value)}
                  disabled={isMembersLoading}
                  className="px-3 py-2 border border-border rounded-lg text-sm bg-white min-w-[180px]"
                >
                  <option value="">未分配</option>
                  {developerMembers.map((member) => (
                    <option key={member.id} value={member.user_id}>
                      {member.user?.email || `用户 #${member.user_id}`}
                    </option>
                  ))}
                </select>
                <Button size="sm" onClick={handleAssignStory} isLoading={isAssigning}>
                  保存分配
                </Button>
              </div>
            ) : (
              canClaimStory && (
                <>
                  {currentStory.assigned_to ? (
                    <div className="flex items-center space-x-2">
                      <div className="w-8 h-8 rounded-full bg-primary-100 text-primary flex items-center justify-center text-sm font-medium">
                        {getUserInitials(currentStory.assigned_to.email)}
                      </div>
                      <span className="text-sm text-text-light">
                        {currentStory.assigned_to.email}
                      </span>
                      {currentStory.assigned_to.id === user?.id && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={handleRelease}
                          disabled={isUpdating}
                        >
                          释放
                        </Button>
                      )}
                    </div>
                  ) : (
                    <Button onClick={handleClaim} disabled={isUpdating}>
                      领取故事
                    </Button>
                  )}
                </>
              )
            )}
          </div>
        </div>
      </div>

      {/* 内容区 */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* 左侧：故事信息 */}
        <div className="lg:col-span-2 space-y-6">
          {/* 描述 */}
          <div className="bg-white rounded-xl border border-border p-6">
            <h2 className="text-lg font-semibold text-text mb-4">描述</h2>
            {currentStory.description ? (
              <p className="text-text whitespace-pre-wrap">{currentStory.description}</p>
            ) : (
              <p className="text-text-light italic">暂无描述</p>
            )}
          </div>

          {/* 验收标准 */}
          <div className="bg-white rounded-xl border border-border p-6">
            <h2 className="text-lg font-semibold text-text mb-4">验收标准</h2>
            <AcceptanceCriteriaList
              storyId={currentStory.id}
              criteria={currentStory.acceptance_criteria}
            />
          </div>

          {/* 活动历史 */}
          <div className="bg-white rounded-xl border border-border p-6">
            <h2 className="text-lg font-semibold text-text mb-4">活动历史</h2>
            <ActivityTimeline activities={activities} />
          </div>
        </div>

        {/* 右侧：侧边栏 */}
        <div className="space-y-6">
          <div className="bg-white rounded-xl border border-border p-6">
            <h2 className="text-lg font-semibold text-text mb-4">冲刺规划</h2>
            <div className="space-y-3 text-sm">
              <div>
                <label className="block text-text-light mb-2">所属冲刺</label>
                <select
                  value={selectedSprintID}
                  onChange={(e) => handleSprintChange(e.target.value)}
                  disabled={isSprintsLoading || isSprintPlanning || !canPlanSprint}
                  className="w-full px-3 py-2 border border-border rounded-lg text-text bg-white disabled:bg-gray-50"
                >
                  <option value="">不加入冲刺</option>
                  {sprints.map((sprint) => (
                    <option key={sprint.id} value={sprint.id}>
                      {sprint.name}（{formatSprintStatus(sprint.status)}）
                    </option>
                  ))}
                </select>
              </div>
              {isSprintsLoading && <div className="text-text-light">冲刺列表加载中...</div>}
              {!canPlanSprint && <div className="text-text-light">仅产品经理可规划冲刺</div>}
              {sprintLoadError && <div className="text-danger">{sprintLoadError}</div>}
              {!isSprintsLoading && !sprintLoadError && sprints.length === 0 && (
                <div className="text-text-light">
                  当前项目暂无冲刺，可前往
                  <Link
                    to={`/projects/${currentStory.project_id}`}
                    className="text-primary hover:underline mx-1"
                  >
                    项目详情
                  </Link>
                  创建。
                </div>
              )}
            </div>
          </div>

          {canManageAssignee && (
            <div className="bg-white rounded-xl border border-border p-6">
              <h2 className="text-lg font-semibold text-text mb-4">负责人分配</h2>
              <div className="text-sm space-y-2">
                {isMembersLoading && <div className="text-text-light">成员列表加载中...</div>}
                {memberLoadError && <div className="text-danger">{memberLoadError}</div>}
                {!isMembersLoading && !memberLoadError && developerMembers.length === 0 && (
                  <div className="text-text-light">项目中暂无开发成员可分配</div>
                )}
                <div className="text-text-light">
                  说明：产品经理可分配负责人；开发人员可在未分配时自行领取。
                </div>
              </div>
            </div>
          )}

          {/* 基本信息 */}
          <div className="bg-white rounded-xl border border-border p-6">
            <h2 className="text-lg font-semibold text-text mb-4">基本信息</h2>
            <div className="space-y-3 text-sm">
              <div className="flex justify-between">
                <span className="text-text-light">创建者:</span>
                <span className="text-text">{currentStory.created_by.email}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-text-light">创建时间:</span>
                <span className="text-text">
                  {new Date(currentStory.created_at).toLocaleString()}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-text-light">更新时间:</span>
                <span className="text-text">
                  {new Date(currentStory.updated_at).toLocaleString()}
                </span>
              </div>
            </div>
          </div>

          {/* 标签 */}
          {currentStory.tags && currentStory.tags.length > 0 && (
            <div className="bg-white rounded-xl border border-border p-6">
              <h2 className="text-lg font-semibold text-text mb-4">标签</h2>
              <div className="flex flex-wrap gap-2">
                {currentStory.tags.map((tag) => (
                  <span
                    key={tag}
                    className="px-3 py-1 bg-primary-50 text-primary rounded-full text-sm"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>

      <StoryForm
        isOpen={isEditOpen}
        onClose={() => setIsEditOpen(false)}
        projectId={currentStory.project_id}
        storyId={currentStory.id}
        mode="edit"
        onSaved={() => {
          fetchStory(currentStory.id);
          fetchActivities(currentStory.id);
        }}
      />
    </div>
  );
}
