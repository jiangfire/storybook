import type { TaskItem } from '../types/api';
import type { Project, User, UserRole } from '../types/models';

type AuthUser = Pick<User, 'id' | 'role'> | null | undefined;
type OptionalUserRef = { id: number } | number | null | undefined;

export function canManageProjectMembers(project?: Pick<Project, 'is_owner'> | null): boolean {
  return Boolean(project?.is_owner);
}

export function canManageTechLeads(role?: UserRole): boolean {
  return role === 'admin';
}

export function canCreateStory(role?: UserRole): boolean {
  return role === 'product' || role === 'admin';
}

export function canManageStoryAssignee(role?: UserRole): boolean {
  return role === 'product' || role === 'tech_lead' || role === 'admin';
}

export function canUseStoryAI(role?: UserRole): boolean {
  return canCreateStory(role);
}

export function canClaimStory(role?: UserRole): boolean {
  return role === 'developer' || role === 'admin';
}

export function canReleaseStory(user: AuthUser, assignedTo: OptionalUserRef): boolean {
  const assignedToID = resolveOptionalUserID(assignedTo);
  if (!user || assignedToID === null) {
    return false;
  }
  return user.role === 'product' || user.role === 'admin' || assignedToID === user.id;
}

export function canCreateBug(role?: UserRole): boolean {
  return role === 'tester' || role === 'admin';
}

export function canUpdateBugStatus(role?: UserRole): boolean {
  return role === 'developer' || role === 'tester' || role === 'admin';
}

export function canAssignBug(role?: UserRole): boolean {
  return role === 'product' || role === 'admin';
}

export function canReceiveBugAssignments(projectRole?: string): boolean {
  return projectRole === 'developer' || projectRole === 'admin';
}

export function canCreateTask(role?: UserRole): boolean {
  return role === 'product' || role === 'developer' || role === 'admin';
}

export function canSplitStoryTasks(role?: UserRole): boolean {
  return role === 'product' || role === 'admin';
}

export function canManageTestCases(role?: UserRole): boolean {
  return role === 'tester' || role === 'admin';
}

export function canAddTaskCodeReference(role?: UserRole): boolean {
  return role === 'developer' || role === 'admin';
}

export function resolveOptionalUserID(value: OptionalUserRef): number | null {
  if (typeof value === 'number') {
    return value;
  }
  if (value && typeof value.id === 'number') {
    return value.id;
  }
  return null;
}

export function resolveTaskCreatorID(task: Pick<TaskItem, 'created_by'>): number | null {
  return resolveOptionalUserID(task.created_by);
}

export function canEditTask(user: AuthUser, task: Pick<TaskItem, 'created_by'>): boolean {
  if (!user) {
    return false;
  }
  return (
    user.role === 'product' || user.role === 'admin' || resolveTaskCreatorID(task) === user.id
  );
}

export function canDeleteTask(user: AuthUser, task: Pick<TaskItem, 'created_by'>): boolean {
  return canEditTask(user, task);
}

export function canUpdateTaskWorkflow(user: AuthUser, assignedTo: OptionalUserRef): boolean {
  const assignedToID = resolveOptionalUserID(assignedTo);
  if (!user) {
    return false;
  }
  return (
    user.role === 'product' ||
    user.role === 'admin' ||
    (assignedToID !== null && assignedToID === user.id)
  );
}

export function canClaimTask(user: AuthUser, assignedTo: OptionalUserRef): boolean {
  if (!user || resolveOptionalUserID(assignedTo) !== null) {
    return false;
  }
  return user.role === 'developer' || user.role === 'admin';
}

export function canReleaseTask(user: AuthUser, assignedTo: OptionalUserRef): boolean {
  const assignedToID = resolveOptionalUserID(assignedTo);
  if (!user || assignedToID === null) {
    return false;
  }
  return user.role === 'product' || user.role === 'admin' || assignedToID === user.id;
}
