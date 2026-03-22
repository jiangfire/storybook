import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { storyService } from '../../services/storyService';
import { projectService } from '../../services/projectService';
import { useAuthStore } from '../../stores/authStore';
import { useToast } from '../ui/Toast';
import { aiService } from '../../services/aiService';
import { isValidStoryTitle, isValidStoryDescription } from '../../utils/validators';
import type {
  AIGeneratedStoryResponse,
  AIFormDraft,
  CreateStoryRequest,
  SprintSummary,
  UpdateStoryRequest,
} from '../../types/api';
import type { StoryType } from '../../types/models';
import { getErrorMessage } from '../../utils/error';
import Modal from '../ui/Modal';
import Button from '../ui/Button';
import { ArchiveIcon, BugIcon, SparklesIcon, XIcon } from '../ui/AppIcon';

interface StoryFormProps {
  isOpen: boolean;
  onClose: () => void;
  projectId: number;
  storyId?: number;
  mode: 'create' | 'edit';
  onSaved?: () => void;
}

const initialAIFieldState = {
  title: false,
  description: false,
  story_type: false,
  priority: false,
  story_points: false,
  acceptance_criteria: false,
  tags: false,
};

const initialStoryFormState = {
  title: '',
  description: '',
  story_type: 'feature' as StoryType,
  priority: 2,
  story_points: undefined as number | undefined,
  acceptance_criteria: [] as Array<{ description: string; order: number }>,
  tags: [] as string[],
};

const storyTypeOptions: Array<{
  value: StoryType;
  label: string;
  icon: typeof SparklesIcon;
  activeClass: string;
}> = [
  {
    value: 'feature',
    label: '功能',
    icon: SparklesIcon,
    activeClass: 'border-blue-500 bg-blue-50 text-blue-700',
  },
  {
    value: 'bug',
    label: 'Bug',
    icon: BugIcon,
    activeClass: 'border-red-500 bg-red-50 text-red-700',
  },
  {
    value: 'chore',
    label: '杂项',
    icon: ArchiveIcon,
    activeClass: 'border-primary-500 bg-secondary-50 text-text',
  },
];

const priorityOptions = [
  { value: 0, label: '无' },
  { value: 1, label: '低' },
  { value: 2, label: '中' },
  { value: 3, label: '高' },
  { value: 4, label: '紧急' },
];

export default function StoryForm({
  isOpen,
  onClose,
  projectId,
  storyId,
  mode,
  onSaved,
}: StoryFormProps) {
  const navigate = useNavigate();
  const { showSuccess, showError } = useToast();
  const { user } = useAuthStore();
  const isCreateMode = mode === 'create';
  const canPlanSprint = user?.role === 'product' || user?.role === 'admin';

  const [formData, setFormData] = useState(initialStoryFormState);

  const [fieldErrors, setFieldErrors] = useState<Record<string, string | undefined>>({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [newACText, setNewACText] = useState('');
  const [newTag, setNewTag] = useState('');
  const [sprints, setSprints] = useState<SprintSummary[]>([]);
  const [isSprintsLoading, setIsSprintsLoading] = useState(false);
  const [selectedSprintID, setSelectedSprintID] = useState('');
  const [isSprintSubmitting, setIsSprintSubmitting] = useState(false);
  const [sprintError, setSprintError] = useState('');
  const [aiRequirement, setAIRequirement] = useState('');
  const [isAIGenerating, setIsAIGenerating] = useState(false);
  const [aiError, setAIError] = useState('');
  const [lastAIResult, setLastAIResult] = useState<AIGeneratedStoryResponse | null>(null);
  const [aiFieldState, setAIFieldState] = useState(initialAIFieldState);

  const loadStoryData = useCallback(async () => {
    if (!storyId) return;

    try {
      const story = await storyService.getStory(storyId);
      setFormData({
        title: story.title,
        description: story.description || '',
        story_type: story.story_type,
        priority: story.priority,
        story_points: story.story_points,
        acceptance_criteria: story.acceptance_criteria.map((ac, index) => ({
          description: ac.description,
          order: ac.order || index + 1,
        })),
        tags: story.tags || [],
      });
      setAIFieldState({
        title: Boolean(story.title),
        description: Boolean(story.description),
        story_type: true,
        priority: true,
        story_points: story.story_points !== undefined,
        acceptance_criteria: story.acceptance_criteria.length > 0,
        tags: (story.tags || []).length > 0,
      });
      setSelectedSprintID(story.sprint_id ? String(story.sprint_id) : '');
    } catch {
      showError('加载故事数据失败');
      onClose();
    }
  }, [onClose, showError, storyId]);

  const loadSprints = useCallback(async () => {
    try {
      setIsSprintsLoading(true);
      setSprintError('');
      const data = await projectService.getSprints(projectId);
      setSprints(data.sprints || []);
    } catch {
      setSprints([]);
      setSprintError('冲刺列表加载失败');
    } finally {
      setIsSprintsLoading(false);
    }
  }, [projectId]);

  // 如果是编辑模式，加载故事数据
  useEffect(() => {
    if (!isCreateMode && storyId && isOpen) {
      void loadStoryData();
      void loadSprints();
    }
  }, [isCreateMode, storyId, isOpen, loadStoryData, loadSprints]);

  useEffect(() => {
    if (!isOpen || !isCreateMode) {
      return;
    }

    setFormData(initialStoryFormState);
    setFieldErrors({});
    setNewACText('');
    setNewTag('');
    setAIRequirement('');
    setAIError('');
    setLastAIResult(null);
    setAIFieldState(initialAIFieldState);
    setSelectedSprintID('');
  }, [isOpen, isCreateMode]);

  const validateForm = () => {
    const errors: Record<string, string> = {};

    if (!formData.title) {
      errors.title = '请输入故事标题';
    } else if (!isValidStoryTitle(formData.title)) {
      errors.title = '标题长度应在2-200字符之间';
    }

    if (formData.description && !isValidStoryDescription(formData.description)) {
      errors.description = '描述最多2000字符';
    }

    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async () => {
    if (!validateForm()) return;

    setIsSubmitting(true);
    try {
      if (isCreateMode) {
        const createRequest: CreateStoryRequest = {
          title: formData.title,
          description: formData.description || undefined,
          story_type: formData.story_type,
          priority: formData.priority,
          story_points: formData.story_points as 1 | 2 | 3 | 5 | 8 | 13 | undefined,
          acceptance_criteria: formData.acceptance_criteria.map((ac, index) => ({
            id: `ac-${index + 1}`,
            description: ac.description,
            order: ac.order || index + 1,
          })),
          tags: formData.tags,
        };
        const story = await storyService.createStory(projectId, createRequest);
        showSuccess('故事创建成功');
        navigate(`/stories/${story.id}`, { replace: true });
        onClose();
      } else {
        const updateRequest: UpdateStoryRequest = {
          title: formData.title,
          description: formData.description || undefined,
          story_type: formData.story_type,
          priority: formData.priority,
          story_points: formData.story_points as 1 | 2 | 3 | 5 | 8 | 13 | undefined,
          acceptance_criteria: formData.acceptance_criteria.map((ac, index) => ({
            id: `ac-${index + 1}`,
            description: ac.description,
            order: ac.order || index + 1,
          })),
          tags: formData.tags,
        };
        await storyService.updateStory(storyId!, updateRequest);
        showSuccess('故事更新成功');
        onSaved?.();
        onClose();
      }
    } catch (error: unknown) {
      // 检查是否是权限错误
      const errorMessage = getErrorMessage(error, '');
      if (errorMessage.includes('权限不足') || errorMessage.includes('permission')) {
        showError(
          '权限不足：只有产品经理可以创建用户故事。如需创建故事，请联系项目管理员或使用产品经理账号。'
        );
      } else if (errorMessage.includes('网络') || errorMessage.includes('network')) {
        showError('网络连接失败，请检查网络设置后重试');
      } else {
        showError(errorMessage || '操作失败，请稍后重试');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleUpdateSprint = async () => {
    if (isCreateMode || !storyId) {
      return;
    }
    if (!canPlanSprint) {
      showError('仅产品经理可规划冲刺');
      return;
    }

    try {
      setIsSprintSubmitting(true);
      setSprintError('');
      await storyService.planToSprint(storyId, {
        sprint_id: selectedSprintID ? Number(selectedSprintID) : null,
      });
      showSuccess('冲刺规划更新成功');
      onSaved?.();
    } catch (error: unknown) {
      const msg = getErrorMessage(error, '冲刺规划更新失败');
      setSprintError(msg);
      showError(msg);
    } finally {
      setIsSprintSubmitting(false);
    }
  };

  const addAC = () => {
    if (!newACText.trim()) return;

    setFormData({
      ...formData,
      acceptance_criteria: [
        ...formData.acceptance_criteria,
        {
          description: newACText.trim(),
          order: formData.acceptance_criteria.length,
        },
      ],
    });
    setAIFieldState((prev) => ({ ...prev, acceptance_criteria: true }));
    setNewACText('');
  };

  const removeAC = (index: number) => {
    setFormData({
      ...formData,
      acceptance_criteria: formData.acceptance_criteria.filter((_, i) => i !== index),
    });
    setAIFieldState((prev) => ({ ...prev, acceptance_criteria: true }));
  };

  const addTag = () => {
    if (!newTag.trim()) return;
    if (formData.tags.includes(newTag.trim())) return;

    setFormData({
      ...formData,
      tags: [...formData.tags, newTag.trim()],
    });
    setAIFieldState((prev) => ({ ...prev, tags: true }));
    setNewTag('');
  };

  const removeTag = (tag: string) => {
    setFormData({
      ...formData,
      tags: formData.tags.filter((t) => t !== tag),
    });
    setAIFieldState((prev) => ({ ...prev, tags: true }));
  };

  const applyAIDraft = (draft: AIFormDraft, strategy: 'replace' | 'fill_empty') => {
    const shouldReplace = strategy === 'replace';
    setFormData((prev) => ({
      title: shouldReplace || !prev.title.trim() ? draft.title : prev.title,
      description: shouldReplace || !prev.description.trim() ? draft.description : prev.description,
      story_type:
        shouldReplace ||
        (!aiFieldState.story_type && prev.story_type === initialStoryFormState.story_type)
          ? draft.story_type
          : prev.story_type,
      priority:
        shouldReplace ||
        (!aiFieldState.priority && prev.priority === initialStoryFormState.priority)
          ? draft.priority
          : prev.priority,
      story_points:
        shouldReplace || prev.story_points === undefined ? draft.story_points : prev.story_points,
      acceptance_criteria:
        shouldReplace || prev.acceptance_criteria.length === 0
          ? draft.acceptance_criteria
          : prev.acceptance_criteria,
      tags: shouldReplace || prev.tags.length === 0 ? draft.tags : prev.tags,
    }));
    setAIFieldState((prev) => ({
      title: prev.title || shouldReplace || draft.title.trim().length > 0,
      description: prev.description || shouldReplace || draft.description.trim().length > 0,
      story_type:
        prev.story_type || shouldReplace || draft.story_type !== initialStoryFormState.story_type,
      priority: prev.priority || shouldReplace || draft.priority !== initialStoryFormState.priority,
      story_points: prev.story_points || shouldReplace || draft.story_points !== undefined,
      acceptance_criteria:
        prev.acceptance_criteria || shouldReplace || draft.acceptance_criteria.length > 0,
      tags: prev.tags || shouldReplace || draft.tags.length > 0,
    }));
  };

  const handleAIGenerate = async (strategy: 'replace' | 'fill_empty') => {
    const requirement = aiRequirement.trim();
    if (!requirement) {
      setAIError('请输入需求描述');
      return;
    }
    try {
      setIsAIGenerating(true);
      setAIError('');
      const data = await aiService.generateStory({ requirement });
      setLastAIResult(data);
      applyAIDraft(data.form_draft, strategy);
      if (!data.is_configured) {
        showSuccess('当前未配置 OpenAI，已使用内置规则草稿填充表单');
      } else {
        showSuccess(strategy === 'replace' ? 'AI 已覆盖填充表单' : 'AI 已补充空白字段');
      }
    } catch (error: unknown) {
      setAIError(getErrorMessage(error, 'AI生成失败'));
    } finally {
      setIsAIGenerating(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={isCreateMode ? '创建用户故事' : '编辑用户故事'}
      size="lg"
    >
      <div className="space-y-6">
        <section
          className={`rounded-2xl border p-5 shadow-sm ${
            isCreateMode
              ? 'border-primary-200 bg-gradient-to-br from-primary-50 via-white to-accent-50'
              : 'border-blue-100 bg-blue-50/70'
          }`}
        >
          <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
            <div className="space-y-1">
              <h3 className="text-lg font-semibold text-text">AI 自动填表</h3>
              <p className="text-sm text-text-light">根据需求生成故事草稿</p>
            </div>
            {lastAIResult && (
              <span
                className={`inline-flex items-center self-start rounded-full px-2.5 py-1 text-xs font-medium ${
                  lastAIResult.source === 'openai'
                    ? 'bg-green-100 text-green-800'
                    : 'bg-yellow-100 text-yellow-800'
                }`}
              >
                {lastAIResult.source === 'openai' ? 'OpenAI 协作' : '规则草稿'}
              </span>
            )}
          </div>

          <div className="mt-5 space-y-3">
            <div>
              <label
                htmlFor="ai-requirement"
                className="mb-2 block text-sm font-medium text-text"
              >
                需求描述
              </label>
              <textarea
                id="ai-requirement"
                value={aiRequirement}
                onChange={(e) => setAIRequirement(e.target.value)}
                rows={isCreateMode ? 4 : 3}
                placeholder="输入需求、会议纪要或原话"
                className="w-full rounded-xl border border-primary-200 bg-white/90 px-3 py-3 text-sm leading-6 text-text shadow-sm outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/20 resize-none"
              />
            </div>

            {aiError && <div className="text-xs text-danger">{aiError}</div>}

            {lastAIResult?.warnings && lastAIResult.warnings.length > 0 && (
              <div className="rounded-xl border border-blue-100 bg-white/85 px-3 py-2 space-y-1">
                {lastAIResult.warnings.map((warning) => (
                  <div key={warning} className="text-xs text-text-light">
                    {warning}
                  </div>
                ))}
              </div>
            )}

            <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
              <div className="flex flex-wrap gap-2">
                <Button
                  size="sm"
                  onClick={() => void handleAIGenerate('replace')}
                  isLoading={isAIGenerating}
                >
                  生成草稿
                </Button>
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={() => void handleAIGenerate('fill_empty')}
                  isLoading={isAIGenerating}
                >
                  补空白
                </Button>
              </div>
            </div>
          </div>
        </section>

        <section className="space-y-6 rounded-xl border border-border bg-white p-4 md:p-5">
          {/* 标题 */}
          <div>
            <label htmlFor="story-title" className="block text-sm font-medium text-text mb-2">
              标题 <span className="text-danger">*</span>
            </label>
            <input
              id="story-title"
              type="text"
              value={formData.title}
              onChange={(e) => {
                setFormData({ ...formData, title: e.target.value });
                setAIFieldState((prev) => ({ ...prev, title: true }));
                if (fieldErrors.title) {
                  setFieldErrors({ ...fieldErrors, title: undefined });
                }
              }}
              className={`w-full px-3 py-2 border ${fieldErrors.title ? 'border-danger' : 'border-border'} rounded-lg focus:outline-none focus:ring-2 focus:ring-primary`}
              placeholder="例如：支持用户用邮箱和密码登录"
              maxLength={200}
            />
            {fieldErrors.title && <p className="mt-1 text-sm text-danger">{fieldErrors.title}</p>}
          </div>

          {/* 描述 */}
          <div>
            <label htmlFor="story-description" className="block text-sm font-medium text-text mb-2">
              描述
            </label>
            <textarea
              id="story-description"
              value={formData.description}
              onChange={(e) => {
                setFormData({ ...formData, description: e.target.value });
                setAIFieldState((prev) => ({ ...prev, description: true }));
                if (fieldErrors.description) {
                  setFieldErrors({ ...fieldErrors, description: undefined });
                }
              }}
              className={`w-full px-3 py-2 border ${fieldErrors.description ? 'border-danger' : 'border-border'} rounded-lg focus:outline-none focus:ring-2 focus:ring-primary resize-none`}
              placeholder="作为已注册用户，我想要通过邮箱和密码登录，以便安全访问自己的数据。"
              rows={4}
              maxLength={2000}
            />
            {fieldErrors.description && (
              <p className="mt-1 text-sm text-danger">{fieldErrors.description}</p>
            )}
          </div>

          <div className="space-y-6 border-t border-border pt-6">

          {/* 故事类型 */}
          <div>
            <label className="block text-sm font-medium text-text mb-2">故事类型</label>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
              {storyTypeOptions.map((type) => {
                const Icon = type.icon;
                return (
                  <button
                    key={type.value}
                    type="button"
                    onClick={() => {
                      setFormData({ ...formData, story_type: type.value });
                      setAIFieldState((prev) => ({ ...prev, story_type: true }));
                    }}
                    className={`rounded-lg border-2 p-3 text-left transition-all ${
                      formData.story_type === type.value
                        ? type.activeClass
                        : 'border-border bg-white hover:border-primary-300'
                    }`}
                  >
                    <div className="mb-2 inline-flex h-9 w-9 items-center justify-center rounded-lg bg-white/80">
                      <Icon size={18} />
                    </div>
                    <div className="text-sm font-medium">{type.label}</div>
                  </button>
                );
              })}
            </div>
          </div>

          {/* 优先级和故事点 */}
          <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
            <div>
              <label className="block text-sm font-medium text-text mb-2">优先级</label>
              <div className="flex flex-wrap items-center gap-2">
                {priorityOptions.map((option) => (
                  <button
                    key={option.value}
                    type="button"
                    onClick={() => {
                      setFormData({ ...formData, priority: option.value });
                      setAIFieldState((prev) => ({ ...prev, priority: true }));
                    }}
                    className={`rounded-full px-3 py-1.5 text-sm transition-all ${
                      formData.priority === option.value
                        ? option.value === 4
                          ? 'bg-danger text-white'
                          : option.value === 3
                            ? 'bg-warning text-text'
                            : option.value === 2
                              ? 'bg-warning-light text-warning'
                              : option.value === 1
                                ? 'bg-info-light text-info'
                                : 'bg-secondary-200 text-text'
                        : 'bg-secondary-100 text-text-light hover:bg-secondary-200'
                    }`}
                  >
                    {option.label}
                  </button>
                ))}
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-text mb-2">故事点</label>
              <div className="grid grid-cols-3 gap-2 sm:grid-cols-6">
                {[1, 2, 3, 5, 8, 13].map((points) => (
                  <button
                    key={points}
                    type="button"
                    onClick={() => {
                      setFormData({
                        ...formData,
                        story_points: formData.story_points === points ? undefined : points,
                      });
                      setAIFieldState((prev) => ({ ...prev, story_points: true }));
                    }}
                    className={`rounded-lg border px-3 py-2 text-sm font-medium transition-all ${
                      formData.story_points === points
                        ? 'border-primary bg-primary text-white'
                        : 'border-border bg-white text-text hover:border-primary-300'
                    }`}
                  >
                    {points}
                  </button>
                ))}
              </div>
            </div>
          </div>

          {!isCreateMode && (
            <div>
              <label className="block text-sm font-medium text-text mb-2">冲刺规划</label>
              <div className="flex flex-col gap-2 sm:flex-row">
                <select
                  value={selectedSprintID}
                  onChange={(e) => setSelectedSprintID(e.target.value)}
                  disabled={isSprintsLoading || !canPlanSprint}
                  className="flex-1 rounded-lg border border-border px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary disabled:bg-secondary-50"
                >
                  <option value="">不加入冲刺</option>
                  {sprints.map((sprint) => (
                    <option key={sprint.id} value={sprint.id}>
                      {sprint.name}
                    </option>
                  ))}
                </select>
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={handleUpdateSprint}
                  disabled={isSprintsLoading || !canPlanSprint}
                  isLoading={isSprintSubmitting}
                >
                  更新冲刺
                </Button>
              </div>
              {isSprintsLoading && <p className="mt-2 text-xs text-text-light">冲刺列表加载中...</p>}
              {!canPlanSprint && <p className="mt-2 text-xs text-text-light">仅产品经理可规划冲刺</p>}
              {sprintError && <p className="mt-2 text-xs text-danger">{sprintError}</p>}
            </div>
          )}

          {/* 验收标准 */}
          <div>
            <label className="block text-sm font-medium text-text mb-2">验收标准 (AC)</label>
            <div className="mb-3 space-y-2">
              {formData.acceptance_criteria.map((ac, index) => (
                <div
                  key={index}
                  className="flex items-start space-x-2 rounded-lg bg-secondary-50 p-3"
                >
                  <span className="mt-1 text-text-light">{index + 1}.</span>
                  <span className="flex-1 text-sm">{ac.description}</span>
                  <button
                    type="button"
                    onClick={() => removeAC(index)}
                    aria-label="删除验收标准"
                    className="text-danger hover:text-danger-700"
                  >
                    <XIcon size={14} />
                  </button>
                </div>
              ))}
            </div>
            <div className="flex flex-col gap-2 sm:flex-row">
              <input
                type="text"
                value={newACText}
                onChange={(e) => setNewACText(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    addAC();
                  }
                }}
                className="flex-1 rounded-lg border border-border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                placeholder="添加验收标准...（按 Enter 快速添加）"
              />
              <button
                type="button"
                onClick={addAC}
                className="rounded-lg bg-secondary px-4 py-2 text-text transition-colors hover:bg-primary-50"
              >
                添加
              </button>
            </div>
          </div>

          {/* 标签 */}
          <div>
            <label className="block text-sm font-medium text-text mb-2">标签</label>
            <div className="mb-3 flex flex-wrap gap-2">
              {formData.tags.map((tag) => (
                <span
                  key={tag}
                  className="inline-flex items-center rounded-full bg-primary-50 px-3 py-1 text-sm text-primary"
                >
                  {tag}
                  <button
                    type="button"
                    onClick={() => removeTag(tag)}
                    aria-label="删除标签"
                    className="ml-2 text-primary hover:text-primary-700"
                  >
                    <XIcon size={12} />
                  </button>
                </span>
              ))}
            </div>
            <div className="flex flex-col gap-2 sm:flex-row">
              <input
                type="text"
                value={newTag}
                onChange={(e) => setNewTag(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    addTag();
                  }
                }}
                className="flex-1 rounded-lg border border-border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                placeholder="添加标签...（按 Enter 快速添加）"
              />
              <button
                type="button"
                onClick={addTag}
                className="rounded-lg bg-secondary px-4 py-2 text-text transition-colors hover:bg-primary-50"
              >
                添加
              </button>
            </div>
          </div>
          </div>
        </section>

        {/* 按钮 */}
        <div className="flex justify-end space-x-3 pt-4 border-t border-border">
          <Button variant="secondary" onClick={onClose} disabled={isSubmitting}>
            取消
          </Button>
          <Button onClick={handleSubmit} disabled={isSubmitting} isLoading={isSubmitting}>
            {isSubmitting ? '保存中...' : isCreateMode ? '创建故事' : '保存更改'}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
