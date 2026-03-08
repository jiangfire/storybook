import axios from 'axios';
import { ApiResponse } from '../types/api';

export function getErrorMessage(error: unknown, fallback = '操作失败，请稍后重试'): string {
  if (axios.isAxiosError<ApiResponse>(error)) {
    return error.response?.data?.message || error.message || fallback;
  }

  if (error instanceof Error) {
    return error.message || fallback;
  }

  if (typeof error === 'string' && error.trim()) {
    return error;
  }

  return fallback;
}
