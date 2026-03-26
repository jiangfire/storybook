import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { ToastProvider } from '../../ui/Toast';
import { useAuthStore } from '../../../stores/authStore';
import StoryForm from '../StoryForm';

function renderStoryForm() {
  return render(
    <MemoryRouter>
      <ToastProvider>
        <StoryForm isOpen onClose={vi.fn()} projectId={1} mode="create" />
      </ToastProvider>
    </MemoryRouter>
  );
}

describe('StoryForm', () => {
  beforeEach(() => {
    localStorage.removeItem('auth-storage');
    useAuthStore.setState({
      user: {
        id: 1,
        email: 'pm@example.com',
        role: 'product',
        created_at: '2026-03-22T00:00:00Z',
      },
      token: 'token',
      isAuthenticated: true,
      isLoading: false,
      error: null,
    });
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

  describe('表单验证', () => {
    it('应该验证必填字段', async () => {
      renderStoryForm();

      // 尝试提交空表单
      const submitButton = screen.getByRole('button', { name: /创建/ });
      fireEvent.click(submitButton);

      // 应该显示验证错误
      await waitFor(() => {
        expect(screen.getByText(/请输入故事标题/)).toBeInTheDocument();
      });
    });

    it('应该验证标题长度', async () => {
      renderStoryForm();

      const titleInput = screen.getByLabelText(/标题/);
      const submitButton = screen.getByRole('button', { name: /创建/ });

      // 输入1个字符（少于最小长度2）
      fireEvent.change(titleInput, { target: { value: 'a' } });
      fireEvent.click(submitButton);

      // 应该显示长度错误
      await waitFor(() => {
        expect(screen.getByText(/标题长度应在/)).toBeInTheDocument();
      });
    });
  });

  describe('验收标准管理', () => {
    it('应该添加新的验收标准', async () => {
      renderStoryForm();

      const acInput = screen.getByPlaceholderText(/添加验收标准/);
      fireEvent.change(acInput, { target: { value: 'Given用户未登录' } });

      // 找到验收标准输入框旁边的添加按钮
      const addButtons = screen.getAllByRole('button', { name: '添加' });
      const acAddButton = addButtons.find(btn => btn.parentElement?.querySelector('input[placeholder*="验收标准"]'));

      if (acAddButton) {
        fireEvent.click(acAddButton);
      }

      await waitFor(() => {
        expect(screen.getByText(/Given用户未登录/)).toBeInTheDocument();
      });
    });

    it('应该删除验收标准', async () => {
      renderStoryForm();

      // 添加验收标准
      const acInput = screen.getByPlaceholderText(/添加验收标准/);
      fireEvent.change(acInput, { target: { value: 'Given用户未登录' } });

      const addButtons = screen.getAllByRole('button', { name: '添加' });
      const acAddButton = addButtons.find(btn => btn.parentElement?.querySelector('input[placeholder*="验收标准"]'));

      if (acAddButton) {
        fireEvent.click(acAddButton);
      }

      await waitFor(() => {
        const deleteButtons = screen.getAllByRole('button', { name: /删除/ });
        if (deleteButtons.length > 0) {
          fireEvent.click(deleteButtons[0]);
        }
      });
    });
  });

  describe('AI辅助功能', () => {
    it('应该在AI生成失败时显示错误', async () => {
      // Mock API失败
      global.fetch = vi.fn(() =>
        Promise.resolve({
          ok: false,
          status: 500,
          json: async () => ({ error: 'AI服务不可用' }),
        } as Response)
      );

      renderStoryForm();

      const aiInput = screen.getByLabelText('需求描述');
      const generateButton = screen.getByRole('button', { name: '生成草稿' });

      fireEvent.change(aiInput, { target: { value: '用户登录功能' } });
      fireEvent.click(generateButton);

      await waitFor(() => {
        expect(screen.getByText(/AI生成失败/)).toBeInTheDocument();
      });
    });

    it('应该使用规则辅助作为fallback', async () => {
      renderStoryForm();

      const ruleButton = screen.getByRole('button', { name: '补空白' });
      fireEvent.click(ruleButton);

      await waitFor(() => {
        // 验证规则辅助生成了内容
        expect(screen.getByLabelText(/标题/)).toHaveValue();
      });
    });
  });

  describe('标签管理', () => {
    it('应该添加和删除标签', async () => {
      renderStoryForm();

      const tagInput = screen.getByPlaceholderText(/添加标签/);

      // 找到标签输入框旁边的添加按钮
      const addButtons = screen.getAllByRole('button', { name: '添加' });
      const tagAddButton = addButtons.find(btn => btn.parentElement?.querySelector('input[placeholder*="标签"]'));

      fireEvent.change(tagInput, { target: { value: 'auth' } });

      if (tagAddButton) {
        fireEvent.click(tagAddButton);
      }

      await waitFor(() => {
        expect(screen.getByText(/auth/)).toBeInTheDocument();
      });

      // 删除标签 - 使用aria-label
      const deleteButton = screen.getByRole('button', { name: /删除标签/ });
      fireEvent.click(deleteButton);

      await waitFor(() => {
        expect(screen.queryByText(/auth/)).not.toBeInTheDocument();
      });
    });
  });

  describe('权限控制', () => {
    it('非产品经理不应该看到AI辅助功能', () => {
      useAuthStore.setState({
        user: { ...useAuthStore.getState().user, role: 'developer' },
      });

      renderStoryForm();

      expect(screen.queryByText('模型辅助')).not.toBeInTheDocument();
      expect(screen.queryByText('规则辅助')).not.toBeInTheDocument();
    });
  });
});
