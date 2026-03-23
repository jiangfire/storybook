import type { UserRole } from '../types/models';

export function getUserRoleLabel(role?: UserRole) {
  switch (role) {
    case 'product':
      return '产品经理';
    case 'developer':
      return '开发人员';
    case 'tester':
      return '测试人员';
    case 'tech_lead':
      return '技术负责人';
    case 'admin':
      return '管理员';
    default:
      return '协作成员';
  }
}
