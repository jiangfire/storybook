import { Link, useLocation } from 'react-router-dom';
import { cn } from '../../utils/cn';
import { useAuthStore } from '../../stores/authStore';
import type { UserRole } from '../../types/models';
import {
  BoardIcon,
  ChartIcon,
  CheckCircleIcon,
  FolderIcon,
  PulseIcon,
  SparklesIcon,
  SprintIcon,
  UsersIcon,
} from '../ui/AppIcon';

interface MenuItem {
  path: string;
  icon: typeof FolderIcon;
  label: string;
  allowedRoles?: UserRole[];
}

const menuItems: MenuItem[] = [
  {
    path: '/projects',
    icon: FolderIcon,
    label: '项目',
  },
  {
    path: '/dashboard',
    icon: ChartIcon,
    label: '工作台',
  },
  {
    path: '/techlead/review',
    icon: CheckCircleIcon,
    label: '审批',
    allowedRoles: ['tech_lead', 'admin'],
  },
  {
    path: '/techlead/workload',
    icon: PulseIcon,
    label: '负载',
    allowedRoles: ['tech_lead', 'admin'],
  },
  {
    path: '/admin/users',
    icon: UsersIcon,
    label: '人员',
    allowedRoles: ['admin'],
  },
  {
    path: '/admin/ai',
    icon: SparklesIcon,
    label: 'AI',
    allowedRoles: ['admin'],
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
  const projectModeLabel = currentProject?.agile_mode === 'scrum' ? '冲刺模式' : '看板模式';

  return (
    <>
      <div className="md:hidden">
        <div className="section-card rounded-[1.2rem] px-3 py-3">
          <nav className="flex gap-2 overflow-x-auto">
            {visibleMenuItems.map((item) => {
              const Icon = item.icon;
              return (
                <Link
                  key={item.path}
                  to={item.path}
                  className={cn(
                    'inline-flex min-w-fit items-center gap-1.5 rounded-full border px-3 py-2 text-sm transition-colors',
                    isActivePath(item.path)
                      ? 'border-primary-200 bg-primary-50 text-primary'
                      : 'border-border bg-white text-text-light hover:bg-primary-50'
                  )}
                >
                  <Icon size={15} />
                  <span>{item.label}</span>
                </Link>
              );
            })}
          </nav>

          {currentProject && (
            <div className="mt-3 border-t border-border pt-3">
              <Link
                to={`/projects/${currentProject.id}`}
                className="section-block flex items-center justify-between rounded-2xl px-4 py-3 transition-colors hover:border-primary-200 hover:bg-white"
              >
                <div className="min-w-0">
                  <div className="truncate text-sm font-medium text-text">{currentProject.name}</div>
                  <div className="mt-1 inline-flex items-center gap-1 text-xs text-text-light">
                    {currentProject.agile_mode === 'scrum' ? (
                      <SprintIcon size={12} />
                    ) : (
                      <BoardIcon size={12} />
                    )}
                    <span>{projectModeLabel}</span>
                  </div>
                </div>
                <span className="rounded-full bg-white px-2 py-1 text-[11px] text-text-light">
                  详情
                </span>
              </Link>
            </div>
          )}
        </div>
      </div>

      <aside className="sticky top-3 hidden w-72 self-start md:block">
        <div className="section-card flex max-h-[calc(100dvh-2rem)] min-h-[calc(100dvh-2rem)] flex-col rounded-[1.5rem] p-3">
          <div className="px-1 pb-2 text-[11px] font-medium text-text-light">
            主导航
          </div>

          <nav className="flex-1 space-y-1 overflow-y-auto pb-3">
            {visibleMenuItems.map((item) => {
              const Icon = item.icon;
              return (
                <Link
                  key={item.path}
                  to={item.path}
                  className={cn(
                    'flex items-center gap-2.5 rounded-2xl px-3 py-3 transition-colors',
                    isActivePath(item.path)
                      ? 'bg-primary-50 text-primary'
                      : 'text-text-light hover:bg-primary-50'
                  )}
                >
                  <span
                    className={cn(
                      'inline-flex h-9 w-9 items-center justify-center rounded-xl border',
                      isActivePath(item.path)
                        ? 'border-primary-100 bg-white text-primary'
                        : 'border-border bg-white text-text-light'
                    )}
                  >
                    <Icon size={18} />
                  </span>
                  <span className="text-sm font-medium">{item.label}</span>
                </Link>
              );
            })}
          </nav>

          {currentProject && (
            <div className="space-y-3 border-t border-border pt-3">
              <div className="px-1 text-[11px] font-medium text-text-light">
                当前项目
              </div>
              <div className="section-block rounded-2xl p-4">
                <div className="mb-4 flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="truncate text-sm font-semibold text-text">
                      {currentProject.name}
                    </div>
                    <div className="mt-1 inline-flex items-center gap-1 text-xs text-text-light">
                      {currentProject.agile_mode === 'scrum' ? (
                        <SprintIcon size={12} />
                      ) : (
                        <BoardIcon size={12} />
                      )}
                      <span>{projectModeLabel}</span>
                    </div>
                  </div>
                  <Link
                    to={`/projects/${currentProject.id}`}
                    className="rounded-full border border-border bg-white px-2.5 py-1 text-[11px] text-text-light transition-colors hover:border-primary-200 hover:text-primary"
                  >
                    查看
                  </Link>
                </div>
                <div className="text-xs leading-5 text-text-light">
                  从这里确认当前上下文，具体操作交给页面正文完成。
                </div>
              </div>
            </div>
          )}
        </div>
      </aside>
    </>
  );
}
