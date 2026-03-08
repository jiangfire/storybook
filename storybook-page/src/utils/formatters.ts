import { format, formatDistanceToNow, isValid } from 'date-fns';
import { zhCN } from 'date-fns/locale';

/**
 * 格式化日期
 */
export function formatDate(date: string | Date, formatStr: string = 'yyyy-MM-dd HH:mm'): string {
  try {
    const d = typeof date === 'string' ? new Date(date) : date;
    if (!isValid(d)) return '-';
    return format(d, formatStr, { locale: zhCN });
  } catch {
    return '-';
  }
}

/**
 * 格式化为相对时间（如"3小时前"）
 */
export function formatRelativeTime(date: string | Date): string {
  try {
    const d = typeof date === 'string' ? new Date(date) : date;
    if (!isValid(d)) return '-';
    return formatDistanceToNow(d, { addSuffix: true, locale: zhCN });
  } catch {
    return '-';
  }
}

/**
 * 格式化故事点
 */
export function formatStoryPoints(points?: number): string {
  if (!points) return '-';
  return points.toString();
}

/**
 * 格式化优先级
 */
export function formatPriority(priority: number): string {
  const labels: Record<number, string> = {
    0: '无',
    1: '低',
    2: '中',
    3: '高',
    4: '紧急',
  };
  return labels[priority] || '-';
}

/**
 * 获取优先级颜色
 */
export function getPriorityColor(priority: number): string {
  const colors: Record<number, string> = {
    0: 'bg-gray-200 text-gray-700',
    1: 'bg-blue-100 text-blue-700',
    2: 'bg-yellow-100 text-yellow-700',
    3: 'bg-orange-100 text-orange-700',
    4: 'bg-red-100 text-red-700',
  };
  return colors[priority] || colors[0];
}

/**
 * 格式化故事类型
 */
export function formatStoryType(type: string): string {
  const labels: Record<string, string> = {
    feature: '功能',
    bug: 'Bug',
    chore: '杂项',
  };
  return labels[type] || type;
}

/**
 * 获取故事类型颜色
 */
export function getStoryTypeColor(type: string): string {
  const colors: Record<string, string> = {
    feature: 'bg-blue-100 text-blue-700',
    bug: 'bg-red-100 text-red-700',
    chore: 'bg-gray-100 text-gray-700',
  };
  return colors[type] || 'bg-gray-100 text-gray-700';
}

/**
 * 格式化故事状态
 */
export function formatStoryStatus(status: string): string {
  const labels: Record<string, string> = {
    backlog: '待办',
    ready: '就绪',
    in_progress: '开发中',
    test: '测试中',
    done: '已完成',
  };
  return labels[status] || status;
}

/**
 * 格式化冲刺状态
 */
export function formatSprintStatus(status: string): string {
  const labels: Record<string, string> = {
    planned: '未开始',
    active: '进行中',
    completed: '已完成',
  };
  return labels[status] || status;
}

/**
 * 格式化用户角色
 */
export function formatUserRole(role: string): string {
  const labels: Record<string, string> = {
    product: '产品经理',
    developer: '开发人员',
    tester: '测试人员',
  };
  return labels[role] || role;
}

/**
 * 格式化数字（如1000 -> 1k）
 */
export function formatNumber(num: number): string {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(1) + 'M';
  }
  if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'K';
  }
  return num.toString();
}

/**
 * 截断文本
 */
export function truncateText(text: string, maxLength: number = 50): string {
  if (text.length <= maxLength) return text;
  return text.slice(0, maxLength) + '...';
}

/**
 * 获取用户名称（从邮箱中提取）
 */
export function getUserName(email: string): string {
  const parts = email.split('@');
  const name = parts[0];
  // 首字母大写
  return name.charAt(0).toUpperCase() + name.slice(1);
}

/**
 * 获取用户头像首字母
 */
export function getUserInitials(email: string): string {
  const name = getUserName(email);
  return name.charAt(0).toUpperCase();
}

/**
 * 计算完成百分比
 */
export function calculatePercentage(completed: number, total: number): number {
  if (total === 0) return 0;
  return Math.round((completed / total) * 100);
}
