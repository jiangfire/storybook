import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { projectService } from '../../../../services/projectService';
import { ProjectSettingsModal } from '../ProjectSettingsModal';
import type { Project } from '../../../../types/models';

const showSuccess = vi.fn();
const showError = vi.fn();

vi.mock('../../../../components/ui/Toast', () => ({
  useToast: () => ({
    showSuccess,
    showError,
  }),
}));

vi.mock('../../../../services/projectService', () => ({
  projectService: {
    updateProject: vi.fn(),
    deleteProject: vi.fn(),
    archiveProject: vi.fn(),
    unarchiveProject: vi.fn(),
    exportProject: vi.fn(),
  },
}));

const mockedProjectService = vi.mocked(projectService, { deep: true });

const baseProject: Project = {
  id: 7,
  name: '门户重构',
  description: '旧描述',
  agile_mode: 'kanban',
  owner: { id: 1, email: 'owner@example.com', role: 'product', created_at: '2026-01-01T00:00:00Z' },
  created_at: '2026-01-01T00:00:00Z',
};

function renderModal(props: Partial<Parameters<typeof ProjectSettingsModal>[0]> = {}) {
  const onClose = vi.fn();
  const onSaved = vi.fn();
  const onDeleted = vi.fn();
  render(
    <ProjectSettingsModal
      isOpen
      onClose={onClose}
      project={baseProject}
      onSaved={onSaved}
      onDeleted={onDeleted}
      {...props}
    />
  );
  return { onClose, onSaved, onDeleted };
}

describe('ProjectSettingsModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedProjectService.updateProject.mockResolvedValue(baseProject);
    mockedProjectService.exportProject.mockResolvedValue({ project: baseProject });
  });

  it('保存设置会提交名称/描述/敏捷模式并刷新', async () => {
    const user = userEvent.setup();
    const { onClose, onSaved } = renderModal();

    const nameInput = screen.getByPlaceholderText('项目名称');
    await user.clear(nameInput);
    await user.type(nameInput, '新门户');
    await user.click(screen.getByRole('button', { name: '保存设置' }));

    expect(mockedProjectService.updateProject).toHaveBeenCalledWith(7, {
      name: '新门户',
      description: '旧描述',
      agile_mode: 'kanban',
    });
    expect(onSaved).toHaveBeenCalled();
    expect(onClose).toHaveBeenCalled();
    expect(showSuccess).toHaveBeenCalledWith('项目已更新');
  });

  it('名称为空时不提交并提示', async () => {
    const user = userEvent.setup();
    renderModal();

    const nameInput = screen.getByPlaceholderText('项目名称');
    await user.clear(nameInput);
    await user.click(screen.getByRole('button', { name: '保存设置' }));

    expect(showError).toHaveBeenCalledWith('项目名称不能为空');
    expect(mockedProjectService.updateProject).not.toHaveBeenCalled();
  });

  it('归档需确认，成功后刷新', async () => {
    const user = userEvent.setup();
    mockedProjectService.archiveProject.mockResolvedValue(undefined);
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValueOnce(false);
    const { onSaved } = renderModal();

    await user.click(screen.getByRole('button', { name: '归档项目' }));
    expect(mockedProjectService.archiveProject).not.toHaveBeenCalled();

    confirmSpy.mockReturnValueOnce(true);
    await user.click(screen.getByRole('button', { name: '归档项目' }));

    expect(mockedProjectService.archiveProject).toHaveBeenCalledWith(7);
    expect(onSaved).toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it('已归档项目显示还原入口', async () => {
    const user = userEvent.setup();
    mockedProjectService.unarchiveProject.mockResolvedValue(undefined);
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValueOnce(true);
    renderModal({ project: { ...baseProject, archived: true } });

    await user.click(screen.getByRole('button', { name: '还原归档' }));

    expect(mockedProjectService.unarchiveProject).toHaveBeenCalledWith(7);
    confirmSpy.mockRestore();
  });

  it('导出会拉取快照', async () => {
    const user = userEvent.setup();
    renderModal();

    await user.click(screen.getByRole('button', { name: '导出 JSON 快照' }));

    expect(mockedProjectService.exportProject).toHaveBeenCalledWith(7);
    expect(showSuccess).toHaveBeenCalledWith('项目快照已导出');
  });

  it('删除需确认，成功后通知父级离开', async () => {
    const user = userEvent.setup();
    mockedProjectService.deleteProject.mockResolvedValue(undefined);
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValueOnce(false);
    const { onDeleted } = renderModal();

    await user.click(screen.getByRole('button', { name: '删除项目' }));
    expect(mockedProjectService.deleteProject).not.toHaveBeenCalled();

    confirmSpy.mockReturnValueOnce(true);
    await user.click(screen.getByRole('button', { name: '删除项目' }));

    expect(mockedProjectService.deleteProject).toHaveBeenCalledWith(7);
    expect(onDeleted).toHaveBeenCalled();
    confirmSpy.mockRestore();
  });
});
