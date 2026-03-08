import { useParams } from 'react-router-dom';
import { useEffect, useState } from 'react';
import { useProjectStore } from '../../stores/projectStore';
import KanbanBoard from '../../components/board/KanbanBoard';
import StoryForm from '../../components/story/StoryForm';
import Button from '../../components/ui/Button';

export default function BoardViewPage() {
  const { id } = useParams<{ id: string }>();
  const { currentProject, fetchProject } = useProjectStore();
  const [isStoryFormOpen, setIsStoryFormOpen] = useState(false);
  const projectID = Number(id);
  const project = currentProject?.id === projectID ? currentProject : null;

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
      <div className="px-8 py-6 border-b border-border bg-white">
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center space-x-3 mb-2">
              <h1 className="text-2xl font-bold text-text">{project.name}</h1>
              <span className="px-3 py-1 rounded-full text-sm font-medium bg-primary-100 text-primary">
                {project.agile_mode === 'scrum' ? '🏃 Scrum' : '📋 Kanban'}
              </span>
            </div>
            {project.description && <p className="text-text-light">{project.description}</p>}
          </div>
          <Button onClick={() => setIsStoryFormOpen(true)}>+ 创建故事</Button>
        </div>
      </div>

      {/* 看板 */}
      <div className="flex-1 p-8 overflow-hidden">
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
