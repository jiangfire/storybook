import { Link, useLocation } from 'react-router-dom';
import { cn } from '../../utils/cn';
import { useAuthStore } from '../../stores/authStore';
import type { UserRole } from '../../types/models';

interface MenuItem {
  path: string;
  icon: string;
  label: string;
  allowedRoles?: UserRole[];
}

const menuItems: MenuItem[] = [
  {
    path: '/projects',
    icon: '📁',
    label: '项目',
  },
  {
    path: '/dashboard',
    icon: '📊',
    label: '工作台',
  },
  {
    path: '/techlead/review',
    icon: '✅',
    label: '审批',
    allowedRoles: ['tech_lead', 'admin'],
  },
  {
    path: '/techlead/workload',
    icon: '📈',
    label: '负载',
    allowedRoles: ['tech_lead', 'admin'],
  },
  {
    path: '/admin/users',
    icon: '👥',
    label: '人员',
    allowedRoles: ['tech_lead', 'admin'],
  },
];

interface SidebarProps {
  currentProject?: {
    id: number;
    name: string;
    agile_mode: string;
  } | null;
}

export default function Sidebar({ currentProject }: SidebarProps) {
  const location = useLocation();
  const { user } = useAuthStore();

  const isActivePath = (path: string) => {
    return location.pathname === path || location.pathname.startsWith(path + '/');
  };

  const canAccessMenu = (item: MenuItem) => {
    if (!item.allowedRoles) return true;
    if (!user) return false;
    return item.allowedRoles.includes(user.role);
  };

  const visibleMenuItems = menuItems.filter(canAccessMenu);

  return (
    <aside className="hidden md:flex w-64 bg-white border-r border-border flex-col h-[calc(100vh-4rem)]">
      {/* 导航菜单 */}
      <nav className="flex-1 p-4 space-y-1">
        {visibleMenuItems.map((item) => (
          <Link
            key={item.path}
            to={item.path}
            className={cn(
              'flex items-center space-x-3 px-4 py-3 rounded-lg transition-colors',
              isActivePath(item.path)
                ? 'bg-primary-50 text-primary'
                : 'text-text-light hover:bg-gray-50'
            )}
          >
            <span className="text-xl">{item.icon}</span>
            <span className="font-medium">{item.label}</span>
          </Link>
        ))}
      </nav>

      {/* 当前项目信息 */}
      {currentProject && (
        <div className="p-4 border-t border-border">
          <div className="text-xs font-medium text-text-light uppercase mb-2">当前项目</div>
          <Link
            to={`/projects/${currentProject.id}`}
            className="block p-3 rounded-lg bg-gray-50 hover:bg-gray-100 transition-colors"
          >
            <div className="font-medium text-text text-sm mb-1">{currentProject.name}</div>
            <div className="text-xs text-text-light flex items-center justify-between">
              <span>{currentProject.agile_mode === 'scrum' ? '🏃 Scrum' : '📋 Kanban'}</span>
              <span>查看详情 →</span>
            </div>
          </Link>
        </div>
      )}
    </aside>
  );
}
