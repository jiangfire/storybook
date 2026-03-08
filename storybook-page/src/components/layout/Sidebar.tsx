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

interface QuickLink {
  path: string;
  label: string;
  icon: string;
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
  const projectQuickLinks: QuickLink[] = currentProject
    ? [
        {
          path: `/projects/${currentProject.id}/board`,
          label: '项目看板',
          icon: '🧩',
        },
        {
          path: `/projects/${currentProject.id}/bugs`,
          label: '缺陷管理',
          icon: '🐞',
        },
        {
          path: `/projects/${currentProject.id}/stories/new`,
          label: '新建故事',
          icon: '✨',
          allowedRoles: ['product', 'admin'],
        },
      ]
    : [];
  const visibleQuickLinks = projectQuickLinks.filter(canAccessMenu);

  return (
    <aside className="hidden md:flex w-64 bg-white border-r border-border flex-col sticky top-14 self-start h-[calc(100dvh-3.5rem)]">
      <div className="flex h-full flex-col">
        {/* 导航菜单 */}
        <nav className="flex-1 overflow-y-auto p-3 space-y-1">
          {visibleMenuItems.map((item) => (
            <Link
              key={item.path}
              to={item.path}
              className={cn(
                'flex items-center space-x-2.5 px-3 py-2.5 rounded-lg transition-colors',
                isActivePath(item.path)
                  ? 'bg-primary-50 text-primary'
                  : 'text-text-light hover:bg-gray-50'
              )}
            >
              <span className="text-lg">{item.icon}</span>
              <span className="text-sm font-medium">{item.label}</span>
            </Link>
          ))}
        </nav>

        {/* 当前项目信息 */}
        {currentProject && (
          <div className="p-3 border-t border-border space-y-2">
            <div className="text-[11px] font-semibold tracking-wide text-text-light uppercase">
              当前项目
            </div>
            <Link
              to={`/projects/${currentProject.id}`}
              className="block p-3 rounded-lg bg-gray-50 hover:bg-gray-100 transition-colors"
            >
              <div className="font-medium text-text text-sm mb-1 truncate">{currentProject.name}</div>
              <div className="text-xs text-text-light flex items-center justify-between">
                <span>{currentProject.agile_mode === 'scrum' ? '🏃 Scrum' : '📋 Kanban'}</span>
                <span>详情 →</span>
              </div>
            </Link>

            {visibleQuickLinks.length > 0 && (
              <div className="space-y-1 pt-1">
                {visibleQuickLinks.map((item) => (
                  <Link
                    key={item.path}
                    to={item.path}
                    className={cn(
                      'flex items-center justify-between px-2.5 py-2 rounded-md text-xs transition-colors',
                      isActivePath(item.path)
                        ? 'bg-primary-50 text-primary'
                        : 'text-text-light hover:bg-gray-50'
                    )}
                  >
                    <span className="inline-flex items-center gap-1.5">
                      <span>{item.icon}</span>
                      {item.label}
                    </span>
                    <span aria-hidden>→</span>
                  </Link>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </aside>
  );
}
