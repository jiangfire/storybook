import { useDroppable } from '@dnd-kit/core';
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable';
import type { StoryBoardItem } from '../../types/models';
import StoryCard from './StoryCard';
import { cn } from '../../utils/cn';
import { InboxIcon } from '../ui/AppIcon';

interface BoardColumnProps {
  id: string;
  stories: StoryBoardItem[];
  title: string;
  icon: typeof InboxIcon;
  count: number;
}

const columnColors: Record<string, string> = {
  pending: 'bg-amber-50 border-amber-200',
  backlog: 'bg-secondary-50 border-border',
  ready: 'bg-primary-50 border-primary-200',
  in_progress: 'bg-warning-light border-warning',
  test: 'bg-info-light border-info',
  done: 'bg-success-light border-success',
};

const iconColors: Record<string, string> = {
  pending: 'bg-amber-200 text-amber-800',
  backlog: 'bg-secondary-200 text-text-light',
  ready: 'bg-primary-200 text-primary-700',
  in_progress: 'bg-warning text-text',
  test: 'bg-info text-text-white',
  done: 'bg-success text-text-white',
};

export default function BoardColumn({ id, stories, title, icon: Icon, count }: BoardColumnProps) {
  const { setNodeRef } = useDroppable({
    id,
  });

  return (
    <div
      ref={setNodeRef}
      className={cn(
        'flex-1 min-w-[280px] max-w-[320px] rounded-xl border-2 p-4 transition-colors snap-start',
        columnColors[id] || 'bg-secondary-50 border-border'
      )}
    >
      {/* 列标题 */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center space-x-2">
          <div
            className={cn(
              'w-8 h-8 rounded-lg flex items-center justify-center',
              iconColors[id] || 'bg-secondary-200 text-text-light'
            )}
          >
            <Icon size={18} />
          </div>
          <h3 className="font-semibold text-text">{title}</h3>
        </div>
        <div className="w-8 h-8 rounded-full bg-white border border-border flex items-center justify-center text-sm font-medium text-text">
          {count}
        </div>
      </div>

      {/* 故事卡片列表 */}
      <SortableContext items={stories.map((s) => s.id)} strategy={verticalListSortingStrategy}>
        <div className="space-y-3 min-h-[200px]">
          {stories.length === 0 ? (
            <div className="text-center py-8">
              <div className="mb-2 inline-flex h-12 w-12 items-center justify-center rounded-full bg-white text-text-light">
                <InboxIcon size={22} />
              </div>
              <p className="text-text-light text-sm mb-3">暂无故事</p>
              {id === 'backlog' && (
                <div className="text-xs text-text-light">
                  <p>点击“创建故事”</p>
                  <p>开始添加第一条用户故事</p>
                </div>
              )}
            </div>
          ) : (
            stories.map((story) => <StoryCard key={story.id} story={story} />)
          )}
        </div>
      </SortableContext>
    </div>
  );
}
