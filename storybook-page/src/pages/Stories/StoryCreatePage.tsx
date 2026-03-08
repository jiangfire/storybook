import { useMemo } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import StoryForm from '../../components/story/StoryForm';

export default function StoryCreatePage() {
  const navigate = useNavigate();
  const { projectId } = useParams<{ projectId: string }>();
  const resolvedProjectID = useMemo(() => Number(projectId), [projectId]);

  if (Number.isNaN(resolvedProjectID) || resolvedProjectID <= 0) {
    return <div className="p-8 text-danger">项目ID无效</div>;
  }

  return (
    <StoryForm
      isOpen
      onClose={() => navigate(`/projects/${resolvedProjectID}/board`, { replace: true })}
      projectId={resolvedProjectID}
      mode="create"
    />
  );
}
