import { render, screen } from '@testing-library/react';
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

  it('保留简洁的 AI 自动填表入口', () => {
    renderStoryForm();

    expect(screen.getByText('AI 自动填表')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '生成草稿' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '补空白' })).toBeInTheDocument();
  });
});
