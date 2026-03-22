import { Link, useLocation } from 'react-router-dom';
import { cn } from '../../utils/cn';
import { useAuthStore } from '../../stores/authStore';
import type { UserRole } from '../../types/models';
import {
  BoardIcon,
  BugIcon,
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

interface QuickLink {
  path: string;
  label: string;
  icon: typeof FolderIcon;
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
  const projectQuickLinks: QuickLink[] = currentProject
    ? [
        {
          path: `/projects/${currentProject.id}/board`,
          label: '项目看板',
          icon: BoardIcon,
        },
        {
          path: `/projects/${currentProject.id}/bugs`,
          label: '缺陷管理',
          icon: BugIcon,
        },
        {
          path: `/projects/${currentProject.id}/stories/new`,
          label: '新建故事',
          icon: SparklesIcon,
          allowedRoles: ['product', 'admin'],
        },
      ]
    : [];
  const visibleQuickLinks = projectQuickLinks.filter(canAccessMenu);

  return (
    <>
      <div className="mb-3 md:hidden">
        <div className="surface-card rounded-[1.4rem] px-3 py-3">
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
                className="surface-card flex items-center justify-between rounded-2xl px-4 py-3 transition-colors hover:border-primary-200 hover:bg-white"
              >
                <div className="min-w-0">
                  <div className="truncate text-sm font-medium text-text">{currentProject.name}</div>
                  <div className="mt-1 inline-flex items-center gap-1 text-xs text-text-light">
                    {currentProject.agile_mode === 'scrum' ? (
                      <SprintIcon size={12} />
                    ) : (
                      <BoardIcon size={12} />
                    )}
                    <span>{currentProject.agile_mode === 'scrum' ? 'Scrum' : 'Kanban'}</span>
                  </div>
                </div>
                <span className="text-xs text-text-light">详情</span>
              </Link>

              {visibleQuickLinks.length > 0 && (
                <div className="mt-2 flex gap-2 overflow-x-auto">
                  {visibleQuickLinks.map((item) => {
                    const Icon = item.icon;
                    return (
                      <Link
                        key={item.path}
                        to={item.path}
                        className={cn(
                          'inline-flex min-w-fit items-center gap-1.5 rounded-full border px-3 py-2 text-xs transition-colors',
                          isActivePath(item.path)
                            ? 'border-primary-200 bg-primary-50 text-primary'
                            : 'border-border bg-white text-text-light hover:bg-primary-50'
                        )}
                      >
                        <Icon size={14} />
                        <span>{item.label}</span>
                      </Link>
                    );
                  })}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      <aside className="sticky top-[104px] hidden w-72 self-start md:block">
        <div className="surface-card flex max-h-[calc(100dvh-7rem)] min-h-[calc(100dvh-7rem)] flex-col rounded-[1.75rem] p-3">
          <div className="mb-3 rounded-2xl bg-gradient-to-br from-primary to-primary-700 px-4 py-4 text-white">
            <div className="text-xs uppercase tracking-[0.24em] text-white/70">Workspace</div>
            <div className="mt-2 text-lg font-semibold">导航</div>
            <div className="mt-1 text-sm text-white/75">在同一套流程里切换项目、审批和管理。</div>
          </div>

          <nav className="flex-1 space-y-1 overflow-y-auto pb-2">
            {visibleMenuItems.map((item) => {
              const Icon = item.icon;
              return (
                <Link
                  key={item.path}
                  to={item.path}
                  className={cn(
                    'flex items-center gap-2.5 rounded-2xl px-3 py-3 transition-colors',
                    isActivePath(item.path)
                      ? 'bg-primary-50 text-primary shadow-sm'
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
              <div className="px-1 text-[11px] font-semibold uppercase tracking-[0.18em] text-text-light">
                当前项目
              </div>
              <Link
                to={`/projects/${currentProject.id}`}
                className="rounded-2xl border border-primary-100 bg-gradient-to-br from-white via-white to-primary-50 p-4 transition-colors hover:border-primary-200"
              >
                <div className="mb-3 flex items-start justify-between gap-3">
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
                      <span>{currentProject.agile_mode === 'scrum' ? 'Scrum' : 'Kanban'}</span>
                    </div>
                  </div>
                  <span className="rounded-full bg-white px-2 py-1 text-[11px] text-text-light shadow-sm">
                    详情
                  </span>
                </div>
                <div className="text-xs leading-5 text-text-light">
                  从这里继续看板、缺陷和故事创建。
                </div>
              </Link>

              {visibleQuickLinks.length > 0 && (
                <div className="space-y-1">
                  {visibleQuickLinks.map((item) => {
                    const Icon = item.icon;
                    return (
                      <Link
                        key={item.path}
                        to={item.path}
                        className={cn(
                          'flex items-center justify-between rounded-xl px-3 py-2.5 text-sm transition-colors',
                          isActivePath(item.path)
                            ? 'bg-primary-50 text-primary'
                            : 'text-text-light hover:bg-primary-50'
                        )}
                      >
                        <span className="inline-flex items-center gap-2">
                          <Icon size={15} />
                          <span>{item.label}</span>
                        </span>
                        <span aria-hidden>→</span>
                      </Link>
                    );
                  })}
                </div>
              )}
            </div>
          )}
        </div>
      </aside>
    </>
  );
}
