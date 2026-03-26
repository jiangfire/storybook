import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import { AICreator } from '../AICreator';

// Mock AI service
vi.mock('../../../services/aiService', () => ({
  aiService: {
    generateStory: vi.fn(),
  },
}));

const { aiService } = await import('../../../services/aiService');

describe('AICreator', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('基本渲染', () => {
    it('应该显示AI输入区域', () => {
      render(<AICreator onGenerated={vi.fn()} />);

      expect(screen.getByLabelText('需求描述')).toBeInTheDocument();
      expect(screen.getByText('模型辅助')).toBeInTheDocument();
      expect(screen.getByText('规则辅助')).toBeInTheDocument();
    });

    it('应该显示生成草稿和补空白按钮', () => {
      render(<AICreator onGenerated={vi.fn()} />);

      expect(screen.getByRole('button', { name: '生成草稿' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: '补空白' })).toBeInTheDocument();
    });
  });

  describe('AI生成功能', () => {
    it('应该调用generateStory并传递正确参数', async () => {
      const onGenerated = vi.fn();
      (aiService.generateStory as any).mockResolvedValue({
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
            { description: 'Given用户未登录', order: 0 },
            { description: 'When输入账号密码', order: 1 },
            { description: 'Then登录成功', order: 2 },
          ],
          tags: ['auth'],
        },
      });

      render(<AICreator onGenerated={onGenerated} />);

      const input = screen.getByLabelText('需求描述');
      const generateBtn = screen.getByRole('button', { name: '生成草稿' });

      await userEvent.type(input, '实现用户登录功能');
      fireEvent.click(generateBtn);

      await waitFor(() => {
        expect(aiService.generateStory).toHaveBeenCalled();
      });
    });

    it('应该支持replace策略', async () => {
      const onGenerated = vi.fn();
      (aiService.generateStory as any).mockResolvedValue({
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
            { description: 'Given用户未登录', order: 0 },
            { description: 'When输入账号密码', order: 1 },
            { description: 'Then登录成功', order: 2 },
          ],
          tags: ['auth'],
        },
      });

      render(<AICreator onGenerated={onGenerated} strategy="replace" />);

      const input = screen.getByLabelText('需求描述');
      const generateBtn = screen.getByRole('button', { name: '生成草稿' });

      await userEvent.type(input, '实现登录');
      fireEvent.click(generateBtn);

      await waitFor(() => {
        expect(onGenerated).toHaveBeenCalledWith(
          expect.objectContaining({
            title: '用户登录',
            story_type: 'feature',
            priority: 2,
            acceptance_criteria: expect.any(Array),
            tags: expect.any(Array),
          }),
          'replace'
        );
      });
    });

    it('应该支持fill_empty策略', async () => {
      const onGenerated = vi.fn();
      (aiService.generateStory as any).mockResolvedValue({
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
            { description: 'Given用户未登录', order: 0 },
            { description: 'When输入账号密码', order: 1 },
            { description: 'Then登录成功', order: 2 },
          ],
          tags: ['auth'],
        },
      });

      render(<AICreator onGenerated={onGenerated} strategy="fill_empty" />);

      const input = screen.getByLabelText('需求描述');
      const generateBtn = screen.getByRole('button', { name: '补空白' });

      await userEvent.type(input, '实现登录');
      fireEvent.click(generateBtn);

      await waitFor(() => {
        expect(onGenerated).toHaveBeenCalledWith(
          expect.objectContaining({
            story_type: 'feature',
          }),
          'fill_empty'
        );
      });
    });

    it('应该显示生成中的加载状态', async () => {
      const onGenerated = vi.fn();
      (aiService.generateStory as any).mockImplementation(
        () => new Promise((resolve) => setTimeout(() => resolve({
          title: '用户登录',
        }), 100))
      );

      render(<AICreator onGenerated={onGenerated} />);

      const input = screen.getByLabelText('需求描述');
      const generateBtn = screen.getByRole('button', { name: '生成草稿' });

      await userEvent.type(input, '实现登录');
      fireEvent.click(generateBtn);

      expect(screen.getByRole('button', { name: /生成中/ })).toBeInTheDocument();
    });

    it('应该在AI生成失败时显示错误', async () => {
      const onGenerated = vi.fn();
      (aiService.generateStory as any).mockRejectedValue(
        new Error('AI服务不可用')
      );

      render(<AICreator onGenerated={onGenerated} />);

      const input = screen.getByLabelText('需求描述');
      const generateBtn = screen.getByRole('button', { name: '生成草稿' });

      await userEvent.type(input, '实现登录');
      fireEvent.click(generateBtn);

      await waitFor(() => {
        expect(screen.getByText(/AI生成失败/)).toBeInTheDocument();
      });
    });
  });

  describe('规则辅助', () => {
    it('应该在点击补空白时生成规则草案', async () => {
      const onGenerated = vi.fn();

      render(<AICreator onGenerated={onGenerated} />);

      const ruleBtn = screen.getByRole('button', { name: '补空白' });
      fireEvent.click(ruleBtn);

      await waitFor(() => {
        expect(onGenerated).toHaveBeenCalledWith(
          expect.objectContaining({
            title: expect.any(String),
            description: expect.any(String),
            story_type: 'feature',
            priority: 2,
          }),
          'fill_empty'
        );
      });
    });
  });

  describe('边界条件', () => {
    it('空输入时禁用生成按钮', () => {
      render(<AICreator onGenerated={vi.fn()} />);

      const generateBtn = screen.getByRole('button', { name: '生成草稿' });
      expect(generateBtn).toBeDisabled();
    });

    it('输入后启用生成按钮', async () => {
      render(<AICreator onGenerated={vi.fn()} />);

      const input = screen.getByLabelText('需求描述');
      await userEvent.type(input, '用户登录');

      const generateBtn = screen.getByRole('button', { name: '生成草稿' });
      await waitFor(() => {
        expect(generateBtn).not.toBeDisabled();
      });
    });

    it('应该限制输入长度', async () => {
      render(<AICreator onGenerated={vi.fn()} />);

      const input = screen.getByLabelText('需求描述') as HTMLTextAreaElement;
      const maxLength = 2000;

      expect(input.maxLength).toBe(maxLength);
    });
  });
});
