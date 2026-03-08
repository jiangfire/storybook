import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { storyService } from '../../services/storyService';
import { projectService } from '../../services/projectService';
import { useAuthStore } from '../../stores/authStore';
import { useToast } from '../ui/Toast';
import { aiService } from '../../services/aiService';
import { isValidStoryTitle, isValidStoryDescription } from '../../utils/validators';
import { CreateStoryRequest, UpdateStoryRequest, SprintSummary } from '../../types/api';
import { StoryType } from '../../types/models';
import { getErrorMessage } from '../../utils/error';
import Modal from '../ui/Modal';
import Button from '../ui/Button';

interface StoryFormProps {
  isOpen: boolean;
  onClose: () => void;
  projectId: number;
  storyId?: number;
  mode: 'create' | 'edit';
  onSaved?: () => void;
}

const storyTypeOptions: Array<{
  value: StoryType;
  label: string;
  emoji: string;
  activeClass: string;
}> = [
  {
    value: 'feature',
    label: '功能',
    emoji: '✨',
    activeClass: 'border-blue-500 bg-blue-50 text-blue-700',
  },
  {
    value: 'bug',
    label: 'Bug',
    emoji: '🐛',
    activeClass: 'border-red-500 bg-red-50 text-red-700',
  },
  {
    value: 'chore',
    label: '杂项',
    emoji: '📦',
    activeClass: 'border-gray-500 bg-gray-50 text-gray-700',
  },
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
  const canPlanSprint = user?.role === 'product' || user?.role === 'admin';

  const [formData, setFormData] = useState({
    title: '',
    description: '',
    story_type: 'feature' as StoryType,
    priority: 2,
    story_points: undefined as number | undefined,
    acceptance_criteria: [] as Array<{ description: string; order: number }>,
    tags: [] as string[],
  });

  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
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
    if (mode === 'edit' && storyId && isOpen) {
      void loadStoryData();
      void loadSprints();
    }
  }, [mode, storyId, isOpen, loadStoryData, loadSprints]);

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
      const request: CreateStoryRequest | UpdateStoryRequest = {
        title: formData.title,
        description: formData.description || undefined,
        story_type: formData.story_type,
        priority: formData.priority,
        story_points: formData.story_points,
        acceptance_criteria: formData.acceptance_criteria.map((ac, index) => ({
          id: `ac-${index + 1}`,
          description: ac.description,
          order: ac.order || index + 1,
        })),
        tags: formData.tags,
      };

      if (mode === 'create') {
        const story = await storyService.createStory(projectId, request);
        showSuccess('故事创建成功');
        navigate(`/stories/${story.id}`, { replace: true });
        onClose();
      } else {
        await storyService.updateStory(storyId!, request);
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
    if (mode !== 'edit' || !storyId) {
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
    setNewACText('');
  };

  const removeAC = (index: number) => {
    setFormData({
      ...formData,
      acceptance_criteria: formData.acceptance_criteria.filter((_, i) => i !== index),
    });
  };

  const addTag = () => {
    if (!newTag.trim()) return;
    if (formData.tags.includes(newTag.trim())) return;

    setFormData({
      ...formData,
      tags: [...formData.tags, newTag.trim()],
    });
    setNewTag('');
  };

  const removeTag = (tag: string) => {
    setFormData({
      ...formData,
      tags: formData.tags.filter((t) => t !== tag),
    });
  };

  const handleAIGenerate = async () => {
    const requirement = aiRequirement.trim();
    if (!requirement) {
      setAIError('请输入需求描述');
      return;
    }
    try {
      setIsAIGenerating(true);
      setAIError('');
      const data = await aiService.generateStory({ requirement });
      const suggestedTitle = data.action ? data.action.slice(0, 200) : requirement.slice(0, 200);
      setFormData((prev) => ({
        ...prev,
        title: suggestedTitle || prev.title,
        description: data.user_story || prev.description,
        story_points: data.story_points || prev.story_points,
        acceptance_criteria: (data.suggested_ac || []).map((desc, index) => ({
          description: desc,
          order: index + 1,
        })),
      }));
      showSuccess('AI草稿已生成并填充表单');
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
      title={mode === 'create' ? '创建用户故事' : '编辑用户故事'}
      size="lg"
    >
      <div className="space-y-6 max-h-[60vh] overflow-auto">
        {/* 标题 */}
        <div>
          <label className="block text-sm font-medium text-text mb-2">
            标题 <span className="text-danger">*</span>
          </label>
          <input
            type="text"
            value={formData.title}
            onChange={(e) => {
              setFormData({ ...formData, title: e.target.value });
              if (fieldErrors.title) {
                setFieldErrors({ ...fieldErrors, title: undefined });
              }
            }}
            className={`w-full px-3 py-2 border ${fieldErrors.title ? 'border-danger' : 'border-gray-300'} rounded-lg focus:outline-none focus:ring-2 focus:ring-primary`}
            placeholder="例如：用户登录功能"
            maxLength={200}
          />
          {fieldErrors.title && <p className="mt-1 text-sm text-danger">{fieldErrors.title}</p>}
        </div>

        {/* 描述 */}
        <div>
          <label className="block text-sm font-medium text-text mb-2">描述</label>
          <textarea
            value={formData.description}
            onChange={(e) => {
              setFormData({ ...formData, description: e.target.value });
              if (fieldErrors.description) {
                setFieldErrors({ ...fieldErrors, description: undefined });
              }
            }}
            className={`w-full px-3 py-2 border ${fieldErrors.description ? 'border-danger' : 'border-gray-300'} rounded-lg focus:outline-none focus:ring-2 focus:ring-primary resize-none`}
            placeholder="作为已注册用户，我想要通过邮箱和密码登录..."
            rows={4}
            maxLength={2000}
          />
          {fieldErrors.description && (
            <p className="mt-1 text-sm text-danger">{fieldErrors.description}</p>
          )}
        </div>

        {mode === 'create' && (
          <div className="bg-blue-50 border border-blue-100 rounded-lg p-4 space-y-3">
            <div className="text-sm font-medium text-text">AI 生成故事草稿</div>
            <textarea
              value={aiRequirement}
              onChange={(e) => setAIRequirement(e.target.value)}
              rows={3}
              placeholder="输入原始需求，AI会生成用户故事、建议AC与故事点"
              className="w-full px-3 py-2 border border-blue-200 rounded-lg resize-none"
            />
            {aiError && <div className="text-xs text-danger">{aiError}</div>}
            <div className="flex justify-end">
              <Button size="sm" variant="secondary" onClick={handleAIGenerate} isLoading={isAIGenerating}>
                AI 生成草稿
              </Button>
            </div>
          </div>
        )}

        {/* 故事类型 */}
        <div>
          <label className="block text-sm font-medium text-text mb-2">故事类型</label>
          <div className="grid grid-cols-3 gap-3">
            {storyTypeOptions.map((type) => (
              <button
                key={type.value}
                type="button"
                onClick={() => setFormData({ ...formData, story_type: type.value })}
                className={`p-3 rounded-lg border-2 transition-all ${
                  formData.story_type === type.value
                    ? type.activeClass
                    : 'border-gray-200 hover:border-gray-300'
                }`}
              >
                <span className="text-2xl">{type.emoji}</span>
                <div className="text-sm font-medium">{type.label}</div>
              </button>
            ))}
          </div>
        </div>

        {/* 优先级和故事点 */}
        <div className="grid grid-cols-2 gap-6">
          {/* 优先级 */}
          <div>
            <label className="block text-sm font-medium text-text mb-2">优先级</label>
            <div className="flex items-center space-x-2">
              {Array.from({ length: 5 }).map((_, i) => (
                <button
                  key={i}
                  type="button"
                  onClick={() => setFormData({ ...formData, priority: i })}
                  className={`w-8 h-8 rounded-full transition-all ${
                    formData.priority >= i ? 'bg-red-500 text-white' : 'bg-gray-200 text-gray-400'
                  }`}
                >
                  <span className="text-xs">🔴</span>
                </button>
              ))}
              <span className="text-sm text-text-light ml-2">
                {formData.priority === 0 && '无'}
                {formData.priority === 1 && '低'}
                {formData.priority === 2 && '中'}
                {formData.priority === 3 && '高'}
                {formData.priority === 4 && '紧急'}
              </span>
            </div>
          </div>

          {/* 故事点 */}
          <div>
            <label className="block text-sm font-medium text-text mb-2">故事点</label>
            <div className="grid grid-cols-6 gap-2">
              {[1, 2, 3, 5, 8, 13].map((points) => (
                <button
                  key={points}
                  type="button"
                  onClick={() =>
                    setFormData({
                      ...formData,
                      story_points: formData.story_points === points ? undefined : points,
                    })
                  }
                  className={`px-3 py-2 rounded-lg border text-sm font-medium transition-all ${
                    formData.story_points === points
                      ? 'border-primary bg-primary text-white'
                      : 'border-gray-200 hover:border-gray-300 text-text'
                  }`}
                >
                  {points}
                </button>
              ))}
            </div>
          </div>
        </div>

        {mode === 'edit' && (
          <div>
            <label className="block text-sm font-medium text-text mb-2">冲刺规划</label>
            <div className="flex items-center gap-2">
              <select
                value={selectedSprintID}
                onChange={(e) => setSelectedSprintID(e.target.value)}
                disabled={isSprintsLoading || !canPlanSprint}
                className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary disabled:bg-gray-50"
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
          <div className="space-y-2 mb-3">
            {formData.acceptance_criteria.map((ac, index) => (
              <div key={index} className="flex items-start space-x-2 p-3 bg-gray-50 rounded-lg">
                <span className="text-text-light mt-1">{index + 1}.</span>
                <span className="flex-1 text-sm">{ac.description}</span>
                <button
                  type="button"
                  onClick={() => removeAC(index)}
                  className="text-danger hover:text-danger-700"
                >
                  ✕
                </button>
              </div>
            ))}
          </div>
          <div className="flex space-x-2">
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
              className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary text-sm"
              placeholder="添加验收标准...（按 Enter 快速添加）"
            />
            <button
              type="button"
              onClick={addAC}
              className="px-4 py-2 bg-secondary text-text rounded-lg hover:bg-primary-50 transition-colors"
            >
              添加
            </button>
          </div>
        </div>

        {/* 标签 */}
        <div>
          <label className="block text-sm font-medium text-text mb-2">标签</label>
          <div className="flex flex-wrap gap-2 mb-3">
            {formData.tags.map((tag) => (
              <span
                key={tag}
                className="inline-flex items-center px-3 py-1 bg-primary-50 text-primary rounded-full text-sm"
              >
                {tag}
                <button
                  type="button"
                  onClick={() => removeTag(tag)}
                  className="ml-2 text-primary hover:text-primary-700"
                >
                  ✕
                </button>
              </span>
            ))}
          </div>
          <div className="flex space-x-2">
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
              className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary text-sm"
              placeholder="添加标签...（按 Enter 快速添加）"
            />
            <button
              type="button"
              onClick={addTag}
              className="px-4 py-2 bg-secondary text-text rounded-lg hover:bg-primary-50 transition-colors"
            >
              添加
            </button>
          </div>
        </div>

        {/* 按钮 */}
        <div className="flex justify-end space-x-3 pt-4 border-t border-border">
          <Button variant="secondary" onClick={onClose} disabled={isSubmitting}>
            取消
          </Button>
          <Button onClick={handleSubmit} disabled={isSubmitting} isLoading={isSubmitting}>
            {isSubmitting ? '保存中...' : mode === 'create' ? '创建故事' : '保存更改'}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
