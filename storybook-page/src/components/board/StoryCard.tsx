import { StoryBoardItem } from '../../types/models';
import {
  formatStoryType,
  getStoryTypeColor,
  getUserInitials,
  calculatePercentage,
} from '../../utils/formatters';
import { useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { useNavigate } from 'react-router-dom';

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
      {...attributes}
      {...listeners}
      className="bg-white rounded-lg shadow-sm border border-gray-200 p-4 cursor-grab active:cursor-grabbing hover:shadow-md transition-all duration-200"
      onDoubleClick={() => navigate(`/stories/${story.id}`)}
    >
      {/* 类型标签 */}
      <div className="flex items-center justify-between mb-3">
        <span
          className={`text-xs px-2 py-1 rounded-full font-medium ${getStoryTypeColor(story.story_type)}`}
        >
          {formatStoryType(story.story_type)}
        </span>

        {/* 优先级 */}
        <div className="flex items-center space-x-1">
          {Array.from({ length: story.priority || 0 }).map((_, i) => (
            <span key={i} className="text-xs">
              🔴
            </span>
          ))}
        </div>
      </div>

      {/* 标题 */}
      <h4 className="font-medium text-text mb-3 line-clamp-2 min-h-[2.5rem]">{story.title}</h4>

      {/* 验收标准进度 */}
      {acTotal > 0 && (
        <div className="mb-3">
          <div className="flex items-center justify-between text-xs text-text-light mb-1">
            <span>AC</span>
            <span>
              {acPassed}/{acTotal}
            </span>
          </div>
          <div className="w-full bg-gray-200 rounded-full h-2">
            <div
              className="bg-success h-2 rounded-full transition-all duration-300"
              style={{ width: `${acPercentage}%` }}
            />
          </div>
        </div>
      )}

      {/* 底部信息 */}
      <div className="flex items-center justify-between pt-3 border-t border-gray-100">
        {/* 故事点 */}
        {story.story_points && (
          <div className="flex items-center space-x-1">
            <span className="text-xs text-text-light">点数:</span>
            <span className="text-sm font-medium text-text">{story.story_points}</span>
          </div>
        )}

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
            className="w-8 h-8 rounded-full bg-gray-100 text-gray-400 flex items-center justify-center text-xs"
            title="未分配"
          >
            ?
          </div>
        )}
      </div>
    </div>
  );
}
