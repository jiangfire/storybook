import { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useStoryStore } from '../../stores/storyStore';
import { useAuthStore } from '../../stores/authStore';
import { useToast } from '../../components/ui/Toast';
import AcceptanceCriteriaList from '../../components/story/AcceptanceCriteriaList';
import ActivityTimeline from '../../components/story/ActivityTimeline';
import StoryTasksPanel from '../../components/story/StoryTasksPanel';
import StoryTestCasesPanel from '../../components/story/StoryTestCasesPanel';
import StoryForm from '../../components/story/StoryForm';
import Button from '../../components/ui/Button';
import { StoryDetailSkeleton } from '../../components/ui/Skeleton';
import { projectService } from '../../services/projectService';
import { storyService } from '../../services/storyService';
import { aiService } from '../../services/aiService';
import type { AISplitStoryData, INVESTCheckData, SprintSummary } from '../../types/api';
import { getErrorMessage } from '../../utils/error';
import {
  canClaimStory as canClaimStoryPermission,
  canManageStoryAssignee,
  canReleaseStory as canReleaseStoryPermission,
  canUseStoryAI,
} from '../../utils/permissions';
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
  const [isInvestLoading, setIsInvestLoading] = useState(false);
  const [investError, setInvestError] = useState('');
  const [investResult, setInvestResult] = useState<INVESTCheckData | null>(null);
  const [splitTargetCount, setSplitTargetCount] = useState(3);
  const [isSplitLoading, setIsSplitLoading] = useState(false);
  const [splitError, setSplitError] = useState('');
  const [splitResult, setSplitResult] = useState<AISplitStoryData | null>(null);

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

  const canManageAssignee = canManageStoryAssignee(user?.role);

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
  const canUseAIInStory = canUseStoryAI(user?.role);
  const canClaimCurrentStory = canClaimStoryPermission(user?.role);
  const canReleaseCurrentStory = canReleaseStoryPermission(user, currentStory.assigned_to);
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
      showError('仅产品经理、技术负责人或管理员可分配故事');
      return;
    }
    if (!selectedAssigneeID && currentStory.assigned_to) {
      await handleRelease();
      return;
    }
    if (!selectedAssigneeID && !currentStory.assigned_to) {
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

  const handleInvestCheck = async () => {
    try {
      setIsInvestLoading(true);
      setInvestError('');
      const data = await aiService.checkInvest(currentStory.id);
      setInvestResult(data);
      showSuccess('INVEST 检查完成');
    } catch (error: unknown) {
      const msg = getErrorMessage(error, 'INVEST 检查失败');
      setInvestError(msg);
      showError(msg);
    } finally {
      setIsInvestLoading(false);
    }
  };

  const handleSplitStory = async () => {
    try {
      setIsSplitLoading(true);
      setSplitError('');
      const data = await aiService.splitStory(currentStory.id, { target_count: splitTargetCount });
      setSplitResult(data);
      showSuccess('AI 拆分建议已生成');
    } catch (error: unknown) {
      const msg = getErrorMessage(error, 'AI 拆分失败');
      setSplitError(msg);
      showError(msg);
    } finally {
      setIsSplitLoading(false);
    }
  };

  return (
    <div className="mx-auto max-w-6xl px-4 py-6 sm:px-6 lg:px-8">
      {/* 头部 */}
      <div className="mb-8">
        <div className="mb-4 flex flex-wrap items-center gap-2 text-sm text-text-light">
          <Link
            to={`/projects/${currentStory.project_id}`}
            className="transition-colors hover:text-primary"
          >
            项目
          </Link>
          <span>›</span>
          <Link
            to={`/projects/${currentStory.project_id}/board`}
            className="transition-colors hover:text-primary"
          >
            看板
          </Link>
          <span>›</span>
          <span className="text-text">故事详情</span>
        </div>

        <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
          <div className="flex-1">
            <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center">
              <h1 className="text-2xl font-bold text-text sm:text-3xl">{currentStory.title}</h1>
              <span
                className={`inline-flex self-start rounded-full px-3 py-1 text-sm font-medium ${getStoryTypeColor(currentStory.story_type)}`}
              >
                {formatStoryType(currentStory.story_type)}
              </span>
            </div>

            {/* 元信息 */}
            <div className="flex flex-wrap items-center gap-4 text-sm text-text-light sm:gap-6">
              <div className="flex items-center gap-2">
                <span>优先级:</span>
                <span
                  className={`rounded px-2 py-1 font-medium ${getPriorityColor(currentStory.priority)}`}
                >
                  {formatPriority(currentStory.priority)}
                </span>
              </div>
              {currentStory.story_points && (
                <div className="flex items-center gap-2">
                  <span>故事点:</span>
                  <span className="font-medium text-text">{currentStory.story_points}</span>
                </div>
              )}
              <div className="flex items-center gap-2">
                <span>状态:</span>
                <span className="font-medium text-text">
                  {formatStoryStatus(currentStory.status)}
                </span>
              </div>
            </div>
            {currentStory.review_status === 'rejected' && currentStory.review_comment && (
              <div className="mt-3 inline-flex items-center rounded-md bg-red-50 px-3 py-2 text-sm text-red-700">
                拒绝原因：{currentStory.review_comment}
              </div>
            )}
          </div>

          {/* 操作按钮 */}
          <div className="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center sm:justify-end">
            {canEditStory && (
              <Button variant="secondary" size="sm" onClick={() => setIsEditOpen(true)}>
                编辑故事
              </Button>
            )}

            {canManageAssignee ? (
              <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                <select
                  value={selectedAssigneeID}
                  onChange={(e) => setSelectedAssigneeID(e.target.value)}
                  disabled={isMembersLoading}
                  className="w-full rounded-lg border border-border bg-white px-3 py-2 text-sm sm:min-w-[180px]"
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
            ) : null}

            {currentStory.assigned_to ? (
              <div className="flex flex-wrap items-center gap-2">
                <div className="w-8 h-8 rounded-full bg-primary-100 text-primary flex items-center justify-center text-sm font-medium">
                  {getUserInitials(currentStory.assigned_to.email)}
                </div>
                <span className="text-sm text-text-light">{currentStory.assigned_to.email}</span>
                {canReleaseCurrentStory && (
                  <Button variant="ghost" size="sm" onClick={handleRelease} disabled={isUpdating}>
                    释放
                  </Button>
                )}
              </div>
            ) : (
              canClaimCurrentStory && (
                <Button onClick={handleClaim} disabled={isUpdating}>
                  领取故事
                </Button>
              )
            )}
          </div>
        </div>
      </div>

      {/* 内容区 */}
      <div className="grid grid-cols-1 gap-8 lg:grid-cols-3">
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

          <StoryTasksPanel storyId={currentStory.id} />

          <StoryTestCasesPanel storyId={currentStory.id} />
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
                  className="w-full px-3 py-2 border border-border rounded-lg text-text bg-white disabled:bg-secondary-50"
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
                    className="mx-1 text-primary transition-colors hover:text-primary-700"
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
                  说明：产品经理、技术负责人和管理员可分配负责人；开发人员和管理员可在未分配时自行领取。
                </div>
              </div>
            </div>
          )}

          {canUseAIInStory && (
            <>
              <div className="bg-white rounded-xl border border-primary-100 p-6">
                <div className="space-y-1">
                  <h2 className="text-lg font-semibold text-text">模型辅助</h2>
                  <p className="text-sm text-text-light">
                    当前详情页未接入直接模型操作；模型草稿生成在创建/编辑故事表单中使用。
                  </p>
                </div>
                {canEditStory && (
                  <div className="mt-4">
                    <Button variant="secondary" size="sm" onClick={() => setIsEditOpen(true)}>
                      打开编辑表单
                    </Button>
                  </div>
                )}
              </div>

              <div className="bg-white rounded-xl border border-border p-6 space-y-4">
                <div className="space-y-1">
                  <h2 className="text-lg font-semibold text-text">规则辅助</h2>
                  <p className="text-sm text-text-light">
                    当前为规则/启发式分析，不调用大模型
                  </p>
                </div>
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-text-light">INVEST 规则检查</span>
                    <Button
                      size="sm"
                      variant="secondary"
                      onClick={handleInvestCheck}
                      isLoading={isInvestLoading}
                    >
                      执行检查
                    </Button>
                  </div>
                  {investError && <div className="text-xs text-danger">{investError}</div>}
                  {investResult && (
                    <div className="bg-secondary-50 border border-border rounded-lg p-3 text-sm space-y-2">
                      <div>
                        总分：
                        <span className="font-semibold">{investResult.invest_score.toFixed(1)}</span>
                      </div>
                      <div className="space-y-1 text-xs">
                        {Object.entries(investResult.checks).map(([key, val]) => (
                          <div key={key} className="flex items-center justify-between">
                            <span>{val.title}</span>
                            <span>
                              {val.score.toFixed(2)} · {val.status}
                            </span>
                          </div>
                        ))}
                      </div>
                      {investResult.suggestions.length > 0 && (
                        <div className="text-xs text-text-light">
                          建议：{investResult.suggestions.join('；')}
                        </div>
                      )}
                    </div>
                  )}
                </div>

                <div className="space-y-2">
                  <div className="text-sm text-text-light">规则拆分建议</div>
                  <div className="flex items-center gap-2">
                    <input
                      type="number"
                      min={1}
                      max={8}
                      value={splitTargetCount}
                      onChange={(e) => setSplitTargetCount(Number(e.target.value) || 3)}
                      className="w-24 px-2 py-1 border border-border rounded text-sm"
                    />
                    <Button
                      size="sm"
                      variant="secondary"
                      onClick={handleSplitStory}
                      isLoading={isSplitLoading}
                    >
                      生成拆分建议
                    </Button>
                  </div>
                  {splitError && <div className="text-xs text-danger">{splitError}</div>}
                  {splitResult && splitResult.sub_stories.length > 0 && (
                    <div className="space-y-2">
                      {splitResult.sub_stories.map((item, idx) => (
                        <div
                          key={`${item.title}-${idx}`}
                          className="border border-border rounded-lg p-2"
                        >
                          <div className="text-sm font-medium text-text">{item.title}</div>
                          <div className="text-xs text-text-light mt-1">
                            点数 {item.story_points} · AC {item.acceptance_criteria.length} 条
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            </>
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
