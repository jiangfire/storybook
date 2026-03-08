import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuthStore } from '../../stores/authStore';
import { isValidEmail, isValidPassword } from '../../utils/validators';

export default function LoginPage() {
  const navigate = useNavigate();
  const { login, isLoading, error, clearError } = useAuthStore();

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [fieldErrors, setFieldErrors] = useState<{
    email?: string;
    password?: string;
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

    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    clearError();
    if (!validateForm()) return;

    try {
      await login({ email, password });
      navigate('/projects');
    } catch {
      // 错误已在 store 中处理
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-primary-50 to-accent-50 px-4">
      <div className="max-w-md w-full">
        {/* Logo 和标题 */}
        <div className="text-center mb-8 animate-slideUp">
          <h1 className="font-display text-4xl font-bold text-primary mb-2">Storybook</h1>
          <p className="text-text-light">故事之书，记录每一个用户故事的完整生命周期</p>
        </div>

        {/* 登录表单 */}
        <div className="bg-white rounded-2xl shadow-lg p-8 animate-slideUp">
          <h2 className="text-2xl font-semibold text-text mb-6">登录</h2>

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
                placeholder="••••••••"
                disabled={isLoading}
              />
              {fieldErrors.password && (
                <p className="mt-1 text-sm text-danger">{fieldErrors.password}</p>
              )}
            </div>

            {/* 错误提示 */}
            {error && (
              <div className="bg-danger-light text-danger px-4 py-3 rounded-lg text-sm">
                {error}
              </div>
            )}

            {/* 登录按钮 */}
            <button
              type="submit"
              disabled={isLoading}
              className="btn btn-primary w-full py-3 text-base"
            >
              {isLoading ? '登录中...' : '登录'}
            </button>
          </form>

          {/* 注册链接 */}
          <p className="mt-6 text-center text-sm text-text-light">
            还没有账号？{' '}
            <Link to="/register" className="text-primary-600 hover:text-primary-700 font-medium">
              立即注册
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
