const axiosMock = vi.hoisted(() => {
  const client = vi.fn();
  const requestUse = vi.fn();
  const responseUse = vi.fn();
  client.interceptors = {
    request: { use: requestUse },
    response: { use: responseUse },
  };

  const axiosDefault = {
    create: vi.fn(() => client),
    post: vi.fn(),
    isAxiosError: vi.fn((error: unknown) => Boolean((error as { isAxiosError?: boolean })?.isAxiosError)),
  };

  return { client, requestUse, responseUse, axiosDefault };
});

vi.mock('axios', () => ({
  default: axiosMock.axiosDefault,
  create: axiosMock.axiosDefault.create,
  post: axiosMock.axiosDefault.post,
  isAxiosError: axiosMock.axiosDefault.isAxiosError,
}));

describe('api client interceptors', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    localStorage.clear();
  });

  it('请求拦截器会注入 Bearer token', async () => {
    let requestHandler: ((config: { headers?: Record<string, string> }) => unknown) | undefined;
    axiosMock.requestUse.mockImplementation((onFulfilled) => {
      requestHandler = onFulfilled;
      return 1;
    });

    await import('../api');

    localStorage.setItem('token', 'access-token');
    const result = requestHandler?.({ headers: {} }) as { headers: Record<string, string> };

    expect(result.headers.Authorization).toBe('Bearer access-token');
  });

  it('401 且存在 refresh_token 时会刷新 token 并重试原请求', async () => {
    let responseRejected:
      | ((error: {
          isAxiosError?: boolean;
          response?: { status: number };
          config: { headers?: Record<string, string>; _retry?: boolean };
        }) => Promise<unknown>)
      | undefined;

    axiosMock.responseUse.mockImplementation((_, onRejected) => {
      responseRejected = onRejected;
      return 1;
    });

    await import('../api');

    localStorage.setItem('refresh_token', 'refresh-token');
    axiosMock.axiosDefault.post.mockResolvedValue({
      data: { data: { token: 'new-token' } },
    });
    axiosMock.client.mockResolvedValue('retried');

    const originalRequest = { headers: {} as Record<string, string> };
    const result = await responseRejected?.({
      isAxiosError: true,
      response: { status: 401 },
      config: originalRequest,
    });

    expect(axiosMock.axiosDefault.post).toHaveBeenCalledWith(
      expect.stringContaining('/api/auth/refresh'),
      { refresh_token: 'refresh-token' }
    );
    expect(localStorage.getItem('token')).toBe('new-token');
    expect(originalRequest.headers.Authorization).toBe('Bearer new-token');
    expect(axiosMock.client).toHaveBeenCalledWith(originalRequest);
    expect(result).toBe('retried');
  });

  it('普通错误会保留服务端 message 并 reject', async () => {
    let responseRejected:
      | ((error: {
          isAxiosError?: boolean;
          message?: string;
          response?: { status: number; data?: { message?: string } };
          config: { headers?: Record<string, string> };
        }) => Promise<unknown>)
      | undefined;

    axiosMock.responseUse.mockImplementation((_, onRejected) => {
      responseRejected = onRejected;
      return 1;
    });

    await import('../api');

    await expect(
      responseRejected?.({
        isAxiosError: true,
        message: 'fallback',
        response: { status: 403, data: { message: '权限不足' } },
        config: { headers: {} },
      })
    ).rejects.toMatchObject({ message: '权限不足' });
  });
});
