import { useState, useCallback } from 'react';
import { isValidStoryTitle, isValidStoryDescription } from '../utils/validators';

export interface ValidationError {
  title?: string;
  description?: string;
}

export interface StoryFormData {
  title: string;
  description?: string;
  story_type: 'feature' | 'bug' | 'chore';
  priority: number;
  story_points?: number;
  acceptance_criteria?: Array<{ description: string; order: number }>;
  tags?: string[];
}

export interface UseStoryFormValidationReturn {
  errors: ValidationError;
  validate: (data: Partial<StoryFormData>) => ValidationError;
  clearErrors: () => void;
}

export const useStoryFormValidation = (): UseStoryFormValidationReturn => {
  const [errors, setErrors] = useState<ValidationError>({});

  const validate = useCallback((data: Partial<StoryFormData>): ValidationError => {
    const newErrors: ValidationError = {};

    // 验证标题
    if (!data.title) {
      newErrors.title = '请输入故事标题';
    } else if (!isValidStoryTitle(data.title)) {
      newErrors.title = '标题长度应在2-200字符之间';
    }

    // 验证描述（可选字段）
    if (data.description && !isValidStoryDescription(data.description)) {
      newErrors.description = '描述最多2000字符';
    }

    setErrors(newErrors);
    return newErrors;
  }, []);

  const clearErrors = useCallback(() => {
    const emptyErrors: ValidationError = {};
    setErrors(emptyErrors);
  }, []);

  return {
    errors,
    validate,
    clearErrors,
  };
};
