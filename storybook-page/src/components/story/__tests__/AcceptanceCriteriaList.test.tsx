import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useStoryStore } from '../../../stores/storyStore';
import AcceptanceCriteriaList from '../AcceptanceCriteriaList';

vi.mock('../../../stores/storyStore', async () => {
  const actual = await vi.importActual<typeof import('../../../stores/storyStore')>(
    '../../../stores/storyStore'
  );
  return actual;
});

describe('AcceptanceCriteriaList', () => {
  beforeEach(() => {
    useStoryStore.setState({
      updateACStatus: vi.fn(async () => {}),
      isUpdating: false,
    });
  });

  it('点击状态按钮会更新 AC 状态', async () => {
    const user = userEvent.setup();
    const updateACStatus = vi.fn(async () => {});
    useStoryStore.setState({ updateACStatus });

    render(
      <AcceptanceCriteriaList
        storyId={12}
        criteria={[
          {
            id: 'ac-1',
            description: 'Given 用户已登录',
            status: 'pending',
            order: 1,
          },
        ]}
      />
    );

    await user.click(screen.getByRole('button', { name: '过' }));

    expect(updateACStatus).toHaveBeenCalledWith(12, 'ac-1', 'passed', '');
  });

  it('通过项可补充证据并保存', async () => {
    const user = userEvent.setup();
    const updateACStatus = vi.fn(async () => {});
    useStoryStore.setState({ updateACStatus });

    render(
      <AcceptanceCriteriaList
        storyId={12}
        criteria={[
          {
            id: 'ac-1',
            description: 'Given 用户已登录',
            status: 'passed',
            order: 1,
          },
        ]}
      />
    );

    await user.click(screen.getByRole('button', { name: '添加证据' }));
    await user.type(
      screen.getByPlaceholderText('添加证据（如代码路径、测试截图等）...'),
      'src/login.ts:42'
    );
    await user.click(screen.getByRole('button', { name: '保存' }));

    expect(updateACStatus).toHaveBeenCalledWith(12, 'ac-1', 'passed', 'src/login.ts:42');
  });

  it('只读模式下显示状态徽标，不显示编辑按钮', () => {
    render(
      <AcceptanceCriteriaList
        storyId={12}
        editable={false}
        criteria={[
          {
            id: 'ac-1',
            description: 'Given 用户已登录',
            status: 'failed',
            order: 1,
          },
        ]}
      />
    );

    expect(screen.getByText('Given 用户已登录')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '待' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '添加证据' })).not.toBeInTheDocument();
  });
});
