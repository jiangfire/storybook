import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { StoryPointsSelector } from '../StoryPointsSelector';

describe('StoryPointsSelector', () => {
  const defaultProps = {
    value: undefined,
    onChange: vi.fn(),
  };

  describe('基本渲染', () => {
    it('应该显示所有故事点选项', () => {
      render(<StoryPointsSelector {...defaultProps} />);

      expect(screen.getByText('1')).toBeInTheDocument();
      expect(screen.getByText('2')).toBeInTheDocument();
      expect(screen.getByText('3')).toBeInTheDocument();
      expect(screen.getByText('5')).toBeInTheDocument();
      expect(screen.getByText('8')).toBeInTheDocument();
      expect(screen.getByText('13')).toBeInTheDocument();
    });

    it('应该显示标签', () => {
      render(<StoryPointsSelector {...defaultProps} label="故事点" />);

      expect(screen.getByText('故事点')).toBeInTheDocument();
    });

    it('应该高亮当前选中的故事点', () => {
      render(<StoryPointsSelector {...defaultProps} value={3} />);

      const threeButton = screen.getByText('3');
      expect(threeButton).toHaveClass('bg-primary');
    });
  });

  describe('交互', () => {
    it('应该能够选择故事点', () => {
      const onChange = vi.fn();
      render(<StoryPointsSelector {...defaultProps} onChange={onChange} />);

      const fiveButton = screen.getByText('5');
      fireEvent.click(fiveButton);

      expect(onChange).toHaveBeenCalledWith(5);
    });

    it('应该能够取消选择故事点', () => {
      const onChange = vi.fn();
      render(<StoryPointsSelector {...defaultProps} value={3} onChange={onChange} />);

      const threeButton = screen.getByText('3');
      fireEvent.click(threeButton);

      expect(onChange).toHaveBeenCalledWith(undefined);
    });

    it('应该支持切换不同的故事点', () => {
      const onChange = vi.fn();
      render(<StoryPointsSelector {...defaultProps} value={2} onChange={onChange} />);

      const eightButton = screen.getByText('8');
      fireEvent.click(eightButton);

      expect(onChange).toHaveBeenCalledWith(8);
    });
  });

  describe('样式', () => {
    it('选中的按钮应该有高亮样式', () => {
      render(<StoryPointsSelector {...defaultProps} value={5} />);

      const fiveButton = screen.getByText('5');
      expect(fiveButton).toHaveClass('border-primary');
      expect(fiveButton).toHaveClass('bg-primary');
      expect(fiveButton).toHaveClass('text-white');
    });

    it('未选中的按钮应该有默认样式', () => {
      render(<StoryPointsSelector {...defaultProps} value={5} />);

      const threeButton = screen.getByText('3');
      expect(threeButton).toHaveClass('border-border');
      expect(threeButton).toHaveClass('bg-white');
      expect(threeButton).toHaveClass('text-text');
    });

    it('未选中的按钮应该有hover效果', () => {
      render(<StoryPointsSelector {...defaultProps} value={5} />);

      const threeButton = screen.getByText('3');
      expect(threeButton).toHaveClass('hover:border-primary-300');
    });

    it('所有按钮应该有圆角和边框', () => {
      render(<StoryPointsSelector {...defaultProps} />);

      const buttons = screen.getAllByRole('button');
      buttons.forEach((button) => {
        expect(button).toHaveClass('rounded-lg');
        expect(button).toHaveClass('border');
      });
    });
  });

  describe('边界条件', () => {
    it('应该处理undefined值', () => {
      render(<StoryPointsSelector {...defaultProps} value={undefined} />);

      // 不应该崩溃
      expect(screen.getByText('1')).toBeInTheDocument();
    });

    it('应该支持自定义className', () => {
      render(
        <StoryPointsSelector {...defaultProps} className="custom-class" />
      );

      const container = screen.getByText('1').closest('div');
      expect(container?.parentElement).toHaveClass('custom-class');
    });
  });
});
