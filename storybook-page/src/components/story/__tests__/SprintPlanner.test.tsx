import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { SprintPlanner } from '../SprintPlanner';
import type { SprintSummary } from '../../../types/api';

describe('SprintPlanner', () => {
  const mockSprints: SprintSummary[] = [
    { id: 1, name: 'Sprint 1', start_date: '2024-01-01', end_date: '2024-01-14' },
    { id: 2, name: 'Sprint 2', start_date: '2024-01-15', end_date: '2024-01-28' },
    { id: 3, name: 'Sprint 3', start_date: '2024-01-29', end_date: '2024-02-11' },
  ];

  const defaultProps = {
    sprintId: '',
    sprints: mockSprints,
    isLoading: false,
    canPlan: true,
    isSubmitting: false,
    error: '',
    onSprintChange: vi.fn(),
    onUpdate: vi.fn(),
  };

  describe('基本渲染', () => {
    it('应该显示标签', () => {
      render(<SprintPlanner {...defaultProps} />);

      expect(screen.getByText('冲刺规划')).toBeInTheDocument();
    });

    it('应该显示下拉选择框', () => {
      render(<SprintPlanner {...defaultProps} />);

      const select = screen.getByRole('combobox');
      expect(select).toBeInTheDocument();
    });

    it('应该显示更新按钮', () => {
      render(<SprintPlanner {...defaultProps} />);

      expect(screen.getByRole('button', { name: '更新冲刺' })).toBeInTheDocument();
    });

    it('应该显示"不加入冲刺"选项', () => {
      render(<SprintPlanner {...defaultProps} />);

      expect(screen.getByText('不加入冲刺')).toBeInTheDocument();
    });
  });

  describe('冲刺列表', () => {
    it('应该显示所有冲刺选项', () => {
      render(<SprintPlanner {...defaultProps} sprintId="2" />);

      expect(screen.getByText('Sprint 1')).toBeInTheDocument();
      expect(screen.getByText('Sprint 2')).toBeInTheDocument();
      expect(screen.getByText('Sprint 3')).toBeInTheDocument();
    });

    it('应该正确显示当前选中的冲刺', () => {
      render(<SprintPlanner {...defaultProps} sprintId="2" />);

      const select = screen.getByRole('combobox') as HTMLSelectElement;
      expect(select.value).toBe('2');
    });

    it('应该能够选择不加入冲刺', () => {
      render(<SprintPlanner {...defaultProps} sprintId="1" />);

      const select = screen.getByRole('combobox');
      fireEvent.change(select, { target: { value: '' } });

      expect(defaultProps.onSprintChange).toHaveBeenCalledWith('');
    });

    it('应该能够选择冲刺', () => {
      render(<SprintPlanner {...defaultProps} sprintId="" />);

      const select = screen.getByRole('combobox');
      fireEvent.change(select, { target: { value: '2' } });

      expect(defaultProps.onSprintChange).toHaveBeenCalledWith('2');
    });
  });

  describe('更新操作', () => {
    it('点击更新按钮应该调用onUpdate', () => {
      render(<SprintPlanner {...defaultProps} sprintId="2" />);

      const updateButton = screen.getByRole('button', { name: '更新冲刺' });
      fireEvent.click(updateButton);

      expect(defaultProps.onUpdate).toHaveBeenCalledTimes(1);
    });

    it('提交时应该显示loading状态', () => {
      render(<SprintPlanner {...defaultProps} isSubmitting={true} />);

      const updateButton = screen.getByRole('button', { name: '处理中...' });
      expect(updateButton).toBeDisabled();
    });
  });

  describe('权限控制', () => {
    it('无权限时应该禁用选择框', () => {
      render(<SprintPlanner {...defaultProps} canPlan={false} />);

      const select = screen.getByRole('combobox');
      expect(select).toBeDisabled();
    });

    it('无权限时应该禁用更新按钮', () => {
      render(<SprintPlanner {...defaultProps} canPlan={false} />);

      const updateButton = screen.getByRole('button', { name: '更新冲刺' });
      expect(updateButton).toBeDisabled();
    });

    it('加载中应该禁用选择框', () => {
      render(<SprintPlanner {...defaultProps} isLoading={true} />);

      const select = screen.getByRole('combobox');
      expect(select).toBeDisabled();
    });
  });

  describe('状态提示', () => {
    it('加载时应该显示提示信息', () => {
      render(<SprintPlanner {...defaultProps} isLoading={true} />);

      expect(screen.getByText('冲刺列表加载中...')).toBeInTheDocument();
    });

    it('无权限时应该显示提示信息', () => {
      render(<SprintPlanner {...defaultProps} canPlan={false} />);

      expect(screen.getByText('仅产品经理可规划冲刺')).toBeInTheDocument();
    });

    it('有错误时应该显示错误信息', () => {
      render(<SprintPlanner {...defaultProps} error="更新失败" />);

      expect(screen.getByText('更新失败')).toBeInTheDocument();
    });

    it('没有错误时不应该显示错误信息', () => {
      render(<SprintPlanner {...defaultProps} error="" />);

      expect(screen.queryByText(/更新失败/)).not.toBeInTheDocument();
    });
  });

  describe('空冲刺列表', () => {
    it('冲刺列表为空时应该只显示"不加入冲刺"选项', () => {
      render(<SprintPlanner {...defaultProps} sprints={[]} />);

      expect(screen.getByText('不加入冲刺')).toBeInTheDocument();
      expect(screen.queryByText('Sprint 1')).not.toBeInTheDocument();
    });
  });

  describe('布局', () => {
    it('应该使用flex布局', () => {
      render(<SprintPlanner {...defaultProps} />);

      const container = screen.getByText('冲刺规划').closest('div');
      const flexContainer = container?.querySelector('.flex');
      expect(flexContainer).toBeInTheDocument();
    });

    it('应该支持自定义className', () => {
      render(<SprintPlanner {...defaultProps} className="custom-class" />);

      const outerContainer = screen.getByText('冲刺规划').closest('.custom-class');
      expect(outerContainer).toBeInTheDocument();
    });
  });
});
