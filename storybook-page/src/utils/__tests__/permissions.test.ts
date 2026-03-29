import {
  canAssignBug,
  canClaimStory,
  canClaimTask,
  canCreateBug,
  canCreateStory,
  canDeleteTask,
  canEditTask,
  canManageProjectMembers,
  canManageTechLeads,
  canManageTestCases,
  canReceiveBugAssignments,
  canReleaseStory,
  canReleaseTask,
  canUpdateBugStatus,
  canUpdateTaskWorkflow,
  resolveOptionalUserID,
  resolveTaskCreatorID,
} from '../permissions';

describe('permissions', () => {
  it('uses project ownership for member management', () => {
    expect(canManageProjectMembers({ is_owner: true })).toBe(true);
    expect(canManageProjectMembers({ is_owner: false })).toBe(false);
    expect(canManageProjectMembers(null)).toBe(false);
  });

  it('keeps admin-only tech lead management and story claim parity', () => {
    expect(canManageTechLeads('admin')).toBe(true);
    expect(canManageTechLeads('product')).toBe(false);
    expect(canCreateStory('product')).toBe(true);
    expect(canCreateStory('developer')).toBe(false);
    expect(canClaimStory('admin')).toBe(true);
    expect(canClaimStory('developer')).toBe(true);
    expect(canClaimStory('tester')).toBe(false);
    expect(canReleaseStory({ id: 9, role: 'admin' }, { id: 7 })).toBe(true);
    expect(canReleaseStory({ id: 9, role: 'product' }, { id: 7 })).toBe(true);
    expect(canReleaseStory({ id: 9, role: 'developer' }, { id: 9 })).toBe(true);
    expect(canReleaseStory({ id: 9, role: 'developer' }, { id: 8 })).toBe(false);
  });

  it('matches bug permissions and assignable roles', () => {
    expect(canCreateBug('tester')).toBe(true);
    expect(canCreateBug('product')).toBe(false);
    expect(canUpdateBugStatus('developer')).toBe(true);
    expect(canUpdateBugStatus('product')).toBe(false);
    expect(canAssignBug('admin')).toBe(true);
    expect(canAssignBug('tester')).toBe(false);
    expect(canReceiveBugAssignments('developer')).toBe(true);
    expect(canReceiveBugAssignments('admin')).toBe(true);
    expect(canReceiveBugAssignments('tester')).toBe(false);
  });

  it('resolves optional user ids from number and object payloads', () => {
    expect(resolveOptionalUserID(7)).toBe(7);
    expect(resolveOptionalUserID({ id: 9 })).toBe(9);
    expect(resolveOptionalUserID(undefined)).toBeNull();
  });

  it('evaluates task permissions from creator and assignee state', () => {
    const developer = { id: 11, role: 'developer' as const };
    const product = { id: 22, role: 'product' as const };
    const tester = { id: 33, role: 'tester' as const };
    const task = { created_by: { id: 11, email: 'dev@example.com' } };

    expect(resolveTaskCreatorID(task)).toBe(11);
    expect(canEditTask(developer, task)).toBe(true);
    expect(canEditTask(product, task)).toBe(true);
    expect(canDeleteTask(tester, task)).toBe(false);

    expect(canUpdateTaskWorkflow(product, { id: 99 })).toBe(true);
    expect(canUpdateTaskWorkflow(developer, { id: 11 })).toBe(true);
    expect(canUpdateTaskWorkflow(tester, { id: 11 })).toBe(false);

    expect(canClaimTask(developer, null)).toBe(true);
    expect(canClaimTask(product, null)).toBe(false);
    expect(canReleaseTask(product, { id: 44 })).toBe(true);
    expect(canReleaseTask(developer, { id: 11 })).toBe(true);
    expect(canReleaseTask(developer, { id: 55 })).toBe(false);
  });

  it('restricts test case management to tester and admin', () => {
    expect(canManageTestCases('tester')).toBe(true);
    expect(canManageTestCases('admin')).toBe(true);
    expect(canManageTestCases('developer')).toBe(false);
  });
});
