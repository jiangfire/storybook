import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import { AICreator } from '../AICreator';
import type { AIGeneratedStoryResponse } from '../../../types/api';

// Mock AI service
vi.mock('../../../services/aiService', () => ({
  aiService: {
    generateStory: vi.fn(),
  },
}));

const { aiService } = await import('../../../services/aiService');

const mockGenerateStory = vi.mocked(aiService.generateStory);

const buildGeneratedStoryResponse = (
  overrides: Partial<AIGeneratedStoryResponse> = {}
): AIGeneratedStoryResponse => ({
  title: '用户登录',
  user_story: '实现登录功能',
  actor: '用户',
  action: '登录',
  value: '安全访问数据',
  story_type: 'feature',
  priority: 2,
  suggested_ac: ['Given用户未登录', 'When输入账号密码', 'Then登录成功'],
  story_points: 3,
  tags: ['auth'],
  warnings: [],
  source: 'openai',
  is_configured: true,
  form_draft: {
    title: '用户登录',
    description: '实现登录功能',
    story_type: 'feature',
    priority: 2,
    story_points: 3,
    acceptance_criteria: [
      { description: 'Given用户未登录', order: 1 },
      { description: 'When输入账号密码', order: 2 },
      { description: 'Then登录成功', order: 3 },
    ],
    tags: ['auth'],
  },
  ...overrides,
});

describe('AICreator', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('基本渲染', () => {
    it('应该显示一句话创建入口和需求输入', () => {
      render(<AICreator onGenerated={vi.fn()} />);

      expect(screen.getByText('一句话创建故事')).toBeInTheDocument();
      expect(screen.getByLabelText('需求描述')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: '生成完整草稿' })).toBeInTheDocument();
    });

    it('空输入或不足 5 字时禁用生成按钮', () => {
      render(<AICreator onGenerated={vi.fn()} />);

      const generateBtn = screen.getByRole('button', { name: '生成完整草稿' });
      expect(generateBtn).toBeDisabled();
    });

    it('输入达到最小长度后启用生成按钮', async () => {
      const user = userEvent.setup();
      render(<AICreator onGenerated={vi.fn()} />);

      await user.type(screen.getByLabelText('需求描述'), '实现用户登录功能');

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '生成完整草稿' })).not.toBeDisabled();
      });
    });

    it('应该限制输入长度', () => {
      render(<AICreator onGenerated={vi.fn()} />);

      const input = screen.getByLabelText('需求描述') as HTMLTextAreaElement;
      expect(input.maxLength).toBe(2000);
    });
  });

  describe('AI生成功能', () => {
    it('应该调用generateStory并把草稿按策略回调', async () => {
      const onGenerated = vi.fn();
      const response = buildGeneratedStoryResponse();
      mockGenerateStory.mockResolvedValue(response);
      const user = userEvent.setup();

      render(<AICreator onGenerated={onGenerated} strategy="replace" />);

      await user.type(screen.getByLabelText('需求描述'), '实现用户登录功能');
      await user.click(screen.getByRole('button', { name: '生成完整草稿' }));

      await waitFor(() => {
        expect(aiService.generateStory).toHaveBeenCalledWith({ requirement: '实现用户登录功能' });
        expect(onGenerated).toHaveBeenCalledWith(response.form_draft, 'replace');
      });
    });

    it('fill_empty 策略会透传给回调', async () => {
      const onGenerated = vi.fn();
      const response = buildGeneratedStoryResponse();
      mockGenerateStory.mockResolvedValue(response);
      const user = userEvent.setup();

      render(<AICreator onGenerated={onGenerated} strategy="fill_empty" />);

      await user.type(screen.getByLabelText('需求描述'), '实现用户登录功能');
      await user.click(screen.getByRole('button', { name: '生成完整草稿' }));

      await waitFor(() => {
        expect(onGenerated).toHaveBeenCalledWith(response.form_draft, 'fill_empty');
      });
    });

    it('应该显示生成中的加载状态', async () => {
      const onGenerated = vi.fn();
      mockGenerateStory.mockImplementation(
        () =>
          new Promise((resolve) => setTimeout(() => resolve(buildGeneratedStoryResponse()), 100))
      );
      const user = userEvent.setup();

      render(<AICreator onGenerated={onGenerated} />);

      await user.type(screen.getByLabelText('需求描述'), '实现用户登录功能');
      await user.click(screen.getByRole('button', { name: '生成完整草稿' }));

      expect(screen.getByRole('button', { name: /生成中/ })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /生成中/ })).toBeDisabled();
    });

    it('生成成功后显示来源与填写提示', async () => {
      const onGenerated = vi.fn();
      mockGenerateStory.mockResolvedValue(buildGeneratedStoryResponse({ source: 'openai' }));
      const user = userEvent.setup();

      render(<AICreator onGenerated={onGenerated} />);

      await user.type(screen.getByLabelText('需求描述'), '实现用户登录功能');
      await user.click(screen.getByRole('button', { name: '生成完整草稿' }));

      expect(await screen.findByText('来源：AI 模型生成')).toBeInTheDocument();
      expect(screen.getByText(/草稿已填入下方表单/)).toBeInTheDocument();
    });

    it('规则降级时显示规则草稿来源标识', async () => {
      const onGenerated = vi.fn();
      mockGenerateStory.mockResolvedValue(
        buildGeneratedStoryResponse({
          source: 'heuristic',
          warnings: ['OpenAI 调用失败，已自动回退到规则草稿，请检查 AI 配置或稍后重试'],
        })
      );
      const user = userEvent.setup();

      render(<AICreator onGenerated={onGenerated} />);

      await user.type(screen.getByLabelText('需求描述'), '实现用户登录功能');
      await user.click(screen.getByRole('button', { name: '生成完整草稿' }));

      expect(await screen.findByText('来源：规则草稿（未调用大模型）')).toBeInTheDocument();
      expect(screen.getByText(/已自动回退到规则草稿/)).toBeInTheDocument();
    });
  });

  describe('错误处理', () => {
    it('输入不足 5 字时提示且不调用接口', async () => {
      const onGenerated = vi.fn();
      const user = userEvent.setup();

      render(<AICreator onGenerated={onGenerated} />);

      await user.type(screen.getByLabelText('需求描述'), '登录');
      // 按钮此时应已禁用，直接尝试提交
      await user.click(screen.getByRole('button', { name: '生成完整草稿' }));

      expect(aiService.generateStory).not.toHaveBeenCalled();
      expect(onGenerated).not.toHaveBeenCalled();
    });

    it('请求被拒绝为拦截器包装对象时也能展示后端错误信息', async () => {
      const onGenerated = vi.fn();
      // 模拟 api.ts 响应拦截器 reject 的对象：展开 AxiosError 后覆盖 message（非 Error 实例，但保留 isAxiosError/response）
      mockGenerateStory.mockRejectedValue({
        isAxiosError: true,
        response: { data: { message: '请求过于频繁，请稍后重试' } },
        message: '请求过于频繁，请稍后重试',
      });
      const user = userEvent.setup();

      render(<AICreator onGenerated={onGenerated} />);

      await user.type(screen.getByLabelText('需求描述'), '实现用户登录功能');
      await user.click(screen.getByRole('button', { name: '生成完整草稿' }));

      await waitFor(() => {
        expect(screen.getByRole('alert')).toHaveTextContent('请求过于频繁，请稍后重试');
      });
      expect(screen.queryByText(/AI生成失败: AI生成失败/)).not.toBeInTheDocument();
    });

    it('请求完全失败时展示兜底文案', async () => {
      const onGenerated = vi.fn();
      mockGenerateStory.mockRejectedValue(undefined);
      const user = userEvent.setup();

      render(<AICreator onGenerated={onGenerated} />);

      await user.type(screen.getByLabelText('需求描述'), '实现用户登录功能');
      await user.click(screen.getByRole('button', { name: '生成完整草稿' }));

      await waitFor(() => {
        expect(screen.getByRole('alert')).toHaveTextContent('AI 生成失败，请稍后重试');
      });
    });
  });
});
