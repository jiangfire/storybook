import { renderHook, act } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { useStoryFormValidation, type StoryFormData } from '../useStoryFormValidation';

describe('useStoryFormValidation', () => {
  describe('表单验证', () => {
    it('应该验证必填字段', () => {
      const { result } = renderHook(() => useStoryFormValidation());

      const formData: Partial<StoryFormData> = {
        title: '',
        description: '',
        story_type: 'feature' as const,
        priority: 2,
      };

      let errors: ReturnType<typeof result.current.validate>;
      act(() => {
        errors = result.current.validate(formData);
      });

      expect(errors!.title).toBe('请输入故事标题');
      expect(Object.keys(errors!)).toHaveLength(1);
    });

    it('应该验证标题长度', () => {
      const { result } = renderHook(() => useStoryFormValidation());

      // 少于2个字符
      const shortTitle: Partial<StoryFormData> = {
        title: 'a',
        story_type: 'feature' as const,
        priority: 2,
      };

      let errors: ReturnType<typeof result.current.validate>;
      act(() => {
        errors = result.current.validate(shortTitle);
      });

      expect(errors!.title).toBe('标题长度应在2-200字符之间');
    });

    it('应该验证描述长度', () => {
      const { result } = renderHook(() => useStoryFormValidation());

      const longDesc: Partial<StoryFormData> = {
        title: '有效的标题',
        description: 'x'.repeat(2001),
        story_type: 'feature' as const,
        priority: 2,
      };

      let errors: ReturnType<typeof result.current.validate>;
      act(() => {
        errors = result.current.validate(longDesc);
      });

      expect(errors!.description).toBe('描述最多2000字符');
    });

    it('有效数据应该通过验证', () => {
      const { result } = renderHook(() => useStoryFormValidation());

      const validData: Partial<StoryFormData> = {
        title: '用户登录功能',
        description: '支持用户使用邮箱和密码登录系统',
        story_type: 'feature' as const,
        priority: 2,
        story_points: 3,
        acceptance_criteria: [],
        tags: [],
      };

      let errors: ReturnType<typeof result.current.validate>;
      act(() => {
        errors = result.current.validate(validData);
      });

      expect(Object.keys(errors!)).toHaveLength(0);
    });

    it('应该清除之前的错误', () => {
      const { result } = renderHook(() => useStoryFormValidation());

      // 先进行一次失败的验证
      let firstErrors: ReturnType<typeof result.current.validate>;
      act(() => {
        firstErrors = result.current.validate({ title: '', story_type: 'feature' as const, priority: 2 });
      });
      expect(firstErrors!.title).toBeDefined();

      // 再进行一次成功的验证
      let secondErrors: ReturnType<typeof result.current.validate>;
      act(() => {
        secondErrors = result.current.validate({
          title: '有效的标题',
          story_type: 'feature' as const,
          priority: 2,
        });
      });

      expect(secondErrors!.title).toBeUndefined();
    });

    it('应该允许空的描述字段', () => {
      const { result } = renderHook(() => useStoryFormValidation());

      const validWithoutDesc: Partial<StoryFormData> = {
        title: '用户登录功能',
        story_type: 'feature' as const,
        priority: 2,
      };

      let errors: ReturnType<typeof result.current.validate>;
      act(() => {
        errors = result.current.validate(validWithoutDesc);
      });

      expect(Object.keys(errors!)).toHaveLength(0);
    });
  });

  describe('错误信息', () => {
    it('应该提供清晰的错误消息', () => {
      const { result } = renderHook(() => useStoryFormValidation());

      let errors: ReturnType<typeof result.current.validate>;
      act(() => {
        errors = result.current.validate({ title: '', story_type: 'feature' as const, priority: 2 });
      });

      expect(errors!.title).toMatch(/请输入/);
    });

    it('应该支持同时有多个错误', () => {
      const { result } = renderHook(() => useStoryFormValidation());

      const invalidData: Partial<StoryFormData> = {
        title: 'a',
        description: 'x'.repeat(2001),
        story_type: 'feature' as const,
        priority: 2,
      };

      let errors: ReturnType<typeof result.current.validate>;
      act(() => {
        errors = result.current.validate(invalidData);
      });

      expect(errors!.title).toBeDefined();
      expect(errors!.description).toBeDefined();
    });
  });
});
