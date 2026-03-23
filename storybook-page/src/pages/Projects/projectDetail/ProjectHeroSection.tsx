import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';
import Button from '../../../components/ui/Button';
import {
  BoardIcon,
  CheckCircleIcon,
  CrownIcon,
  SprintIcon,
  StoryIcon,
  UsersIcon,
  WrenchIcon,
} from '../../../components/ui/AppIcon';
import type { Project } from '../../../types/models';

interface ProjectHeroSectionProps {
  project: Project | null;
  projectID: number;
  projectModeLabel: string;
  totalStories: number;
  inProgressStories: number;
  completionRate: number;
  activeMembers: number;
  canCreateStory: boolean;
}

function MetricCard({
  icon,
  label,
  value,
  valueClassName,
}: {
  icon: ReactNode;
  label: string;
  value: number | string;
  valueClassName: string;
}) {
  return (
    <div className="hero-subcard rounded-[1.4rem] p-4">
      <div className="text-xs font-medium text-text-light">{label}</div>
      <div className={`mt-2 inline-flex items-center gap-2 text-2xl font-semibold ${valueClassName}`}>
        {icon}
        {value}
      </div>
    </div>
  );
}

export function ProjectHeroSection({
  project,
  projectID,
  projectModeLabel,
  totalStories,
  inProgressStories,
  completionRate,
  activeMembers,
  canCreateStory,
}: ProjectHeroSectionProps) {
  return (
    <div className="space-y-4 px-5 py-5 sm:px-6 lg:px-7 lg:py-6">
      <div className="space-y-4">
        <div className="space-y-3">
          <div className="flex flex-wrap items-center gap-2">
            <span className="inline-flex items-center gap-2 rounded-full bg-primary-50 px-3 py-1 text-xs font-medium text-primary">
              {project?.agile_mode === 'scrum' ? <SprintIcon size={14} /> : <BoardIcon size={14} />}
              {projectModeLabel}
            </span>
            {project?.owner && (
              <span className="inline-flex items-center gap-1 rounded-full bg-amber-50 px-3 py-1 text-xs font-medium text-amber-700">
                <CrownIcon size={12} />
                负责人 {project.owner.email}
              </span>
            )}
          </div>
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-text sm:text-4xl">
              {project?.name || '项目详情'}
            </h1>
            <p className="mt-2 max-w-3xl text-sm leading-6 text-text-light sm:text-base">
              {project?.description || '暂无项目描述。这里查看项目节奏、质量、成员和冲刺进展。'}
            </p>
          </div>
        </div>

        <div className="hero-subcard rounded-[1.7rem] p-3">
          <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
            <MetricCard
              icon={<StoryIcon size={20} />}
              label="总故事数"
              value={totalStories}
              valueClassName="text-text"
            />
            <MetricCard
              icon={<WrenchIcon size={20} />}
              label="推进中"
              value={inProgressStories}
              valueClassName="text-primary"
            />
            <MetricCard
              icon={<CheckCircleIcon size={20} />}
              label="完成率"
              value={`${completionRate.toFixed(1)}%`}
              valueClassName="text-success"
            />
            <MetricCard
              icon={<UsersIcon size={20} />}
              label="活跃成员"
              value={activeMembers}
              valueClassName="text-text"
            />
          </div>
        </div>
      </div>

      <div className="hero-subcard rounded-[1.6rem] p-3.5 sm:p-4">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h2 className="text-base font-semibold text-text sm:text-lg">工作入口</h2>
            <p className="mt-1 text-sm text-text-light">
              从这里继续进入看板、缺陷或故事创建，首屏统计只负责说明项目状态。
            </p>
          </div>
          <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:justify-end">
            <Link to={`/projects/${projectID}/board`}>
              <Button className="w-full sm:w-auto">进入看板</Button>
            </Link>
            <Link to={`/projects/${projectID}/bugs`}>
              <Button variant="secondary" className="w-full sm:w-auto">
                缺陷管理
              </Button>
            </Link>
            {canCreateStory ? (
              <Link to={`/projects/${projectID}/stories/new`}>
                <Button variant="secondary" className="w-full sm:w-auto">
                  创建故事
                </Button>
              </Link>
            ) : (
              <div className="rounded-xl border border-white/70 bg-white/70 px-3 py-2 text-xs text-text-light">
                当前角色无创建故事权限
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
