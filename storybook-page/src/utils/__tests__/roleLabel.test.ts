import { getUserRoleLabel } from '../roleLabel';

describe('getUserRoleLabel', () => {
  it('返回已知角色标签', () => {
    expect(getUserRoleLabel('product')).toBe('产品经理');
    expect(getUserRoleLabel('developer')).toBe('开发人员');
    expect(getUserRoleLabel('tester')).toBe('测试人员');
    expect(getUserRoleLabel('tech_lead')).toBe('技术负责人');
    expect(getUserRoleLabel('admin')).toBe('管理员');
  });

  it('未知角色返回协作成员', () => {
    expect(getUserRoleLabel()).toBe('协作成员');
  });
});
