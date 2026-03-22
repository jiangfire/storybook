import { Link } from 'react-router-dom';
import type { Project } from '../types/models';
import { cn } from '../utils/cn';

interface ProjectCardProps {
  project: Project;
}

export default function ProjectCard({ project }: ProjectCardProps) {
  const { id, name, description, agile_mode, member_count, story_count, is_owner, created_at } =
    project;

  return (
    <Link
      to={`/projects/${id}`}
      className="block bg-white rounded-xl shadow-sm hover:shadow-lg transition-all duration-200 border border-border p-6 card-hover"
    >
      {/* 头部 */}
      <div className="flex items-start justify-between mb-4">
        <div className="flex-1">
          <h3 className="text-lg font-semibold text-text mb-2 line-clamp-1">{name}</h3>
          {description && (
            <p className="text-sm text-text-light line-clamp-2 mb-3">{description}</p>
          )}
        </div>

        {/* 敏捷模式标签 */}
        <div className="flex-shrink-0 ml-4">
          <span
            className={cn(
              'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium',
              agile_mode === 'scrum'
                ? 'bg-primary-100 text-primary-700'
                : 'bg-success-light text-success'
            )}
          >
            {agile_mode === 'scrum' ? '🏃 Scrum' : '📋 Kanban'}
          </span>
        </div>
      </div>

      {/* 统计信息 */}
      <div className="flex items-center space-x-4 text-sm text-text-light mb-4">
        <div className="flex items-center space-x-1">
          <span>👥</span>
          <span>{member_count} 成员</span>
        </div>
        <div className="flex items-center space-x-1">
          <span>📝</span>
          <span>{story_count} 故事</span>
        </div>
        {is_owner && (
          <div className="flex items-center space-x-1">
            <span>👑</span>
            <span>Owner</span>
          </div>
        )}
      </div>

      {/* 底部信息 */}
      <div className="pt-4 border-t border-border flex items-center justify-between text-xs text-text-light">
        <span>创建于 {new Date(created_at).toLocaleDateString()}</span>
        <span className="text-primary hover:text-primary-700 font-medium">查看详情 →</span>
      </div>
    </Link>
  );
}
