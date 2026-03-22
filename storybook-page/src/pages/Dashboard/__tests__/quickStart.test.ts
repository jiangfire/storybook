import { resolveQuickStartProject } from '../quickStart';
import type { Project } from '../../../types/models';

const baseProject: Omit<Project, 'id' | 'name'> = {
  agile_mode: 'kanban',
  owner: {
    id: 1,
    email: 'owner@example.com',
    role: 'product',
    created_at: '2026-03-22T00:00:00Z',
  },
  created_at: '2026-03-22T00:00:00Z',
};

describe('resolveQuickStartProject', () => {
  it('无项目时返回 null', () => {
    expect(resolveQuickStartProject([], null)).toBeNull();
  });

  it('优先返回用户当前选择的项目', () => {
    const projects: Project[] = [
      { id: 1, name: 'A 项目', ...baseProject },
      { id: 2, name: 'B 项目', ...baseProject },
    ];

    expect(resolveQuickStartProject(projects, 2)?.name).toBe('B 项目');
  });

  it('选择失效时回退到第一个项目', () => {
    const projects: Project[] = [
      { id: 3, name: '默认项目', ...baseProject },
      { id: 4, name: '其他项目', ...baseProject },
    ];

    expect(resolveQuickStartProject(projects, 99)?.name).toBe('默认项目');
  });
});
