import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { PrioritySelector } from '../PrioritySelector';

describe('PrioritySelector', () => {
  const defaultProps = {
    value: 2,
    onChange: vi.fn(),
  };

  describe('基本渲染', () => {
    it('应该显示所有优先级选项', () => {
      render(<PrioritySelector {...defaultProps} />);

      expect(screen.getByText('无')).toBeInTheDocument();
      expect(screen.getByText('低')).toBeInTheDocument();
      expect(screen.getByText('中')).toBeInTheDocument();
      expect(screen.getByText('高')).toBeInTheDocument();
      expect(screen.getByText('紧急')).toBeInTheDocument();
    });

    it('应该显示标签', () => {
      render(<PrioritySelector {...defaultProps} label="优先级" />);

      expect(screen.getByText('优先级')).toBeInTheDocument();
    });

    it('应该高亮当前选中的优先级', () => {
      render(<PrioritySelector {...defaultProps} value={3} />);

      const highButton = screen.getByText('高');
      expect(highButton).toHaveClass('bg-warning');
    });
  });

  describe('交互', () => {
    it('应该能够选择优先级', () => {
      const onChange = vi.fn();
      render(<PrioritySelector {...defaultProps} onChange={onChange} />);

      const urgentButton = screen.getByText('紧急');
      fireEvent.click(urgentButton);

      expect(onChange).toHaveBeenCalledWith(4);
    });

    it('应该能够选择"无"优先级', () => {
      const onChange = vi.fn();
      render(<PrioritySelector {...defaultProps} onChange={onChange} value={2} />);

      const noneButton = screen.getByText('无');
      fireEvent.click(noneButton);

      expect(onChange).toHaveBeenCalledWith(0);
    });

    it('应该能够切换到低优先级', () => {
      const onChange = vi.fn();
      render(<PrioritySelector {...defaultProps} onChange={onChange} />);

      const lowButton = screen.getByText('低');
      fireEvent.click(lowButton);

      expect(onChange).toHaveBeenCalledWith(1);
    });
  });

  describe('样式', () => {
    it('紧急优先级应该有危险样式', () => {
      render(<PrioritySelector {...defaultProps} value={4} />);

      const urgentButton = screen.getByText('紧急');
      expect(urgentButton).toHaveClass('bg-danger');
      expect(urgentButton).toHaveClass('text-white');
    });

    it('高优先级应该有警告样式', () => {
      render(<PrioritySelector {...defaultProps} value={3} />);

      const highButton = screen.getByText('高');
      expect(highButton).toHaveClass('bg-warning');
    });

    it('中优先级应该有默认样式', () => {
      render(<PrioritySelector {...defaultProps} value={2} />);

      const mediumButton = screen.getByText('中');
      expect(mediumButton).toHaveClass('bg-warning-light');
    });

    it('低优先级应该有信息样式', () => {
      render(<PrioritySelector {...defaultProps} value={1} />);

      const lowButton = screen.getByText('低');
      expect(lowButton).toHaveClass('bg-info-light');
    });

    it('未选中的按钮应该有hover效果', () => {
      render(<PrioritySelector {...defaultProps} value={2} />);

      const noneButton = screen.getByText('无');
      expect(noneButton).toHaveClass('hover:bg-secondary-200');
    });
  });

  describe('边界条件', () => {
    it('应该处理无效的优先级值', () => {
      render(<PrioritySelector {...defaultProps} value={999} />);

      // 不应该崩溃，所有按钮都应该渲染
      expect(screen.getByText('无')).toBeInTheDocument();
      expect(screen.getByText('紧急')).toBeInTheDocument();
    });

    it('应该支持自定义className', () => {
      render(
        <PrioritySelector {...defaultProps} className="custom-class" />
      );

      const container = screen.getByText('无').closest('div');
      expect(container?.parentElement).toHaveClass('custom-class');
    });
  });
});
