import { render, screen } from '@testing-library/react';
import { SprintManagementSection } from '../SprintManagementSection';
import type { SprintSummary } from '../../../../types/api';

const plannedSprint: SprintSummary = {
  id: 7,
  project_id: 1,
  name: 'Sprint 1',
  goal: '',
  start_date: '2026-04-01',
  end_date: '2026-04-14',
  status: 'planned',
  total_stories: 3,
  done_stories: 0,
  created_at: '2026-03-29T00:00:00Z',
  updated_at: '2026-03-29T00:00:00Z',
};

function renderSection(props: Partial<Parameters<typeof SprintManagementSection>[0]> & Pick<Parameters<typeof SprintManagementSection>[0], 'canManage'>) {
  return render(
    <SprintManagementSection
      sprintError=""
      sprints={[plannedSprint]}
      statusUpdatingSprintID={null}
      onCreateSprint={vi.fn()}
      onSelectSprint={vi.fn()}
      onUpdateSprintStatus={vi.fn()}
      onCancelSprint={vi.fn()}
      onDeleteSprint={vi.fn()}
      {...props}
    />
  );
}

describe('SprintManagementSection', () => {
  it('canManage 时渲染新建/取消/删除冲刺入口', () => {
    renderSection({ canManage: true });

    expect(screen.getByRole('button', { name: '+ 新建冲刺' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '删除冲刺' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '取消冲刺' })).toBeInTheDocument();
  });

  it('非 PM/admin 不渲染冲刺管理操作入口（后端仅 PM/admin，避免点了 403）', () => {
    renderSection({ canManage: false });

    expect(screen.queryByRole('button', { name: '+ 新建冲刺' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '删除冲刺' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '取消冲刺' })).not.toBeInTheDocument();
    // 只读操作保留
    expect(screen.getByRole('button', { name: '查看燃尽图' })).toBeInTheDocument();
  });
});
