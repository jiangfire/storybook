import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aiService } from '../../../services/aiService';
import { storyService } from '../../../services/storyService';
import { StoryAIPanel } from '../StoryAIPanel';

const showSuccess = vi.fn();
const showError = vi.fn();

vi.mock('../../ui/Toast', () => ({
  useToast: () => ({
    showSuccess,
    showError,
  }),
}));

vi.mock('../../../services/aiService', () => ({
  aiService: {
    refineAC: vi.fn(),
    summarizeStory: vi.fn(),
    translateStory: vi.fn(),
    checkDoR: vi.fn(),
  },
}));

vi.mock('../../../services/storyService', () => ({
  storyService: {
    addAC: vi.fn(),
  },
}));

const mockedAiService = vi.mocked(aiService, { deep: true });
const mockedStoryService = vi.mocked(storyService, { deep: true });

describe('StoryAIPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('DoR 检查展示评分与逐项结果', async () => {
    const user = userEvent.setup();
    mockedAiService.checkDoR.mockResolvedValue({
      story_id: 12,
      ready: false,
      score: 50,
      passed: 3,
      total: 6,
      checks: [
        { key: 'title', title: '标题清晰', pass: true, message: '' },
        { key: 'description', title: '描述完整', pass: false, message: '描述需≥30个字符' },
      ],
      suggestions: ['描述需≥30个字符'],
    });

    render(<StoryAIPanel storyId={12} canRefineAC={false} />);

    await user.click(screen.getByRole('button', { name: '检查' }));

    expect(await screen.findByText(/尚未就绪 · 50 分/)).toBeInTheDocument();
    expect(screen.getByText('标题清晰')).toBeInTheDocument();
    expect(screen.getByText(/建议：描述需≥30个字符/)).toBeInTheDocument();
  });

  it('干系人摘要生成后展示文本', async () => {
    const user = userEvent.setup();
    mockedAiService.summarizeStory.mockResolvedValue({
      story_id: 12,
      summary: '该故事将支持邮箱登录，覆盖验证码校验场景。',
    });

    render(<StoryAIPanel storyId={12} canRefineAC={false} />);

    await user.click(screen.getByRole('button', { name: '生成' }));

    expect(
      await screen.findByText('该故事将支持邮箱登录，覆盖验证码校验场景。')
    ).toBeInTheDocument();
  });

  it('翻译按所选语言调用并展示结果', async () => {
    const user = userEvent.setup();
    mockedAiService.translateStory.mockResolvedValue({
      story_id: 12,
      language: 'en',
      translated: {
        title: 'Support email login',
        description: 'As a registered user...',
        ac: ['Given a valid email'],
      },
      raw: '',
    });

    render(<StoryAIPanel storyId={12} canRefineAC={false} />);

    await user.click(screen.getByRole('button', { name: '翻译' }));

    expect(await screen.findByText('Support email login')).toBeInTheDocument();
    expect(screen.getByText('Given a valid email')).toBeInTheDocument();
    expect(mockedAiService.translateStory).toHaveBeenCalledWith(12, 'en');
  });

  it('AC 优化建议可逐条应用并回调刷新', async () => {
    const user = userEvent.setup();
    mockedAiService.refineAC.mockResolvedValue({
      story_id: 12,
      original_ac: [],
      suggested: ['Given 有效邮箱 When 提交 Then 收到验证码'],
      raw: '',
    });
    mockedStoryService.addAC.mockResolvedValue(undefined);
    const onACChanged = vi.fn();

    render(<StoryAIPanel storyId={12} canRefineAC onACChanged={onACChanged} />);

    await user.type(
      screen.getByPlaceholderText(/描述优化方向/),
      '改成 Given/When/Then 格式'
    );
    await user.click(screen.getByRole('button', { name: '生成建议' }));

    const applyButton = await screen.findByRole('button', { name: '添加' });
    await user.click(applyButton);

    expect(mockedStoryService.addAC).toHaveBeenCalledWith(12, {
      description: 'Given 有效邮箱 When 提交 Then 收到验证码',
    });
    expect(onACChanged).toHaveBeenCalled();
    expect(await screen.findByRole('button', { name: '已添加' })).toBeDisabled();
  });

  it('canRefineAC 为 false 时不渲染 AC 优化入口', () => {
    render(<StoryAIPanel storyId={12} canRefineAC={false} />);

    expect(screen.queryByText('优化验收标准')).not.toBeInTheDocument();
  });

  it('反馈过短时提示且不发起请求', async () => {
    const user = userEvent.setup();
    render(<StoryAIPanel storyId={12} canRefineAC />);

    await user.click(screen.getByRole('button', { name: '生成建议' }));

    expect(showError).toHaveBeenCalledWith('请先描述优化方向（至少 2 个字符）');
    expect(mockedAiService.refineAC).not.toHaveBeenCalled();
  });
});
