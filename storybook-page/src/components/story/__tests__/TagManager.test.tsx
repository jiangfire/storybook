import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { TagManager } from '../TagManager';

describe('TagManager', () => {
  describe('基本渲染', () => {
    it('应该显示标签列表和输入框', () => {
      const tags = ['auth', 'api'];

      render(
        <TagManager
          tags={tags}
          onChange={vi.fn()}
        />
      );

      expect(screen.getByText('标签')).toBeInTheDocument();
      expect(screen.getByText('auth')).toBeInTheDocument();
      expect(screen.getByText('api')).toBeInTheDocument();
      expect(screen.getByPlaceholderText(/添加标签/)).toBeInTheDocument();
    });

    it('应该显示添加按钮', () => {
      render(
        <TagManager
          tags={[]}
          onChange={vi.fn()}
        />
      );

      expect(screen.getByRole('button', { name: '添加' })).toBeInTheDocument();
    });
  });

  describe('添加标签', () => {
    it('应该能够添加新标签', () => {
      const onChange = vi.fn();
      render(
        <TagManager
          tags={[]}
          onChange={onChange}
        />
      );

      const input = screen.getByPlaceholderText(/添加标签/);
      const addButton = screen.getByRole('button', { name: '添加' });

      fireEvent.change(input, { target: { value: 'auth' } });
      fireEvent.click(addButton);

      expect(onChange).toHaveBeenCalledWith(['auth']);
    });

    it('应该支持Enter键添加', () => {
      const onChange = vi.fn();
      render(
        <TagManager
          tags={[]}
          onChange={onChange}
        />
      );

      const input = screen.getByPlaceholderText(/添加标签/);

      fireEvent.change(input, { target: { value: 'api' } });
      fireEvent.keyDown(input, { key: 'Enter', code: 'Enter' });

      expect(onChange).toHaveBeenCalledWith(['api']);
    });

    it('不应该添加空标签', () => {
      const onChange = vi.fn();
      render(
        <TagManager
          tags={[]}
          onChange={onChange}
        />
      );

      const input = screen.getByPlaceholderText(/添加标签/);
      const addButton = screen.getByRole('button', { name: '添加' });

      fireEvent.change(input, { target: { value: '   ' } });
      fireEvent.click(addButton);

      expect(onChange).not.toHaveBeenCalled();
    });

    it('不应该添加重复标签', () => {
      const onChange = vi.fn();
      render(
        <TagManager
          tags={['auth']}
          onChange={onChange}
        />
      );

      const input = screen.getByPlaceholderText(/添加标签/);
      const addButton = screen.getByRole('button', { name: '添加' });

      fireEvent.change(input, { target: { value: 'auth' } });
      fireEvent.click(addButton);

      expect(onChange).not.toHaveBeenCalled();
    });

    it('应该在添加后清空输入框', () => {
      const onChange = vi.fn();
      render(
        <TagManager
          tags={[]}
          onChange={onChange}
        />
      );

      const input = screen.getByPlaceholderText(/添加标签/) as HTMLInputElement;
      const addButton = screen.getByRole('button', { name: '添加' });

      fireEvent.change(input, { target: { value: 'auth' } });
      fireEvent.click(addButton);

      expect(input.value).toBe('');
    });
  });

  describe('删除标签', () => {
    it('应该能够删除标签', () => {
      const onChange = vi.fn();
      render(
        <TagManager
          tags={['auth', 'api']}
          onChange={onChange}
        />
      );

      const deleteButtons = screen.getAllByRole('button', { name: /删除标签/ });
      fireEvent.click(deleteButtons[0]);

      expect(onChange).toHaveBeenCalledWith(['api']);
    });

    it('删除按钮应该有正确的aria-label', () => {
      render(
        <TagManager
          tags={['auth']}
          onChange={vi.fn()}
        />
      );

      const deleteButton = screen.getByRole('button', { name: /删除标签auth/ });
      expect(deleteButton).toHaveAttribute('aria-label', '删除标签auth');
    });
  });

  describe('边界条件', () => {
    it('空列表时应该不显示标签', () => {
      render(
        <TagManager
          tags={[]}
          onChange={vi.fn()}
        />
      );

      expect(screen.queryByText(/auth/)).not.toBeInTheDocument();
    });

    it('应该正确显示多个标签', () => {
      const onChange = vi.fn();
      const tags = ['auth', 'api', 'frontend'];

      render(
        <TagManager
          tags={tags}
          onChange={onChange}
        />
      );

      expect(screen.getByText('auth')).toBeInTheDocument();
      expect(screen.getByText('api')).toBeInTheDocument();
      expect(screen.getByText('frontend')).toBeInTheDocument();
    });

    it('应该支持标签的特殊字符', () => {
      const onChange = vi.fn();
      render(
        <TagManager
          tags={[]}
          onChange={onChange}
        />
      );

      const input = screen.getByPlaceholderText(/添加标签/);
      const addButton = screen.getByRole('button', { name: '添加' });

      fireEvent.change(input, { target: { value: 'api-v2' } });
      fireEvent.click(addButton);

      expect(onChange).toHaveBeenCalledWith(['api-v2']);
    });
  });
});
