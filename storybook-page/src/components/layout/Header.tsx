import { Link, useNavigate } from 'react-router-dom';
import { useAuthStore } from '../../stores/authStore';
import { getUserInitials } from '../../utils/formatters';

export default function Header() {
  const navigate = useNavigate();
  const { user, logout } = useAuthStore();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <header className="h-16 bg-white border-b border-border flex items-center justify-between px-6 sticky top-0 z-50 glass">
      {/* Logo */}
      <Link to="/projects" className="flex items-center space-x-3">
        <div className="w-8 h-8 bg-gradient-to-br from-primary to-accent rounded-lg flex items-center justify-center">
          <span className="text-white font-bold text-lg">S</span>
        </div>
        <h1 className="font-display text-xl font-bold text-primary">Storybook</h1>
      </Link>

      {/* 右侧用户菜单 */}
      <div className="flex items-center space-x-4">
        {/* 用户信息 */}
        <div className="flex items-center space-x-3">
          <div className="text-right">
            <div className="text-sm font-medium text-text">{user?.email}</div>
            <div className="text-xs text-text-light">
              {user?.role === 'product' && '产品经理'}
              {user?.role === 'developer' && '开发人员'}
              {user?.role === 'tester' && '测试人员'}
            </div>
          </div>

          {/* 头像 */}
          <div className="relative group">
            <div className="w-10 h-10 bg-primary-100 text-primary rounded-full flex items-center justify-center font-medium cursor-pointer hover:bg-primary-200 transition-colors">
              {user?.email && getUserInitials(user.email)}
            </div>

            {/* 下拉菜单 */}
            <div className="absolute right-0 top-full mt-2 w-48 bg-white rounded-lg shadow-lg border border-border opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all duration-200">
              <div className="py-2">
                <Link
                  to="/dashboard"
                  className="block px-4 py-2 text-sm text-text hover:bg-gray-50 transition-colors"
                >
                  个人工作台
                </Link>
                <button
                  onClick={handleLogout}
                  className="w-full text-left px-4 py-2 text-sm text-danger hover:bg-danger-light transition-colors"
                >
                  退出登录
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </header>
  );
}
