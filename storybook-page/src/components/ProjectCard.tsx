import { Link } from 'react-router-dom';
import type { Project } from '../types/models';
import { cn } from '../utils/cn';
import {
  BoardIcon,
  CrownIcon,
  SprintIcon,
  StoryIcon,
  UsersIcon,
} from './ui/AppIcon';

interface ProjectCardProps {
  project: Project;
}

export default function ProjectCard({ project }: ProjectCardProps) {
  const { id, name, description, agile_mode, member_count, story_count, is_owner, created_at } =
    project;
  const accentClass =
    agile_mode === 'scrum'
      ? 'from-primary-50 via-white to-primary-100/70'
      : 'from-accent-50 via-white to-accent-100/70';
  const badgeClass =
    agile_mode === 'scrum'
      ? 'bg-primary-100 text-primary-700'
      : 'bg-success-light text-success';
  const modeLabel = agile_mode === 'scrum' ? '冲刺' : '看板';

  return (
    <Link
      to={`/projects/${id}`}
      className={`card-hover block rounded-[1.6rem] border border-border bg-gradient-to-br ${accentClass} p-5 shadow-sm transition-all duration-200 hover:border-primary-200 sm:p-6`}
    >
      <div className="mb-5 flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-start gap-3">
          <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-white text-base font-semibold text-primary shadow-sm">
            {name[0]}
          </div>
          <div className="min-w-0">
            <h3 className="line-clamp-1 text-lg font-semibold text-text">{name}</h3>
            <p className="mt-1 text-sm text-text-light">进入项目总览、看板与缺陷管理</p>
          </div>
        </div>

        <div className="ml-2 flex-shrink-0">
          <span
            className={cn(
              'inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium',
              badgeClass
            )}
            title={`${modeLabel}项目`}
          >
            {agile_mode === 'scrum' ? (
              <>
                <SprintIcon size={14} />
                {modeLabel}
              </>
            ) : (
              <>
                <BoardIcon size={14} />
                {modeLabel}
              </>
            )}
          </span>
        </div>
      </div>

      {description ? (
        <p className="mb-5 line-clamp-2 min-h-[3rem] text-sm leading-6 text-text-light">
          {description}
        </p>
      ) : (
        <p className="mb-5 min-h-[3rem] text-sm leading-6 text-text-lighter">
          暂无项目描述，进入后可以继续补充目标、成员与工作节奏。
        </p>
      )}

      <div className="mb-5 flex flex-wrap items-center gap-2 text-sm text-text-light">
        <div
          className="inline-flex items-center gap-1.5 rounded-full bg-white px-2.5 py-1.5 shadow-sm"
          title={`${member_count ?? 0} 名成员`}
        >
          <UsersIcon size={14} />
          <span className="font-medium text-text">{member_count ?? 0}</span>
          <span className="text-xs text-text-light">成员</span>
        </div>
        <div
          className="inline-flex items-center gap-1.5 rounded-full bg-white px-2.5 py-1.5 shadow-sm"
          title={`${story_count ?? 0} 个故事`}
        >
          <StoryIcon size={14} />
          <span className="font-medium text-text">{story_count ?? 0}</span>
          <span className="text-xs text-text-light">故事</span>
        </div>
        {is_owner && (
          <div
            className="inline-flex items-center gap-1 rounded-full bg-amber-50 px-2.5 py-1.5 text-amber-700"
            title="项目负责人"
            aria-label="项目负责人"
          >
            <CrownIcon size={14} />
            <span className="text-xs font-medium">负责人</span>
          </div>
        )}
      </div>

      <div className="flex items-center justify-between border-t border-border/80 pt-4 text-xs text-text-light">
        <span>创建于 {new Date(created_at).toLocaleDateString()}</span>
        <span className="font-medium text-primary">查看项目</span>
      </div>
    </Link>
  );
}
