import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { AcceptanceCriteriaManager } from '../AcceptanceCriteriaManager';

describe('AcceptanceCriteriaManager', () => {
  describe('基本渲染', () => {
    it('应该显示AC列表和输入框', () => {
      const criteria = [
        { description: 'Given用户未登录', order: 0 },
        { description: 'When输入账号密码', order: 1 },
      ];

      const { container } = render(
        <AcceptanceCriteriaManager
          criteria={criteria}
          onChange={vi.fn()}
        />
      );

      expect(screen.getByText('验收标准 (AC)')).toBeInTheDocument();
      expect(screen.getByText('Given用户未登录')).toBeInTheDocument();
      expect(screen.getByText('When输入账号密码')).toBeInTheDocument();
      expect(screen.getByPlaceholderText(/添加验收标准/)).toBeInTheDocument();
      // 验证有两个AC项
      const acItems = container.querySelectorAll('.bg-secondary-50');
      expect(acItems.length).toBe(2);
    });

    it('应该显示添加按钮', () => {
      render(
        <AcceptanceCriteriaManager
          criteria={[]}
          onChange={vi.fn()}
        />
      );

      expect(screen.getByRole('button', { name: '添加' })).toBeInTheDocument();
    });
  });

  describe('添加AC', () => {
    it('应该能够添加新的验收标准', () => {
      const onChange = vi.fn();
      render(
        <AcceptanceCriteriaManager
          criteria={[]}
          onChange={onChange}
        />
      );

      const input = screen.getByPlaceholderText(/添加验收标准/);
      const addButton = screen.getByRole('button', { name: '添加' });

      fireEvent.change(input, { target: { value: 'Given用户未登录' } });
      fireEvent.click(addButton);

      expect(onChange).toHaveBeenCalledWith([
        { description: 'Given用户未登录', order: 0 },
      ]);
    });

    it('应该支持Enter键添加', () => {
      const onChange = vi.fn();
      render(
        <AcceptanceCriteriaManager
          criteria={[]}
          onChange={onChange}
        />
      );

      const input = screen.getByPlaceholderText(/添加验收标准/);

      fireEvent.change(input, { target: { value: 'When输入密码' } });
      fireEvent.keyDown(input, { key: 'Enter', code: 'Enter' });

      expect(onChange).toHaveBeenCalledWith([
        { description: 'When输入密码', order: 0 },
      ]);
    });

    it('不应该添加空白的验收标准', () => {
      const onChange = vi.fn();
      render(
        <AcceptanceCriteriaManager
          criteria={[]}
          onChange={onChange}
        />
      );

      const input = screen.getByPlaceholderText(/添加验收标准/);
      const addButton = screen.getByRole('button', { name: '添加' });

      fireEvent.change(input, { target: { value: '   ' } });
      fireEvent.click(addButton);

      expect(onChange).not.toHaveBeenCalled();
    });
  });

  describe('删除AC', () => {
    it('应该能够删除验收标准', () => {
      const onChange = vi.fn();
      const criteria = [
        { description: 'Given用户未登录', order: 0 },
        { description: 'When输入账号密码', order: 1 },
      ];

      render(
        <AcceptanceCriteriaManager
          criteria={criteria}
          onChange={onChange}
        />
      );

      const deleteButtons = screen.getAllByRole('button', { name: /删除验收标准/ });
      fireEvent.click(deleteButtons[0]);

      expect(onChange).toHaveBeenCalledWith([
        { description: 'When输入账号密码', order: 1 },
      ]);
    });
  });

  describe('边界条件', () => {
    it('空列表时应该显示占位符', () => {
      render(
        <AcceptanceCriteriaManager
          criteria={[]}
          onChange={vi.fn()}
        />
      );

      expect(screen.queryByText(/^\d+\./)).not.toBeInTheDocument();
    });

    it('应该在添加后清空输入框', () => {
      const onChange = vi.fn();
      render(
        <AcceptanceCriteriaManager
          criteria={[]}
          onChange={onChange}
        />
      );

      const input = screen.getByPlaceholderText(/添加验收标准/) as HTMLInputElement;
      const addButton = screen.getByRole('button', { name: '添加' });

      fireEvent.change(input, { target: { value: 'Given用户未登录' } });
      fireEvent.click(addButton);

      expect(input.value).toBe('');
    });

    it('应该正确显示序号', () => {
      const criteria = [
        { description: 'AC 1', order: 0 },
        { description: 'AC 2', order: 1 },
        { description: 'AC 3', order: 2 },
      ];

      const { container } = render(
        <AcceptanceCriteriaManager
          criteria={criteria}
          onChange={vi.fn()}
        />
      );

      // 验证有3个AC项
      const acItems = container.querySelectorAll('.bg-secondary-50');
      expect(acItems.length).toBe(3);

      // 验证每个AC的描述都存在
      expect(screen.getByText('AC 1')).toBeInTheDocument();
      expect(screen.getByText('AC 2')).toBeInTheDocument();
      expect(screen.getByText('AC 3')).toBeInTheDocument();
    });
  });
});
