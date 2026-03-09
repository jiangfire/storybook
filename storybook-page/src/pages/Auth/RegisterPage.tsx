import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuthStore } from '../../stores/authStore';
import type { RegisterRequest } from '../../types/api';
import { isValidEmail, isValidPassword } from '../../utils/validators';

const roleOptions: Array<{
  value: RegisterRequest['role'];
  label: string;
  emoji: string;
  description: string;
}> = [
  {
    value: 'product',
    label: '产品经理',
    emoji: '📋',
    description: '创建故事、管理需求',
  },
  {
    value: 'developer',
    label: '开发人员',
    emoji: '💻',
    description: '领取故事、开发功能',
  },
  {
    value: 'tester',
    label: '测试人员',
    emoji: '🔍',
    description: '验收测试、提交Bug',
  },
];

export default function RegisterPage() {
  const navigate = useNavigate();
  const { register, isLoading, error, clearError } = useAuthStore();

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [role, setRole] = useState<RegisterRequest['role']>('developer');
  const [fieldErrors, setFieldErrors] = useState<{
    email?: string;
    password?: string;
    confirmPassword?: string;
  }>({});

  const validateForm = () => {
    const errors: typeof fieldErrors = {};

    if (!email) {
      errors.email = '请输入邮箱';
    } else if (!isValidEmail(email)) {
      errors.email = '邮箱格式不正确';
    }

    if (!password) {
      errors.password = '请输入密码';
    } else if (!isValidPassword(password)) {
      errors.password = '密码至少8位，包含字母和数字';
    }

    if (!confirmPassword) {
      errors.confirmPassword = '请确认密码';
    } else if (password !== confirmPassword) {
      errors.confirmPassword = '两次输入的密码不一致';
    }

    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    clearError();
    if (!validateForm()) return;

    try {
      await register({ email, password, role });
      navigate('/projects');
    } catch {
      // 错误已在 store 中处理
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-primary-50 to-accent-50 px-4 py-8">
      <div className="max-w-md w-full">
        {/* Logo 和标题 */}
        <div className="text-center mb-8 animate-slideUp">
          <h1 className="font-display text-4xl font-bold text-primary mb-2">Storybook</h1>
          <p className="text-text-light">创建账号，开始你的敏捷之旅</p>
        </div>

        {/* 注册表单 */}
        <div className="bg-white rounded-2xl shadow-lg p-8 animate-slideUp">
          <h2 className="text-2xl font-semibold text-text mb-6">注册</h2>

          <form onSubmit={handleSubmit} className="space-y-4">
            {/* 邮箱输入 */}
            <div>
              <label htmlFor="email" className="block text-sm font-medium text-text mb-2">
                邮箱
              </label>
              <input
                id="email"
                type="email"
                value={email}
                onChange={(e) => {
                  setEmail(e.target.value);
                  if (fieldErrors.email) {
                    setFieldErrors({ ...fieldErrors, email: undefined });
                  }
                }}
                className={`input ${fieldErrors.email ? 'border-danger' : ''}`}
                placeholder="your@email.com"
                disabled={isLoading}
              />
              {fieldErrors.email && <p className="mt-1 text-sm text-danger">{fieldErrors.email}</p>}
            </div>

            {/* 角色选择 */}
            <div>
              <label className="block text-sm font-medium text-text mb-2">角色</label>
              <div className="grid grid-cols-3 gap-3">
                {roleOptions.map((item) => (
                  <button
                    key={item.value}
                    type="button"
                    onClick={() => setRole(item.value)}
                    className={`p-3 rounded-lg border-2 transition-all text-left ${
                      role === item.value
                        ? 'border-primary bg-primary-50 text-primary'
                        : 'border-border hover:border-primary-300'
                    }`}
                    disabled={isLoading}
                  >
                    <div className="text-2xl mb-1">{item.emoji}</div>
                    <div className="text-xs font-medium mb-1">{item.label}</div>
                    <div className="text-xs text-text-light">{item.description}</div>
                  </button>
                ))}
              </div>
            </div>

            {/* 密码输入 */}
            <div>
              <label htmlFor="password" className="block text-sm font-medium text-text mb-2">
                密码
              </label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => {
                  setPassword(e.target.value);
                  if (fieldErrors.password) {
                    setFieldErrors({ ...fieldErrors, password: undefined });
                  }
                }}
                className={`input ${fieldErrors.password ? 'border-danger' : ''}`}
                placeholder="至少8位，包含字母和数字"
                disabled={isLoading}
              />
              {fieldErrors.password && (
                <p className="mt-1 text-sm text-danger">{fieldErrors.password}</p>
              )}
            </div>

            {/* 确认密码 */}
            <div>
              <label htmlFor="confirmPassword" className="block text-sm font-medium text-text mb-2">
                确认密码
              </label>
              <input
                id="confirmPassword"
                type="password"
                value={confirmPassword}
                onChange={(e) => {
                  setConfirmPassword(e.target.value);
                  if (fieldErrors.confirmPassword) {
                    setFieldErrors({ ...fieldErrors, confirmPassword: undefined });
                  }
                }}
                className={`input ${fieldErrors.confirmPassword ? 'border-danger' : ''}`}
                placeholder="再次输入密码"
                disabled={isLoading}
              />
              {fieldErrors.confirmPassword && (
                <p className="mt-1 text-sm text-danger">{fieldErrors.confirmPassword}</p>
              )}
            </div>

            {/* 错误提示 */}
            {error && (
              <div className="bg-danger-light text-danger px-4 py-3 rounded-lg text-sm">
                {error}
              </div>
            )}

            {/* 注册按钮 */}
            <button
              type="submit"
              disabled={isLoading}
              className="btn btn-primary w-full py-3 text-base"
            >
              {isLoading ? '注册中...' : '注册'}
            </button>
          </form>

          {/* 登录链接 */}
          <p className="mt-6 text-center text-sm text-text-light">
            已有账号？{' '}
            <Link to="/login" className="text-primary-600 hover:text-primary-700 font-medium">
              立即登录
            </Link>
          </p>
        </div>

        {/* 底部信息 */}
        <p className="mt-8 text-center text-xs text-text-lighter">
          © 2025 Storybook. 敏捷项目管理工具
        </p>
      </div>
    </div>
  );
}
