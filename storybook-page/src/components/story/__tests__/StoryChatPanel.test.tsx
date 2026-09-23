import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import { StoryChatPanel } from '../StoryChatPanel';
import type { AIFormDraft } from '../../../types/api';

vi.mock('../../../services/aiService', () => ({
  aiService: {
    storyChat: vi.fn(),
  },
}));

const { aiService } = await import('../../../services/aiService');
const mockStoryChat = vi.mocked(aiService.storyChat);

const draft: AIFormDraft = {
  title: '手机号验证码登录',
  description: '作为APP用户，我想要用手机号验证码登录，以便免密快速进入',
  story_type: 'feature',
  priority: 3,
  story_points: 5,
  acceptance_criteria: [{ description: 'Given 未登录用户 When 输入正确验证码 Then 登录成功', order: 1 }],
  tags: ['auth'],
};

describe('StoryChatPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('渲染空状态提示，输入为空时发送按钮禁用', () => {
    render(<StoryChatPanel currentDraft={null} onApply={vi.fn()} strategy="replace" />);

    expect(screen.getByText('对话式完善')).toBeInTheDocument();
    expect(screen.getByText(/在下方输入框描述需求/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '发送' })).toBeDisabled();
  });

  it('发送消息后展示 AI 回复，并携带当前草稿作为上下文', async () => {
    const onApply = vi.fn();
    mockStoryChat.mockResolvedValue({ reply: '已生成初稿，要补充异常场景吗？', form_draft: draft, source: 'openai' });
    const user = userEvent.setup();

    render(
      <StoryChatPanel
        currentDraft={null}
        onApply={onApply}
        strategy="replace"
      />
    );

    await user.type(screen.getByLabelText('对话输入'), '做一个手机号验证码登录');
    await user.click(screen.getByRole('button', { name: '发送' }));

    await waitFor(() => {
      expect(aiService.storyChat).toHaveBeenCalledWith({
        messages: [{ role: 'user', content: '做一个手机号验证码登录' }],
        current_draft: null,
      });
    });
    expect(await screen.findByText('已生成初稿，要补充异常场景吗？')).toBeInTheDocument();
  });

  it('点击“应用到表单”回调草稿与策略', async () => {
    const onApply = vi.fn();
    mockStoryChat.mockResolvedValue({ reply: '草稿已更新', form_draft: draft, source: 'openai' });
    const user = userEvent.setup();

    render(<StoryChatPanel currentDraft={null} onApply={onApply} strategy="fill_empty" />);

    await user.type(screen.getByLabelText('对话输入'), '优先级调高');
    await user.click(screen.getByRole('button', { name: '发送' }));

    const applyBtn = await screen.findByRole('button', { name: '应用到表单' });
    await user.click(applyBtn);

    expect(onApply).toHaveBeenCalledWith(draft, 'fill_empty');
    expect(screen.getByText(/已写入下方表单/)).toBeInTheDocument();
  });

  it('请求失败时展示后端错误并恢复输入', async () => {
    const onApply = vi.fn();
    // 模拟 api.ts 拦截器 reject 的包装对象
    mockStoryChat.mockRejectedValue({
      isAxiosError: true,
      response: { data: { message: '请求过于频繁，请稍后重试' } },
      message: '请求过于频繁，请稍后重试',
    });
    const user = userEvent.setup();

    render(<StoryChatPanel currentDraft={null} onApply={onApply} strategy="replace" />);

    await user.type(screen.getByLabelText('对话输入'), '做一个登录功能');
    await user.click(screen.getByRole('button', { name: '发送' }));

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('请求过于频繁，请稍后重试');
    });
    expect(screen.getByLabelText('对话输入')).toHaveValue('做一个登录功能');
    expect(onApply).not.toHaveBeenCalled();
  });

  it('发送过程中显示思考占位并禁用发送', async () => {
    let resolveChat: (value: { reply: string; form_draft: AIFormDraft; source: 'openai' }) => void =
      () => {};
    mockStoryChat.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveChat = resolve;
        })
    );
    const user = userEvent.setup();

    render(<StoryChatPanel currentDraft={null} onApply={vi.fn()} strategy="replace" />);

    await user.type(screen.getByLabelText('对话输入'), '做一个登录功能');
    await user.click(screen.getByRole('button', { name: '发送' }));

    expect(screen.getByText('AI 正在思考…')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /发送中/ })).toBeDisabled();

    resolveChat({ reply: 'done', form_draft: draft, source: 'openai' });
    await waitFor(() => {
      expect(screen.getByText('done')).toBeInTheDocument();
    });
  });
});
