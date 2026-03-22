import { useParams } from 'react-router-dom';
import { useEffect, useState } from 'react';
import { useProjectStore } from '../../stores/projectStore';
import { useAuthStore } from '../../stores/authStore';
import KanbanBoard from '../../components/board/KanbanBoard';
import StoryForm from '../../components/story/StoryForm';
import Button from '../../components/ui/Button';
import { BoardIcon, SprintIcon } from '../../components/ui/AppIcon';

export default function BoardViewPage() {
  const { id } = useParams<{ id: string }>();
  const { currentProject, fetchProject } = useProjectStore();
  const { user } = useAuthStore();
  const [isStoryFormOpen, setIsStoryFormOpen] = useState(false);
  const projectID = Number(id);
  const project = currentProject?.id === projectID ? currentProject : null;
  const canCreateStory = user?.role === 'product' || user?.role === 'admin';

  useEffect(() => {
    if (!Number.isNaN(projectID) && projectID > 0) {
      fetchProject(projectID);
    }
  }, [projectID, fetchProject]);

  if (!project) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-text-light">加载中...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col">
      {/* 头部 */}
      <div className="border-b border-border bg-white px-4 py-5 sm:px-6 lg:px-8">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <div className="mb-2 flex flex-wrap items-center gap-3">
              <h1 className="text-2xl font-bold text-text">{project.name}</h1>
              <span className="inline-flex items-center gap-1 rounded-full bg-primary-100 px-3 py-1 text-sm font-medium text-primary">
                {project.agile_mode === 'scrum' ? <SprintIcon size={14} /> : <BoardIcon size={14} />}
                {project.agile_mode === 'scrum' ? 'Scrum' : 'Kanban'}
              </span>
            </div>
            {project.description && <p className="text-text-light">{project.description}</p>}
          </div>
          <div className="flex flex-col items-start gap-2 sm:items-end">
            {canCreateStory ? (
              <Button onClick={() => setIsStoryFormOpen(true)}>创建故事</Button>
            ) : (
              <span className="text-xs text-text-light">仅产品经理和管理员可创建故事</span>
            )}
          </div>
        </div>
      </div>

      {/* 看板 */}
      <div className="flex-1 overflow-hidden p-4 sm:p-6 lg:p-8">
        <KanbanBoard projectId={project.id} />
      </div>

      {/* 创建故事弹窗 */}
      <StoryForm
        isOpen={isStoryFormOpen}
        onClose={() => setIsStoryFormOpen(false)}
        projectId={project.id}
        mode="create"
      />
    </div>
  );
}
