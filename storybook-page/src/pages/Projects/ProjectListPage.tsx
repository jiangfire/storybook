import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useProjectStore } from '../../stores/projectStore';
import { useToast } from '../../components/ui/Toast';
import ProjectCard from '../../components/ProjectCard';
import Button from '../../components/ui/Button';
import Modal from '../../components/ui/Modal';
import { ProjectListSkeleton } from '../../components/ui/Skeleton';
import { isValidProjectName } from '../../utils/validators';
import { getErrorMessage } from '../../utils/error';

const agileModeOptions: Array<{
  value: 'kanban' | 'scrum';
  label: string;
  emoji: string;
  desc: string;
}> = [
  { value: 'kanban', label: 'Kanban', emoji: '📋', desc: '看板' },
  { value: 'scrum', label: 'Scrum', emoji: '🏃', desc: '冲刺' },
];

export default function ProjectListPage() {
  const navigate = useNavigate();
  const { projects, isLoading, error, fetchProjects, createProject } = useProjectStore();
  const { showSuccess, showError } = useToast();

  const [searchQuery, setSearchQuery] = useState('');
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [newProject, setNewProject] = useState({
    name: '',
    description: '',
    agile_mode: 'kanban' as 'scrum' | 'kanban',
  });
  const [formError, setFormError] = useState('');

  useEffect(() => {
    fetchProjects();
  }, [fetchProjects]);

  const filteredProjects = projects.filter(
    (project) =>
      project.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      project.description?.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const handleCreateProject = async () => {
    // 验证
    if (!isValidProjectName(newProject.name)) {
      setFormError('项目名称长度应在2-100字符之间');
      return;
    }

    try {
      const project = await createProject(newProject);
      showSuccess('项目创建成功');
      setIsCreateModalOpen(false);
      setNewProject({ name: '', description: '', agile_mode: 'kanban' });
      setFormError('');
      navigate(`/projects/${project.id}`);
    } catch (error: unknown) {
      showError(getErrorMessage(error, '创建失败'));
    }
  };

  return (
    <div className="p-8">
      {/* 头部 */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-text mb-2">项目</h1>
          <p className="text-text-light">管理你的敏捷项目</p>
        </div>
        <Button onClick={() => setIsCreateModalOpen(true)}>+ 新建项目</Button>
      </div>

      {/* 搜索框 */}
      <div className="mb-6">
        <input
          type="text"
          placeholder="搜索项目..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="max-w-md w-full px-4 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
        />
      </div>

      {/* 项目列表 */}
      {isLoading ? (
        <ProjectListSkeleton />
      ) : error ? (
        <div className="bg-danger-light text-danger px-4 py-3 rounded-lg">{error}</div>
      ) : filteredProjects.length === 0 ? (
        <div className="text-center py-12">
          <div className="text-6xl mb-4">📁</div>
          <h3 className="text-lg font-medium text-text mb-2">
            {searchQuery ? '没有找到匹配的项目' : '还没有项目'}
          </h3>
          <p className="text-text-light mb-6">
            {searchQuery ? '试试其他关键词' : '创建你的第一个项目，开始敏捷之旅'}
          </p>
          {!searchQuery && <Button onClick={() => setIsCreateModalOpen(true)}>+ 新建项目</Button>}
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredProjects.map((project) => (
            <ProjectCard key={project.id} project={project} />
          ))}
        </div>
      )}

      {/* 创建项目弹窗 */}
      <Modal
        isOpen={isCreateModalOpen}
        onClose={() => {
          setIsCreateModalOpen(false);
          setNewProject({ name: '', description: '', agile_mode: 'kanban' });
          setFormError('');
        }}
        title="新建项目"
        size="md"
      >
        <div className="space-y-4">
          {/* 项目名称 */}
          <div>
            <label className="block text-sm font-medium text-text mb-2">
              项目名称 <span className="text-danger">*</span>
            </label>
            <input
              type="text"
              value={newProject.name}
              onChange={(e) => setNewProject({ ...newProject, name: e.target.value })}
              placeholder="例如：电商平台"
              className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
              maxLength={100}
            />
          </div>

          {/* 项目描述 */}
          <div>
            <label className="block text-sm font-medium text-text mb-2">项目描述</label>
            <textarea
              value={newProject.description}
              onChange={(e) => setNewProject({ ...newProject, description: e.target.value })}
              placeholder="简要描述项目的目标和范围..."
              className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary resize-none"
              rows={3}
              maxLength={500}
            />
          </div>

          {/* 敏捷模式 */}
          <div>
            <label className="block text-sm font-medium text-text mb-2">敏捷模式</label>
            <div className="grid grid-cols-2 gap-3">
              {agileModeOptions.map((mode) => (
                <button
                  key={mode.value}
                  type="button"
                  onClick={() => setNewProject({ ...newProject, agile_mode: mode.value })}
                  className={`p-4 rounded-lg border-2 transition-all ${
                    newProject.agile_mode === mode.value
                      ? 'border-primary bg-primary-50 text-primary'
                      : 'border-border hover:border-primary-300'
                  }`}
                >
                  <div className="text-3xl mb-2">{mode.emoji}</div>
                  <div className="font-medium mb-1">{mode.label}</div>
                  <div className="text-xs text-text-light">{mode.desc}</div>
                </button>
              ))}
            </div>
          </div>

          {/* 错误提示 */}
          {formError && (
            <div className="bg-danger-light text-danger px-4 py-3 rounded-lg text-sm">
              {formError}
            </div>
          )}

          {/* 按钮 */}
          <div className="flex justify-end space-x-3 pt-4">
            <Button
              variant="secondary"
              onClick={() => {
                setIsCreateModalOpen(false);
                setNewProject({ name: '', description: '', agile_mode: 'kanban' });
                setFormError('');
              }}
            >
              取消
            </Button>
            <Button onClick={handleCreateProject}>创建项目</Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
