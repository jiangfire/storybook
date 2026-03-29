import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { storyService } from '../../services/storyService';
import { projectService } from '../../services/projectService';
import { useAuthStore } from '../../stores/authStore';
import { useToast } from '../ui/Toast';
import { useStoryFormValidation } from '../../hooks/useStoryFormValidation';
import type {
  AIFormDraft,
  CreateStoryRequest,
  SprintSummary,
  UpdateStoryRequest,
} from '../../types/api';
import type { StoryType } from '../../types/models';
import { getErrorMessage } from '../../utils/error';
import Modal from '../ui/Modal';
import Button from '../ui/Button';
import { AICreator } from './AICreator';
import { AcceptanceCriteriaManager } from './AcceptanceCriteriaManager';
import { TagManager } from './TagManager';
import { PrioritySelector } from './PrioritySelector';
import { StoryPointsSelector } from './StoryPointsSelector';
import { StoryTypeSelector } from './StoryTypeSelector';
import { SprintPlanner } from './SprintPlanner';

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

  const { errors, validate, clearErrors } = useStoryFormValidation();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [sprints, setSprints] = useState<SprintSummary[]>([]);
  const [isSprintsLoading, setIsSprintsLoading] = useState(false);
  const [selectedSprintID, setSelectedSprintID] = useState('');
  const [isSprintSubmitting, setIsSprintSubmitting] = useState(false);
  const [sprintError, setSprintError] = useState('');
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
    clearErrors();
    setAIFieldState(initialAIFieldState);
    setSelectedSprintID('');
  }, [isOpen, isCreateMode, clearErrors]);

  const handleSubmit = async () => {
    const validationErrors = validate(formData);
    if (Object.keys(validationErrors).length > 0) return;

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

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={isCreateMode ? '创建用户故事' : '编辑用户故事'}
      size="lg"
    >
      <div className="space-y-6">
        <div className="grid gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(280px,1fr)]">
          {user?.role === 'product' || user?.role === 'admin' ? (
            <AICreator onGenerated={applyAIDraft} />
          ) : null}
        </div>

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
                if (errors.title) {
                  clearErrors();
                }
              }}
              className={`w-full px-3 py-2 border ${errors.title ? 'border-danger' : 'border-border'} rounded-lg focus:outline-none focus:ring-2 focus:ring-primary`}
              placeholder="例如：支持用户用邮箱和密码登录"
              maxLength={200}
            />
            {errors.title && <p className="mt-1 text-sm text-danger">{errors.title}</p>}
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
                if (errors.description) {
                  clearErrors();
                }
              }}
              className={`w-full px-3 py-2 border ${errors.description ? 'border-danger' : 'border-border'} rounded-lg focus:outline-none focus:ring-2 focus:ring-primary resize-none`}
              placeholder="作为已注册用户，我想要通过邮箱和密码登录，以便安全访问自己的数据。"
              rows={4}
              maxLength={2000}
            />
            {errors.description && (
              <p className="mt-1 text-sm text-danger">{errors.description}</p>
            )}
          </div>

          <div className="space-y-6 border-t border-border pt-6">

          {/* 故事类型 */}
          <StoryTypeSelector
            value={formData.story_type}
            onChange={(value) => {
              setFormData({ ...formData, story_type: value });
              setAIFieldState((prev) => ({ ...prev, story_type: true }));
            }}
          />

          {/* 优先级和故事点 */}
          <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
            <PrioritySelector
              value={formData.priority}
              onChange={(value) => {
                setFormData({ ...formData, priority: value });
                setAIFieldState((prev) => ({ ...prev, priority: true }));
              }}
            />

            <StoryPointsSelector
              value={formData.story_points}
              onChange={(value) => {
                setFormData({ ...formData, story_points: value });
                setAIFieldState((prev) => ({ ...prev, story_points: true }));
              }}
            />
          </div>

          {!isCreateMode && (
            <SprintPlanner
              sprintId={selectedSprintID}
              sprints={sprints}
              isLoading={isSprintsLoading}
              canPlan={canPlanSprint}
              isSubmitting={isSprintSubmitting}
              error={sprintError}
              onSprintChange={setSelectedSprintID}
              onUpdate={handleUpdateSprint}
            />
          )}

          {/* 验收标准 */}
          <AcceptanceCriteriaManager
            criteria={formData.acceptance_criteria}
            onChange={(criteria) => {
              setFormData({ ...formData, acceptance_criteria: criteria });
              setAIFieldState((prev) => ({ ...prev, acceptance_criteria: true }));
            }}
          />

          {/* 标签 */}
          <TagManager
            tags={formData.tags}
            onChange={(tags) => {
              setFormData({ ...formData, tags });
              setAIFieldState((prev) => ({ ...prev, tags: true }));
            }}
          />
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
