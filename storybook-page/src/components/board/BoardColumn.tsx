import { useDroppable } from '@dnd-kit/core';
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable';
import { StoryBoardItem } from '../../types/models';
import StoryCard from './StoryCard';
import { cn } from '../../utils/cn';

interface BoardColumnProps {
  id: string;
  stories: StoryBoardItem[];
  title: string;
  icon: string;
  count: number;
}

const columnColors: Record<string, string> = {
  backlog: 'bg-gray-50 border-gray-200',
  ready: 'bg-blue-50 border-blue-200',
  in_progress: 'bg-yellow-50 border-yellow-200',
  test: 'bg-purple-50 border-purple-200',
  done: 'bg-green-50 border-green-200',
};

const iconColors: Record<string, string> = {
  backlog: 'bg-gray-200 text-gray-700',
  ready: 'bg-blue-200 text-blue-700',
  in_progress: 'bg-yellow-200 text-yellow-700',
  test: 'bg-purple-200 text-purple-700',
  done: 'bg-green-200 text-green-700',
};

export default function BoardColumn({ id, stories, title, icon, count }: BoardColumnProps) {
  const { setNodeRef } = useDroppable({
    id,
  });

  return (
    <div
      ref={setNodeRef}
      className={cn(
        'flex-1 min-w-[280px] max-w-[320px] rounded-xl border-2 p-4 transition-colors',
        columnColors[id] || 'bg-gray-50 border-gray-200'
      )}
    >
      {/* 列标题 */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center space-x-2">
          <div
            className={cn(
              'w-8 h-8 rounded-lg flex items-center justify-center',
              iconColors[id] || 'bg-gray-200 text-gray-700'
            )}
          >
            <span className="text-lg">{icon}</span>
          </div>
          <h3 className="font-semibold text-text">{title}</h3>
        </div>
        <div className="w-8 h-8 rounded-full bg-white border border-gray-200 flex items-center justify-center text-sm font-medium text-text">
          {count}
        </div>
      </div>

      {/* 故事卡片列表 */}
      <SortableContext items={stories.map((s) => s.id)} strategy={verticalListSortingStrategy}>
        <div className="space-y-3 min-h-[200px]">
          {stories.length === 0 ? (
            <div className="text-center py-8">
              <div className="text-4xl mb-2">📭</div>
              <p className="text-text-light text-sm mb-3">暂无故事</p>
              {id === 'backlog' && (
                <div className="text-xs text-text-light">
                  <p>点击右上角</p>
                  <p className="font-medium text-primary">"+ 创建故事"</p>
                  <p>开始创建你的第一个用户故事</p>
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
