import { useState, useEffect, useCallback } from 'react';
import { userManagementService } from '../../services/userManagementService';
import type { User, UserRole } from '../../types/models';
import { getErrorMessage } from '../../utils/error';
import LoadingSpinner from '../../components/ui/LoadingSpinner';
import Button from '../../components/ui/Button';
import Modal from '../../components/ui/Modal';
import Input from '../../components/ui/Input';

const roleLabels: Record<UserRole, string> = {
  product: '产品经理',
  developer: '开发人员',
  tester: '测试人员',
  tech_lead: '技术负责人',
};

const roleColors: Record<UserRole, string> = {
  product: 'bg-blue-100 text-blue-800',
  developer: 'bg-green-100 text-green-800',
  tester: 'bg-purple-100 text-purple-800',
  tech_lead: 'bg-orange-100 text-orange-800',
};

export default function UserManagementPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedRole, setSelectedRole] = useState<UserRole | ''>('');
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [formData, setFormData] = useState({
    email: '',
    username: '',
    password: '',
    role: 'developer' as UserRole,
  });
  const [processing, setProcessing] = useState(false);

  const loadUsers = useCallback(async () => {
    try {
      setLoading(true);
      const res = await userManagementService.getUsers({
        role: selectedRole || undefined,
        search: searchQuery || undefined,
      });
      setUsers(res.data.users);
    } catch (error) {
      console.error('Failed to load users:', error);
    } finally {
      setLoading(false);
    }
  }, [searchQuery, selectedRole]);

  useEffect(() => {
    void loadUsers();
  }, [loadUsers]);

  const handleSearch = () => {
    void loadUsers();
  };

  const openCreateModal = () => {
    setFormData({
      email: '',
      username: '',
      password: '',
      role: 'developer',
    });
    setIsCreateModalOpen(true);
  };

  const openEditModal = (user: User) => {
    setSelectedUser(user);
    setFormData({
      email: user.email,
      username: user.email.split('@')[0],
      password: '',
      role: user.role,
    });
    setIsEditModalOpen(true);
  };

  const handleCreate = async () => {
    try {
      setProcessing(true);
      await userManagementService.createUser({
        email: formData.email,
        username: formData.username,
        password: formData.password,
        role: formData.role,
      });
      setIsCreateModalOpen(false);
      await loadUsers();
    } catch (error: unknown) {
      console.error('Failed to create user:', error);
      alert(getErrorMessage(error, '创建失败，请重试'));
    } finally {
      setProcessing(false);
    }
  };

  const handleUpdate = async () => {
    if (!selectedUser) return;

    try {
      setProcessing(true);
      const updateData: { username?: string; role?: UserRole; password?: string } = {};
      if (formData.username) updateData.username = formData.username;
      if (formData.role !== selectedUser.role) updateData.role = formData.role;
      if (formData.password) updateData.password = formData.password;

      await userManagementService.updateUser(selectedUser.id, updateData);
      setIsEditModalOpen(false);
      await loadUsers();
    } catch (error: unknown) {
      console.error('Failed to update user:', error);
      alert(getErrorMessage(error, '更新失败，请重试'));
    } finally {
      setProcessing(false);
    }
  };

  const handleDelete = async (user: User) => {
    if (!confirm(`确定要删除用户 ${user.email} 吗？此操作不可恢复。`)) {
      return;
    }

    try {
      await userManagementService.deleteUser(user.id);
      await loadUsers();
    } catch (error: unknown) {
      console.error('Failed to delete user:', error);
      alert(getErrorMessage(error, '删除失败，请重试'));
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <LoadingSpinner />
      </div>
    );
  }

  return (
    <div className="container mx-auto px-4 py-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text mb-2">人员管理</h1>
          <p className="text-text-light">管理团队成员和权限</p>
        </div>
        <Button onClick={openCreateModal}>+ 新建用户</Button>
      </div>

      {/* 筛选栏 */}
      <div className="bg-white rounded-lg shadow-sm border border-border p-4 mb-6">
        <div className="flex flex-wrap gap-4">
          <div className="w-48">
            <label className="block text-sm font-medium text-text mb-1">角色</label>
            <select
              value={selectedRole}
              onChange={(e) => setSelectedRole(e.target.value as UserRole)}
              className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <option value="">全部角色</option>
              {(Object.keys(roleLabels) as UserRole[]).map((role) => (
                <option key={role} value={role}>
                  {roleLabels[role]}
                </option>
              ))}
            </select>
          </div>
          <div className="flex-[2] min-w-[300px]">
            <label className="block text-sm font-medium text-text mb-1">搜索</label>
            <div className="flex gap-2">
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="搜索邮箱或用户名..."
                className="flex-1 px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
                onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              />
              <Button onClick={handleSearch}>搜索</Button>
            </div>
          </div>
        </div>
      </div>

      {/* 用户列表 */}
      <div className="bg-white rounded-lg shadow-sm border border-border overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-border">
              <tr>
                <th className="px-4 py-3 text-left text-sm font-medium text-text">用户</th>
                <th className="px-4 py-3 text-center text-sm font-medium text-text">角色</th>
                <th className="px-4 py-3 text-center text-sm font-medium text-text">创建时间</th>
                <th className="px-4 py-3 text-right text-sm font-medium text-text">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {users.length === 0 ? (
                <tr>
                  <td colSpan={4} className="px-4 py-8 text-center text-text-light">
                    暂无用户
                  </td>
                </tr>
              ) : (
                users.map((user) => (
                  <tr key={user.id} className="hover:bg-gray-50">
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-3">
                        <div className="w-8 h-8 rounded-full bg-primary-100 flex items-center justify-center text-primary font-medium">
                          {user.email.charAt(0).toUpperCase()}
                        </div>
                        <div>
                          <div className="font-medium text-text">{user.email}</div>
                          <div className="text-xs text-text-light">ID: {user.id}</div>
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-center">
                      <span
                        className={`px-2 py-1 rounded-full text-xs font-medium ${roleColors[user.role]}`}
                      >
                        {roleLabels[user.role]}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-center text-text">
                      {new Date(user.created_at).toLocaleDateString()}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <div className="flex items-center justify-end gap-2">
                        <Button variant="secondary" size="sm" onClick={() => openEditModal(user)}>
                          编辑
                        </Button>
                        <Button variant="danger" size="sm" onClick={() => handleDelete(user)}>
                          删除
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* 创建用户弹窗 */}
      <Modal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        title="新建用户"
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-text mb-1">邮箱 *</label>
            <Input
              type="email"
              value={formData.email}
              onChange={(e) => setFormData({ ...formData, email: e.target.value })}
              placeholder="user@example.com"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-text mb-1">用户名 *</label>
            <Input
              type="text"
              value={formData.username}
              onChange={(e) => setFormData({ ...formData, username: e.target.value })}
              placeholder="用户名"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-text mb-1">密码 *</label>
            <Input
              type="password"
              value={formData.password}
              onChange={(e) => setFormData({ ...formData, password: e.target.value })}
              placeholder="至少6位"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-text mb-1">角色 *</label>
            <select
              value={formData.role}
              onChange={(e) => setFormData({ ...formData, role: e.target.value as UserRole })}
              className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              {(Object.keys(roleLabels) as UserRole[]).map((role) => (
                <option key={role} value={role}>
                  {roleLabels[role]}
                </option>
              ))}
            </select>
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setIsCreateModalOpen(false)}>
              取消
            </Button>
            <Button
              onClick={handleCreate}
              disabled={processing || !formData.email || !formData.username || !formData.password}
            >
              {processing ? '创建中...' : '创建'}
            </Button>
          </div>
        </div>
      </Modal>

      {/* 编辑用户弹窗 */}
      <Modal isOpen={isEditModalOpen} onClose={() => setIsEditModalOpen(false)} title="编辑用户">
        <div className="space-y-4">
          {selectedUser && (
            <div className="bg-gray-50 p-3 rounded-lg">
              <p className="text-sm text-text-light">
                编辑用户: <span className="font-medium text-text">{selectedUser.email}</span>
              </p>
            </div>
          )}
          <div>
            <label className="block text-sm font-medium text-text mb-1">用户名</label>
            <Input
              type="text"
              value={formData.username}
              onChange={(e) => setFormData({ ...formData, username: e.target.value })}
              placeholder="用户名"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-text mb-1">
              新密码（留空则不修改）
            </label>
            <Input
              type="password"
              value={formData.password}
              onChange={(e) => setFormData({ ...formData, password: e.target.value })}
              placeholder="不修改请留空"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-text mb-1">角色</label>
            <select
              value={formData.role}
              onChange={(e) => setFormData({ ...formData, role: e.target.value as UserRole })}
              className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              {(Object.keys(roleLabels) as UserRole[]).map((role) => (
                <option key={role} value={role}>
                  {roleLabels[role]}
                </option>
              ))}
            </select>
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setIsEditModalOpen(false)}>
              取消
            </Button>
            <Button onClick={handleUpdate} disabled={processing}>
              {processing ? '保存中...' : '保存'}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
