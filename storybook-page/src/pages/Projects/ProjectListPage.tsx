import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useProjectStore } from '../../stores/projectStore';
import { useToast } from '../../components/ui/Toast';
import { PageContainer, PageHero } from '../../components/page/PageLayout';
import ProjectCard from '../../components/ProjectCard';
import Button from '../../components/ui/Button';
import Modal from '../../components/ui/Modal';
import { ProjectListSkeleton } from '../../components/ui/Skeleton';
import { isValidProjectName } from '../../utils/validators';
import { getErrorMessage } from '../../utils/error';
import {
  BoardIcon,
  CrownIcon,
  FolderIcon,
  SearchIcon,
  SprintIcon,
} from '../../components/ui/AppIcon';

const agileModeOptions: Array<{
  value: 'kanban' | 'scrum';
  label: string;
  desc: string;
}> = [
  { value: 'kanban', label: '看板模式', desc: '持续流转' },
  { value: 'scrum', label: '冲刺模式', desc: '按周期推进' },
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

  const ownerProjects = projects.filter((project) => project.is_owner).length;
  const scrumProjects = projects.filter((project) => project.agile_mode === 'scrum').length;
  const kanbanProjects = projects.filter((project) => project.agile_mode !== 'scrum').length;

  const filteredProjects = projects.filter(
    (project) =>
      project.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      project.description?.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const resetCreateProjectForm = () => {
    setNewProject({ name: '', description: '', agile_mode: 'kanban' });
    setFormError('');
  };

  const closeCreateProjectModal = () => {
    setIsCreateModalOpen(false);
    resetCreateProjectForm();
  };

  const handleCreateProject = async () => {
    // 验证
    if (!isValidProjectName(newProject.name)) {
      setFormError('项目名称长度应在2-100字符之间');
      return;
    }

    try {
      const project = await createProject(newProject);
      showSuccess('项目创建成功');
      closeCreateProjectModal();
      navigate(`/projects/${project.id}`);
    } catch (error: unknown) {
      showError(getErrorMessage(error, '创建失败'));
    }
  };

  return (
    <PageContainer>
      <PageHero className="border-primary-100 bg-gradient-to-br from-white via-secondary-50 to-primary-50/60 shadow-[0_20px_44px_-38px_rgba(16,42,67,0.28)]">
        <div className="grid gap-4 px-5 py-5 sm:px-6 lg:grid-cols-[minmax(0,1fr)_320px] lg:px-7 lg:py-6">
          <div className="space-y-4">
            <div className="space-y-3">
              <div className="inline-flex items-center gap-2 rounded-full bg-primary-50 px-3 py-1 text-xs font-medium text-primary">
                <FolderIcon size={14} />
                项目空间
              </div>
              <div>
                <h1 className="text-3xl font-bold tracking-tight text-text sm:text-4xl">项目</h1>
                <p className="mt-2 max-w-2xl text-sm leading-6 text-text-light sm:text-base">
                  在这里管理团队的交付空间。先找到项目，再进入详情、看板或冲刺。
                </p>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
              <div className="hero-subcard rounded-2xl p-4">
                <div className="text-xs font-medium text-text-light">全部项目</div>
                <div className="mt-2 text-3xl font-semibold text-text">{projects.length}</div>
              </div>
              <div className="hero-subcard rounded-2xl p-4">
                <div className="text-xs font-medium text-text-light">负责项目</div>
                <div className="mt-2 inline-flex items-center gap-2 text-3xl font-semibold text-text">
                  <CrownIcon size={20} />
                  {ownerProjects}
                </div>
              </div>
              <div className="hero-subcard rounded-2xl p-4">
                <div className="text-xs font-medium text-text-light">敏捷模式</div>
                <div className="mt-2 flex items-center gap-3 text-sm text-text">
                  <span className="inline-flex items-center gap-1 rounded-full bg-primary-50 px-2.5 py-1">
                    <SprintIcon size={13} />
                    冲刺 {scrumProjects}
                  </span>
                  <span className="inline-flex items-center gap-1 rounded-full bg-accent-50 px-2.5 py-1">
                    <BoardIcon size={13} />
                    看板 {kanbanProjects}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <div className="rounded-[1.6rem] border border-primary-700 bg-primary-800 p-5 text-white">
            <div className="text-xs font-medium text-white/70">快速操作</div>
            <h2 className="mt-3 text-2xl font-semibold">创建新项目</h2>
            <p className="mt-2 text-sm leading-6 text-white/80">
              创建后会直接进入项目页，再继续补充成员、故事和工作方式。
            </p>
            <Button className="mt-5 bg-white text-primary hover:bg-white/90" onClick={() => setIsCreateModalOpen(true)}>
              + 新建项目
            </Button>
          </div>
        </div>
      </PageHero>

      <section className="section-card rounded-[1.7rem] px-4 py-4 sm:px-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h2 className="text-lg font-semibold text-text">项目列表</h2>
            <p className="mt-1 text-sm text-text-light">
              {searchQuery ? `当前筛出 ${filteredProjects.length} 个结果` : '按名称或描述快速定位项目'}
            </p>
          </div>
          <div className="w-full lg:max-w-md">
            <label htmlFor="project-search" className="mb-2 block text-xs font-medium text-text-light">
              搜索
            </label>
            <div className="relative">
              <input
                id="project-search"
                type="text"
                placeholder="搜索项目..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full rounded-2xl border border-border bg-white px-4 py-3 pl-11 text-sm outline-none transition focus:border-primary-200 focus:ring-4 focus:ring-primary/10"
              />
              <span className="absolute left-4 top-1/2 -translate-y-1/2 text-text-light">
                <SearchIcon size={16} />
              </span>
            </div>
          </div>
        </div>
      </section>

      {isLoading ? (
        <ProjectListSkeleton />
      ) : error ? (
        <div className="state-panel state-panel-error">{error}</div>
      ) : filteredProjects.length === 0 ? (
        <div className="section-card rounded-[1.8rem] px-4 py-12 text-center">
          <div className="mb-4 inline-flex h-16 w-16 items-center justify-center rounded-full bg-secondary-50 text-text-light">
            <FolderIcon size={28} />
          </div>
          <h3 className="mb-2 text-lg font-medium text-text">
            {searchQuery ? '没有找到匹配的项目' : '还没有项目'}
          </h3>
          <p className="mb-6 text-text-light">
            {searchQuery ? '试试其他关键词' : '创建你的第一个项目，开始敏捷之旅'}
          </p>
          {!searchQuery && <Button onClick={() => setIsCreateModalOpen(true)}>+ 新建项目</Button>}
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          {filteredProjects.map((project) => (
            <ProjectCard key={project.id} project={project} />
          ))}
        </div>
      )}

      {/* 创建项目弹窗 */}
      <Modal
        isOpen={isCreateModalOpen}
        onClose={closeCreateProjectModal}
        title="新建项目"
        size="md"
      >
        <div className="space-y-4">
          {/* 项目名称 */}
          <div>
            <label htmlFor="new-project-name" className="block text-sm font-medium text-text mb-2">
              项目名称 <span className="text-danger">*</span>
            </label>
            <input
              id="new-project-name"
              type="text"
              value={newProject.name}
              onChange={(e) => setNewProject({ ...newProject, name: e.target.value })}
              placeholder="例如：电商平台"
              className="field-control"
              maxLength={100}
            />
          </div>

          {/* 项目描述 */}
          <div>
            <label htmlFor="new-project-description" className="block text-sm font-medium text-text mb-2">
              项目描述
            </label>
            <textarea
              id="new-project-description"
              value={newProject.description}
              onChange={(e) => setNewProject({ ...newProject, description: e.target.value })}
              placeholder="简要描述项目的目标和范围..."
              className="field-control"
              rows={3}
              maxLength={500}
            />
          </div>

          {/* 敏捷模式 */}
          <div>
            <label className="block text-sm font-medium text-text mb-2">敏捷模式</label>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              {agileModeOptions.map((mode) => (
                <button
                  key={mode.value}
                  type="button"
                  onClick={() => setNewProject({ ...newProject, agile_mode: mode.value })}
                  className={`rounded-2xl border p-4 transition-colors ${
                    newProject.agile_mode === mode.value
                      ? 'border-primary-200 bg-primary-50 text-primary'
                      : 'border-border bg-white hover:border-primary-200 hover:bg-primary-50'
                  }`}
                >
                  <div className="mb-2 flex justify-center">
                    {mode.value === 'scrum' ? (
                      <SprintIcon size={24} />
                    ) : (
                      <BoardIcon size={24} />
                    )}
                  </div>
                  <div className="font-medium mb-1">{mode.label}</div>
                  <div className="text-xs text-text-light">{mode.desc}</div>
                </button>
              ))}
            </div>
          </div>

          {/* 错误提示 */}
          {formError && (
            <div className="state-panel state-panel-error">{formError}</div>
          )}

          {/* 按钮 */}
          <div className="flex flex-col-reverse gap-3 pt-4 sm:flex-row sm:justify-end">
            <Button
              variant="secondary"
              onClick={closeCreateProjectModal}
            >
              取消
            </Button>
            <Button onClick={handleCreateProject}>创建项目</Button>
          </div>
        </div>
      </Modal>
    </PageContainer>
  );
}
