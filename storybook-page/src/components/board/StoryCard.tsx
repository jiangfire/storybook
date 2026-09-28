import type { StoryBoardItem } from '../../types/models';
import {
  formatPriority,
  formatStoryType,
  getPriorityColor,
  getStoryTypeColor,
  getUserInitials,
  calculatePercentage,
} from '../../utils/formatters';
import { useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { useNavigate } from 'react-router-dom';
import { GripIcon } from '../ui/AppIcon';

interface StoryCardProps {
  story: StoryBoardItem;
}

export default function StoryCard({ story }: StoryCardProps) {
  const navigate = useNavigate();
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: story.id,
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  // 计算AC完成度
  const acTotal =
    story.acceptance_criteria_summary?.total || story.acceptance_criteria?.length || 0;
  const acPassed =
    story.acceptance_criteria_summary?.passed ||
    story.acceptance_criteria?.filter((ac) => ac.status === 'passed').length ||
    0;
  const acPercentage = acTotal > 0 ? calculatePercentage(acPassed, acTotal) : 0;

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="bg-white rounded-lg shadow-sm border border-border p-4 transition-all duration-200 hover:shadow-md"
    >
      {/* 类型标签 + 拖拽把手 */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-1.5">
          <button
            type="button"
            {...attributes}
            {...listeners}
            aria-label={`拖拽排序 ${story.title}`}
            className="touch-none rounded p-0.5 text-text-light transition-colors hover:bg-secondary-100 hover:text-text active:cursor-grabbing cursor-grab"
          >
            <GripIcon size={14} />
          </button>
          <span
            className={`text-xs px-2 py-1 rounded-full font-medium ${getStoryTypeColor(story.story_type)}`}
          >
            {formatStoryType(story.story_type)}
          </span>
        </div>

        <span
          className={`rounded-full px-2 py-1 text-xs font-medium ${getPriorityColor(story.priority ?? 0)}`}
        >
          {formatPriority(story.priority ?? 0)}
        </span>
      </div>

      {/* 标题 */}
      <button
        type="button"
        onClick={() => navigate(`/stories/${story.id}`)}
        className="block w-full text-left font-medium text-text mb-3 line-clamp-2 min-h-[2.5rem]"
      >
        {story.title}
      </button>

      {/* 验收标准进度 */}
      {acTotal > 0 && (
        <div className="mb-3">
          <div className="flex items-center justify-between text-xs text-text-light mb-1">
            <span>AC</span>
            <span>
              {acPassed}/{acTotal}
            </span>
          </div>
          <div className="w-full bg-secondary-200 rounded-full h-2">
            <div
              className="bg-success h-2 rounded-full transition-all duration-300"
              style={{ width: `${acPercentage}%` }}
            />
          </div>
        </div>
      )}

      {/* 底部信息 */}
      <div className="flex items-center justify-between pt-3 border-t border-border-light">
        <div className="flex items-center gap-2">
          {story.story_points && (
            <div className="flex items-center space-x-1">
              <span className="text-xs text-text-light">点数:</span>
              <span className="text-sm font-medium text-text">{story.story_points}</span>
            </div>
          )}
        </div>

        {/* 负责人 */}
        {story.assigned_to ? (
          <div
            className="w-8 h-8 rounded-full bg-primary-100 text-primary flex items-center justify-center text-xs font-medium"
            title={story.assigned_to.email}
          >
            {getUserInitials(story.assigned_to.email)}
          </div>
        ) : (
          <div
            className="w-8 h-8 rounded-full bg-secondary-100 text-text-light flex items-center justify-center text-xs"
            title="未分配"
          >
            ?
          </div>
        )}
      </div>
    </div>
  );
}
