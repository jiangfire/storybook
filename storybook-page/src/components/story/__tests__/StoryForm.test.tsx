import type { ComponentProps } from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import type { SprintSummary } from '../../../types/api';
import type { Story, User } from '../../../types/models';
import { aiService } from '../../../services/aiService';
import { projectService } from '../../../services/projectService';
import { storyService } from '../../../services/storyService';
import { useAuthStore } from '../../../stores/authStore';
import StoryForm from '../StoryForm';

const navigateMock = vi.fn();
const showSuccess = vi.fn();
const showError = vi.fn();

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom');
  return {
    ...actual,
    useNavigate: () => navigateMock,
  };
});

vi.mock('../../ui/Toast', () => ({
  useToast: () => ({
    showSuccess,
    showError,
  }),
}));

vi.mock('../../../services/storyService', () => ({
  storyService: {
    getStory: vi.fn(),
    createStory: vi.fn(),
    updateStory: vi.fn(),
    planToSprint: vi.fn(),
  },
}));

vi.mock('../../../services/projectService', () => ({
  projectService: {
    getSprints: vi.fn(),
  },
}));

vi.mock('../../../services/aiService', () => ({
  aiService: {
    generateStory: vi.fn(),
  },
}));

const mockedStoryService = vi.mocked(storyService, { deep: true });
const mockedProjectService = vi.mocked(projectService, { deep: true });
const mockedAiService = vi.mocked(aiService, { deep: true });

const NOW = '2026-03-29T00:00:00Z';

const productUser: User = {
  id: 1,
  email: 'pm@example.com',
  role: 'product',
  created_at: NOW,
};

const baseStory: Story = {
  id: 12,
  project_id: 1,
  title: '现有故事',
  description: '已有描述',
  story_type: 'feature',
  status: 'pending',
  priority: 2,
  story_points: 5,
  position: 1,
  created_by: productUser,
  acceptance_criteria: [
    {
      id: 'ac-1',
      description: '已存在 AC',
      status: 'pending',
      order: 1,
    },
  ],
  tags: ['legacy'],
  created_at: NOW,
  updated_at: NOW,
  sprint_id: 31,
};

const sprintList: SprintSummary[] = [
  {
    id: 31,
    project_id: 1,
    name: 'Sprint 1',
    goal: '完成登录',
    start_date: '2026-04-01',
    end_date: '2026-04-14',
    status: 'planned',
    total_stories: 4,
    done_stories: 1,
    created_at: NOW,
    updated_at: NOW,
  },
  {
    id: 32,
    project_id: 1,
    name: 'Sprint 2',
    goal: '完成资料页',
    start_date: '2026-04-15',
    end_date: '2026-04-28',
    status: 'active',
    total_stories: 5,
    done_stories: 2,
    created_at: NOW,
    updated_at: NOW,
  },
];

function setAuthUser(role: User['role']) {
  useAuthStore.setState({
    user: {
      id: role === 'developer' ? 2 : 1,
      email: `${role}@example.com`,
      role,
      created_at: NOW,
    },
    token: 'token',
    isAuthenticated: true,
    isLoading: false,
    error: null,
  });
}

function renderStoryForm(props: Partial<ComponentProps<typeof StoryForm>> = {}) {
  const onClose = props.onClose ?? vi.fn();
  const onSaved = props.onSaved ?? vi.fn();

  const result = render(
    <MemoryRouter>
      <StoryForm
        isOpen
        onClose={onClose}
        projectId={1}
        mode="create"
        onSaved={onSaved}
        {...props}
      />
    </MemoryRouter>
  );

  return {
    ...result,
    onClose,
    onSaved,
  };
}

function findAddButton(placeholderKeyword: string) {
  const addButton = screen.getAllByRole('button', { name: '添加' }).find((button) =>
    button.parentElement?.querySelector(`input[placeholder*="${placeholderKeyword}"]`)
  );
  if (!addButton) {
    throw new Error(`add button for "${placeholderKeyword}" not found`);
  }
  return addButton;
}

function getSprintSelect() {
  const select = screen
    .getAllByRole('combobox')
    .find((element) => element.querySelector('option[value=""]'));

  if (!(select instanceof HTMLSelectElement)) {
    throw new Error('sprint planner select not found');
  }

  return select;
}

describe('StoryForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setAuthUser('product');

    mockedAiService.generateStory.mockResolvedValue({
      form_draft: {
        title: 'AI 生成的标题',
        description: 'AI 生成的描述',
        story_type: 'bug',
        priority: 4,
        story_points: 8,
        acceptance_criteria: [
          { description: 'AI AC 1', order: 1 },
          { description: 'AI AC 2', order: 2 },
        ],
        tags: ['ai'],
      },
    });
    mockedStoryService.getStory.mockResolvedValue(baseStory);
    mockedStoryService.createStory.mockResolvedValue({
      ...baseStory,
      id: 88,
      title: '支持邮箱登录',
      description: '创建后的故事',
      acceptance_criteria: [],
      tags: [],
    });
    mockedStoryService.updateStory.mockResolvedValue(baseStory);
    mockedStoryService.planToSprint.mockResolvedValue({
      story_id: 12,
      sprint_id: 32,
      updated_at: NOW,
    });
    mockedProjectService.getSprints.mockResolvedValue({ sprints: sprintList });
  });

  it('在创建模式下将 AI 共创区放在标题字段之前', () => {
    renderStoryForm();

    const aiInput = screen.getByLabelText('需求描述');
    const titleInput = screen.getByLabelText(/标题/);

    expect(aiInput.compareDocumentPosition(titleInput) & Node.DOCUMENT_POSITION_FOLLOWING).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING
    );
  });

  it('保留模型辅助与规则辅助入口', () => {
    renderStoryForm();

    expect(screen.getByText('模型辅助')).toBeInTheDocument();
    expect(screen.getByText('规则辅助')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '生成草稿' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '补空白' })).toBeInTheDocument();
  });

  it('应该验证必填字段和标题长度', async () => {
    renderStoryForm();

    const titleInput = screen.getByLabelText(/标题/);
    fireEvent.click(screen.getByRole('button', { name: '创建故事' }));

    expect(await screen.findByText('请输入故事标题')).toBeInTheDocument();

    fireEvent.change(titleInput, { target: { value: 'a' } });
    fireEvent.click(screen.getByRole('button', { name: '创建故事' }));

    expect(await screen.findByText('标题长度应在2-200字符之间')).toBeInTheDocument();
  });

  it('规则辅助会补齐空字段，并生成结构化验收标准', async () => {
    renderStoryForm();

    fireEvent.change(screen.getByLabelText('需求描述'), { target: { value: '用户登录功能' } });
    fireEvent.click(screen.getByRole('button', { name: '补空白' }));

    expect(await screen.findByLabelText(/标题/)).toHaveValue('用户登录功能');
    expect(screen.getByLabelText('描述')).toHaveValue('用户登录功能');
    expect(screen.getByText('Given 用户未登录')).toBeInTheDocument();
    expect(screen.getByText('When 用户输入有效的邮箱和密码')).toBeInTheDocument();
    expect(screen.getByText('Then 用户成功登录并跳转到首页')).toBeInTheDocument();
  });

  it('模型辅助 replace 会覆盖已有字段，而 fill_empty 会保留用户已编辑内容', async () => {
    renderStoryForm();

    const titleInput = screen.getByLabelText(/标题/);
    fireEvent.change(titleInput, { target: { value: '用户手写标题' } });
    fireEvent.change(screen.getByLabelText('需求描述'), { target: { value: '登录场景' } });

    fireEvent.click(screen.getByRole('button', { name: '补空白' }));
    expect(await screen.findByDisplayValue('用户手写标题')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '生成草稿' }));

    await waitFor(() => {
      expect(mockedAiService.generateStory).toHaveBeenCalledWith({ requirement: '登录场景' });
      expect(screen.getByDisplayValue('AI 生成的标题')).toBeInTheDocument();
      expect(screen.getByDisplayValue('AI 生成的描述')).toBeInTheDocument();
    });
  });

  it('创建故事成功后会提交表单、跳转详情并关闭弹窗', async () => {
    const user = userEvent.setup();
    const { onClose } = renderStoryForm();

    await user.type(screen.getByLabelText(/标题/), '支持邮箱登录');
    await user.type(screen.getByLabelText('描述'), '作为已注册用户，我想用邮箱和密码登录。');

    fireEvent.change(screen.getByPlaceholderText(/添加验收标准/), {
      target: { value: 'Given 用户输入正确账号密码' },
    });
    fireEvent.click(findAddButton('验收标准'));

    fireEvent.change(screen.getByPlaceholderText(/添加标签/), { target: { value: 'auth' } });
    fireEvent.click(findAddButton('标签'));

    await user.click(screen.getByRole('button', { name: '创建故事' }));

    await waitFor(() => {
      expect(mockedStoryService.createStory).toHaveBeenCalledWith(1, {
        title: '支持邮箱登录',
        description: '作为已注册用户，我想用邮箱和密码登录。',
        story_type: 'feature',
        priority: 2,
        story_points: undefined,
        acceptance_criteria: [
          {
            id: 'ac-1',
            description: 'Given 用户输入正确账号密码',
            order: 1,
          },
        ],
        tags: ['auth'],
      });
    });

    expect(showSuccess).toHaveBeenCalledWith('故事创建成功');
    expect(navigateMock).toHaveBeenCalledWith('/stories/88', { replace: true });
    expect(onClose).toHaveBeenCalled();
  });

  it('创建故事遇到权限或网络错误时会给出明确提示', async () => {
    const user = userEvent.setup();

    mockedStoryService.createStory.mockRejectedValueOnce(new Error('权限不足'));
    const firstRender = renderStoryForm();

    await user.type(screen.getByLabelText(/标题/), '支持邮箱登录');
    await user.click(screen.getByRole('button', { name: '创建故事' }));

    await waitFor(() => {
      expect(showError).toHaveBeenCalledWith(
        '权限不足：只有产品经理可以创建用户故事。如需创建故事，请联系项目管理员或使用产品经理账号。'
      );
    });

    firstRender.unmount();
    mockedStoryService.createStory.mockRejectedValueOnce(new Error('network timeout'));
    renderStoryForm();
    await user.type(screen.getByLabelText(/标题/), '支持邮箱登录');
    await user.click(screen.getByRole('button', { name: '创建故事' }));

    await waitFor(() => {
      expect(showError).toHaveBeenCalledWith('网络连接失败，请检查网络设置后重试');
    });
  });

  it('编辑模式会加载故事和冲刺，并在保存后触发回调', async () => {
    const user = userEvent.setup();
    const { onClose, onSaved } = renderStoryForm({
      mode: 'edit',
      storyId: 12,
    });

    expect(await screen.findByDisplayValue('现有故事')).toBeInTheDocument();
    expect(mockedProjectService.getSprints).toHaveBeenCalledWith(1);
    expect(getSprintSelect()).toHaveValue('31');

    await user.clear(screen.getByLabelText(/标题/));
    await user.type(screen.getByLabelText(/标题/), '重新命名的故事');
    await user.click(screen.getByRole('button', { name: '保存更改' }));

    await waitFor(() => {
      expect(mockedStoryService.updateStory).toHaveBeenCalledWith(12, {
        title: '重新命名的故事',
        description: '已有描述',
        story_type: 'feature',
        priority: 2,
        story_points: 5,
        acceptance_criteria: [
          {
            id: 'ac-1',
            description: '已存在 AC',
            order: 1,
          },
        ],
        tags: ['legacy'],
      });
    });

    expect(showSuccess).toHaveBeenCalledWith('故事更新成功');
    expect(onSaved).toHaveBeenCalled();
    expect(onClose).toHaveBeenCalled();
  });

  it('产品经理在编辑模式下可以更新冲刺规划', async () => {
    const user = userEvent.setup();
    const { onSaved } = renderStoryForm({
      mode: 'edit',
      storyId: 12,
    });

    const sprintSelect = await waitFor(() => getSprintSelect());
    await user.selectOptions(sprintSelect, '32');
    await user.click(screen.getByRole('button', { name: '更新冲刺' }));

    await waitFor(() => {
      expect(mockedStoryService.planToSprint).toHaveBeenCalledWith(12, { sprint_id: 32 });
    });

    expect(showSuccess).toHaveBeenCalledWith('冲刺规划更新成功');
    expect(onSaved).toHaveBeenCalled();
  });

  it('编辑模式加载失败或冲刺加载失败时会正确处理', async () => {
    const onClose = vi.fn();
    mockedStoryService.getStory.mockRejectedValueOnce(new Error('boom'));

    const firstRender = renderStoryForm({
      mode: 'edit',
      storyId: 12,
      onClose,
    });

    await waitFor(() => {
      expect(showError).toHaveBeenCalledWith('加载故事数据失败');
      expect(onClose).toHaveBeenCalled();
    });

    firstRender.unmount();
    mockedStoryService.getStory.mockResolvedValueOnce(baseStory);
    mockedProjectService.getSprints.mockRejectedValueOnce(new Error('load sprints failed'));

    renderStoryForm({
      mode: 'edit',
      storyId: 12,
    });

    expect(await screen.findByText('冲刺列表加载失败')).toBeInTheDocument();
  });

  it('非产品经理不显示 AI 辅助，且不能规划冲刺', async () => {
    setAuthUser('developer');

    renderStoryForm({
      mode: 'edit',
      storyId: 12,
    });

    expect(screen.queryByText('模型辅助')).not.toBeInTheDocument();
    expect(screen.queryByText('规则辅助')).not.toBeInTheDocument();
    expect(await screen.findByText('仅产品经理可规划冲刺')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '更新冲刺' })).toBeDisabled();
  });
});
