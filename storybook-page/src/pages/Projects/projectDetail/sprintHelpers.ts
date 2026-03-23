import type { SprintSummary } from '../../../types/api';

export function getSprintStatusClass(status: SprintSummary['status']) {
  switch (status) {
    case 'active':
      return 'bg-green-100 text-green-700';
    case 'completed':
      return 'bg-secondary-200 text-text-light';
    default:
      return 'bg-blue-100 text-blue-700';
  }
}

export function getNextSprintAction(status: SprintSummary['status']) {
  if (status === 'planned') {
    return { label: '开始冲刺', target: 'active' as const };
  }
  if (status === 'active') {
    return { label: '完成冲刺', target: 'completed' as const };
  }
  return { label: '重新激活', target: 'active' as const };
}
