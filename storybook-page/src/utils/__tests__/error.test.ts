import { getErrorMessage } from '../error';

describe('getErrorMessage', () => {
  it('优先返回 axios 响应 message', () => {
    const axiosErrorLike = {
      isAxiosError: true,
      message: 'network error',
      response: {
        data: {
          message: '后端提示',
        },
      },
    };

    expect(getErrorMessage(axiosErrorLike)).toBe('后端提示');
  });

  it('axios 无响应体时回落到 error.message', () => {
    const axiosErrorLike = {
      isAxiosError: true,
      message: 'request failed',
    };

    expect(getErrorMessage(axiosErrorLike)).toBe('request failed');
  });

  it('普通 Error 与字符串', () => {
    expect(getErrorMessage(new Error('boom'))).toBe('boom');
    expect(getErrorMessage('直接错误')).toBe('直接错误');
  });

  it('未知错误回退 fallback', () => {
    expect(getErrorMessage(null, '默认错误')).toBe('默认错误');
  });
});
