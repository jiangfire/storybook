import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { StoryTypeSelector } from '../StoryTypeSelector';
import type { StoryType } from '../../../types/models';

describe('StoryTypeSelector', () => {
  const defaultProps = {
    value: 'feature' as StoryType,
    onChange: vi.fn(),
  };

  describe('基本渲染', () => {
    it('应该显示所有故事类型选项', () => {
      render(<StoryTypeSelector {...defaultProps} />);

      expect(screen.getByText('功能')).toBeInTheDocument();
      expect(screen.getByText('Bug')).toBeInTheDocument();
      expect(screen.getByText('杂项')).toBeInTheDocument();
    });

    it('应该显示标签', () => {
      render(<StoryTypeSelector {...defaultProps} label="故事类型" />);

      expect(screen.getByText('故事类型')).toBeInTheDocument();
    });

    it('应该高亮当前选中的类型', () => {
      render(<StoryTypeSelector {...defaultProps} value="bug" as StoryType />);

      const bugOption = screen.getByText('Bug').closest('button');
      expect(bugOption).toHaveClass('border-red-500');
    });
  });

  describe('交互', () => {
    it('应该能够选择功能类型', () => {
      const onChange = vi.fn();
      render(<StoryTypeSelector {...defaultProps} onChange={onChange} value="bug" as StoryType />);

      const featureButton = screen.getByText('功能');
      fireEvent.click(featureButton);

      expect(onChange).toHaveBeenCalledWith('feature');
    });

    it('应该能够选择Bug类型', () => {
      const onChange = vi.fn();
      render(<StoryTypeSelector {...defaultProps} onChange={onChange} />);

      const bugButton = screen.getByText('Bug');
      fireEvent.click(bugButton);

      expect(onChange).toHaveBeenCalledWith('bug');
    });

    it('应该能够选择杂项类型', () => {
      const onChange = vi.fn();
      render(<StoryTypeSelector {...defaultProps} onChange={onChange} />);

      const choreButton = screen.getByText('杂项');
      fireEvent.click(choreButton);

      expect(onChange).toHaveBeenCalledWith('chore');
    });
  });

  describe('样式', () => {
    it('功能类型应该有蓝色样式', () => {
      render(<StoryTypeSelector {...defaultProps} value="feature" as StoryType />);

      const featureOption = screen.getByText('功能').closest('button');
      expect(featureOption).toHaveClass('border-blue-500');
      expect(featureOption).toHaveClass('bg-blue-50');
      expect(featureOption).toHaveClass('text-blue-700');
    });

    it('Bug类型应该有红色样式', () => {
      render(<StoryTypeSelector {...defaultProps} value="bug" as StoryType />);

      const bugOption = screen.getByText('Bug').closest('button');
      expect(bugOption).toHaveClass('border-red-500');
      expect(bugOption).toHaveClass('bg-red-50');
      expect(bugOption).toHaveClass('text-red-700');
    });

    it('杂项类型应该有默认样式', () => {
      render(<StoryTypeSelector {...defaultProps} value="chore" as StoryType />);

      const choreOption = screen.getByText('杂项').closest('button');
      expect(choreOption).toHaveClass('border-primary-500');
      expect(choreOption).toHaveClass('bg-secondary-50');
    });

    it('未选中的类型应该有hover效果', () => {
      render(<StoryTypeSelector {...defaultProps} value="feature" as StoryType />);

      const bugOption = screen.getByText('Bug').closest('button');
      expect(bugOption).toHaveClass('hover:border-primary-300');
    });

    it('所有选项应该有图标', () => {
      render(<StoryTypeSelector {...defaultProps} />);

      // 检查每个选项都有一个图标容器
      const icons = document.querySelectorAll('.inline-flex.h-9.w-9');
      expect(icons).toHaveLength(3);
    });
  });

  describe('布局', () => {
    it('应该在移动端显示单列', () => {
      render(<StoryTypeSelector {...defaultProps} />);

      const gridContainer = screen.getByText('功能').closest('button')?.parentElement;
      expect(gridContainer).toHaveClass('grid');
      expect(gridContainer).toHaveClass('grid-cols-1');
    });

    it('应该在桌面端显示三列', () => {
      render(<StoryTypeSelector {...defaultProps} />);

      const gridContainer = screen.getByText('功能').closest('button')?.parentElement;
      expect(gridContainer).toHaveClass('sm:grid-cols-3');
    });

    it('应该支持自定义className', () => {
      render(
        <StoryTypeSelector {...defaultProps} className="custom-class" />
      );

      const outerContainer = screen.getByText('功能').closest('button')?.parentElement?.parentElement;
      expect(outerContainer).toHaveClass('custom-class');
    });
  });
});
