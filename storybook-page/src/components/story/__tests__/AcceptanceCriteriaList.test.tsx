import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useStoryStore } from '../../../stores/storyStore';
import { storyService } from '../../../services/storyService';
import AcceptanceCriteriaList from '../AcceptanceCriteriaList';

vi.mock('../../../stores/storyStore', async () => {
  const actual = await vi.importActual<typeof import('../../../stores/storyStore')>(
    '../../../stores/storyStore'
  );
  return actual;
});

vi.mock('../../../services/storyService', () => ({
  storyService: {
    addAC: vi.fn(),
    updateAC: vi.fn(),
    deleteAC: vi.fn(),
  },
}));

vi.mock('../../ui/Toast', () => ({
  useToast: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}));

const mockedStoryService = vi.mocked(storyService, { deep: true });

const oneAC = [
  {
    id: 'ac-1',
    description: 'Given 用户已登录',
    ref: 'REQ-1',
    status: 'pending' as const,
    order: 1,
  },
];

describe('AcceptanceCriteriaList', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useStoryStore.setState({
      updateACStatus: vi.fn(async () => {}),
      isUpdating: false,
    });
  });

  it('点击状态按钮会更新 AC 状态', async () => {
    const user = userEvent.setup();
    const updateACStatus = vi.fn(async () => {});
    useStoryStore.setState({ updateACStatus });

    render(<AcceptanceCriteriaList storyId={12} criteria={oneAC} />);

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
        criteria={[{ ...oneAC[0], status: 'passed' as const, ref: undefined }]}
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

  it('contentEditable 下可新增验收标准并回调刷新', async () => {
    const user = userEvent.setup();
    mockedStoryService.addAC.mockResolvedValue(undefined);
    const onChanged = vi.fn();

    render(
      <AcceptanceCriteriaList storyId={12} criteria={[]} contentEditable onChanged={onChanged} />
    );

    await user.click(screen.getByRole('button', { name: '+ 新增验收标准' }));
    await user.type(
      screen.getByPlaceholderText('验收标准描述（如：用户输入手机号后可收到验证码）'),
      '用户可用手机号登录'
    );
    await user.click(screen.getByRole('button', { name: '保存' }));

    expect(mockedStoryService.addAC).toHaveBeenCalledWith(12, {
      description: '用户可用手机号登录',
      ref: undefined,
      notes: undefined,
    });
    expect(onChanged).toHaveBeenCalled();
  });

  it('contentEditable 下可编辑验收标准内容', async () => {
    const user = userEvent.setup();
    mockedStoryService.updateAC.mockResolvedValue(undefined);
    const onChanged = vi.fn();

    render(
      <AcceptanceCriteriaList storyId={12} criteria={oneAC} contentEditable onChanged={onChanged} />
    );

    await user.click(screen.getByRole('button', { name: '编辑' }));
    const textarea = screen.getByPlaceholderText(
      '验收标准描述（如：用户输入手机号后可收到验证码）'
    );
    await user.clear(textarea);
    await user.type(textarea, '更严格的标准');
    await user.click(screen.getByRole('button', { name: '保存' }));

    expect(mockedStoryService.updateAC).toHaveBeenCalledWith(12, 'ac-1', {
      description: '更严格的标准',
      ref: 'REQ-1',
      notes: '',
    });
    expect(onChanged).toHaveBeenCalled();
  });

  it('contentEditable 下清空引用/备注后保存发送空串以清除字段', async () => {
    const user = userEvent.setup();
    mockedStoryService.updateAC.mockResolvedValue(undefined);

    const withExtras = [
      {
        id: 'ac-1',
        description: 'Given 用户已登录',
        ref: 'REQ-1',
        notes: '旧备注',
        status: 'pending' as const,
        order: 1,
      },
    ];
    render(<AcceptanceCriteriaList storyId={12} criteria={withExtras} contentEditable />);

    await user.click(screen.getByRole('button', { name: '编辑' }));
    await user.clear(screen.getByPlaceholderText('引用（可选，如 REQ-101）'));
    await user.clear(screen.getByPlaceholderText('备注（可选）'));
    await user.click(screen.getByRole('button', { name: '保存' }));

    expect(mockedStoryService.updateAC).toHaveBeenCalledWith(12, 'ac-1', {
      description: 'Given 用户已登录',
      ref: '',
      notes: '',
    });
  });

  it('contentEditable 下删除验收标准前需要确认', async () => {
    const user = userEvent.setup();
    mockedStoryService.deleteAC.mockResolvedValue(undefined);
    const onChanged = vi.fn();
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValueOnce(false);

    render(
      <AcceptanceCriteriaList storyId={12} criteria={oneAC} contentEditable onChanged={onChanged} />
    );

    await user.click(screen.getByRole('button', { name: '删除' }));
    expect(confirmSpy).toHaveBeenCalled();
    expect(mockedStoryService.deleteAC).not.toHaveBeenCalled();

    confirmSpy.mockReturnValueOnce(true);
    await user.click(screen.getByRole('button', { name: '删除' }));

    expect(mockedStoryService.deleteAC).toHaveBeenCalledWith(12, 'ac-1');
    expect(onChanged).toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it('默认不渲染内容编辑入口', () => {
    render(<AcceptanceCriteriaList storyId={12} criteria={oneAC} />);

    expect(screen.queryByRole('button', { name: '+ 新增验收标准' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '编辑' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '删除' })).not.toBeInTheDocument();
  });
});
