import {
  calculatePercentage,
  formatDate,
  formatNumber,
  formatPriority,
  formatRelativeTime,
  formatSprintStatus,
  formatStoryPoints,
  formatStoryStatus,
  formatStoryType,
  formatUserRole,
  getPriorityColor,
  getStoryTypeColor,
  getUserInitials,
  getUserName,
  truncateText,
} from '../formatters';

describe('formatters', () => {
  it('格式化日期与相对时间', () => {
    expect(formatDate('2026-03-08T12:00:00+08:00', 'yyyy-MM-dd')).toBe('2026-03-08');
    expect(formatDate('invalid-date')).toBe('-');
    expect(formatRelativeTime('invalid-date')).toBe('-');
    expect(formatRelativeTime(new Date())).not.toBe('-');
  });

  it('格式化故事点、优先级、颜色', () => {
    expect(formatStoryPoints()).toBe('-');
    expect(formatStoryPoints(5)).toBe('5');
    expect(formatPriority(4)).toBe('紧急');
    expect(formatPriority(999)).toBe('-');
    expect(getPriorityColor(3)).toContain('warning');
    expect(getPriorityColor(999)).toContain('secondary');
  });

  it('格式化类型与状态', () => {
    expect(formatStoryType('feature')).toBe('功能');
    expect(formatStoryType('custom')).toBe('custom');
    expect(getStoryTypeColor('bug')).toContain('danger');
    expect(getStoryTypeColor('unknown')).toContain('secondary');
    expect(formatStoryStatus('ready')).toBe('就绪');
    expect(formatStoryStatus('custom')).toBe('custom');
    expect(formatSprintStatus('planned')).toBe('未开始');
    expect(formatSprintStatus('custom')).toBe('custom');
    expect(formatUserRole('product')).toBe('产品经理');
    expect(formatUserRole('tech_lead')).toBe('tech_lead');
  });

  it('格式化数字与文本', () => {
    expect(formatNumber(999)).toBe('999');
    expect(formatNumber(1200)).toBe('1.2K');
    expect(formatNumber(2000000)).toBe('2.0M');
    expect(truncateText('hello', 10)).toBe('hello');
    expect(truncateText('abcdefgh', 5)).toBe('abcde...');
  });

  it('用户名称与百分比', () => {
    expect(getUserName('demo_user@example.com')).toBe('Demo_user');
    expect(getUserInitials('demo_user@example.com')).toBe('D');
    expect(calculatePercentage(3, 4)).toBe(75);
    expect(calculatePercentage(1, 0)).toBe(0);
  });
});
