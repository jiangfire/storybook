import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import StoryCreatePage from '../StoryCreatePage';

vi.mock('../../../components/story/StoryForm', () => ({
  default: ({
    projectId,
    onClose,
  }: {
    projectId: number;
    onClose: () => void;
  }) => (
    <div>
      <span>StoryForm:{projectId}</span>
      <button onClick={onClose}>关闭创建</button>
    </div>
  ),
}));

describe('StoryCreatePage', () => {
  it('非法项目 ID 会提示错误', () => {
    render(
      <MemoryRouter initialEntries={['/projects/abc/stories/new']}>
        <Routes>
          <Route path="/projects/:projectId/stories/new" element={<StoryCreatePage />} />
        </Routes>
      </MemoryRouter>
    );

    expect(screen.getByText('项目ID无效')).toBeInTheDocument();
  });

  it('关闭创建表单后会跳回项目看板', async () => {
    const user = userEvent.setup();

    render(
      <MemoryRouter initialEntries={['/projects/7/stories/new']}>
        <Routes>
          <Route path="/projects/:projectId/stories/new" element={<StoryCreatePage />} />
          <Route path="/projects/:projectId/board" element={<div>Board Page</div>} />
        </Routes>
      </MemoryRouter>
    );

    expect(screen.getByText('StoryForm:7')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '关闭创建' }));

    expect(screen.getByText('Board Page')).toBeInTheDocument();
  });
});
