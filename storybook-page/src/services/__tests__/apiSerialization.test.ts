import apiClient from '../api';

// 序列化行为必须用真实 axios 实例验证（getUri 会应用实例的 paramsSerializer）
describe('apiClient 查询参数序列化', () => {
  it('数组参数序列化为重复键（status=a&status=b），与后端 c.QueryArray 对齐', () => {
    const uri = apiClient.getUri({
      url: '/api/search',
      params: { q: '登录', status: ['pending', 'done'] },
    });

    expect(uri).toContain('q=');
    expect(uri).toContain('status=pending&status=done');
    expect(uri).not.toContain('status%5B%5D');
    expect(uri).not.toContain('status[]');
  });
});
