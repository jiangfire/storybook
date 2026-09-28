import type { SprintSummary } from '../../../types/api';

export function getSprintStatusClass(status: SprintSummary['status']) {
  switch (status) {
    case 'active':
      return 'bg-green-100 text-green-700';
    case 'completed':
      return 'bg-secondary-200 text-text-light';
    case 'cancelled':
      return 'bg-gray-100 text-gray-500';
    default:
      return 'bg-blue-100 text-blue-700';
  }
}

export type SprintActionKind = 'activate' | 'close';

export function getNextSprintAction(status: SprintSummary['status']) {
  if (status === 'planned') {
    return { label: '开始冲刺', kind: 'activate' as const, target: 'active' as const };
  }
  if (status === 'active') {
    // 完成走专用 close 端点，后端会事务性把未完成故事退回待办池。
    return { label: '完成冲刺', kind: 'close' as const, target: 'completed' as const };
  }
  if (status === 'completed') {
    return { label: '重新激活', kind: 'activate' as const, target: 'active' as const };
  }
  return null;
}

export function canCancelSprint(status: SprintSummary['status']) {
  return status === 'planned' || status === 'active';
}
