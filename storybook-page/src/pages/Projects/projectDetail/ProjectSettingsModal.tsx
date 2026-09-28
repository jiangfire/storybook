import { useEffect, useState } from 'react';
import { projectService } from '../../../services/projectService';
import { useToast } from '../../../components/ui/Toast';
import { getErrorMessage } from '../../../utils/error';
import Button from '../../../components/ui/Button';
import Modal from '../../../components/ui/Modal';
import type { Project } from '../../../types/models';

interface ProjectSettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  project: Project;
  /** 保存/归档等操作成功后刷新项目 */
  onSaved: () => void;
  /** 项目删除成功后由父级导航离开 */
  onDeleted: () => void;
}

export function ProjectSettingsModal({
  isOpen,
  onClose,
  project,
  onSaved,
  onDeleted,
}: ProjectSettingsModalProps) {
  const { showSuccess, showError } = useToast();
  const [name, setName] = useState(project.name);
  const [description, setDescription] = useState(project.description || '');
  const [agileMode, setAgileMode] = useState(project.agile_mode);
  const [isSaving, setIsSaving] = useState(false);
  const [isArchiving, setIsArchiving] = useState(false);
  const [isExporting, setIsExporting] = useState(false);

  useEffect(() => {
    if (isOpen) {
      setName(project.name);
      setDescription(project.description || '');
      setAgileMode(project.agile_mode);
    }
  }, [isOpen, project]);

  const handleSave = async () => {
    const trimmed = name.trim();
    if (!trimmed) {
      showError('项目名称不能为空');
      return;
    }
    try {
      setIsSaving(true);
      await projectService.updateProject(project.id, {
        name: trimmed,
        description: description.trim(),
        agile_mode: agileMode,
      });
      showSuccess('项目已更新');
      onSaved();
      onClose();
    } catch (error: unknown) {
      showError(getErrorMessage(error, '项目更新失败'));
    } finally {
      setIsSaving(false);
    }
  };

  const handleToggleArchive = async () => {
    const archiving = !project.archived;
    if (!window.confirm(archiving ? '确认归档该项目吗？归档后可在设置中还原。' : '确认还原该项目吗？')) {
      return;
    }
    try {
      setIsArchiving(true);
      if (archiving) {
        await projectService.archiveProject(project.id);
        showSuccess('项目已归档');
      } else {
        await projectService.unarchiveProject(project.id);
        showSuccess('项目已还原');
      }
      onSaved();
    } catch (error: unknown) {
      showError(getErrorMessage(error, archiving ? '归档失败' : '还原失败'));
    } finally {
      setIsArchiving(false);
    }
  };

  const handleExport = async () => {
    try {
      setIsExporting(true);
      const snapshot = await projectService.exportProject(project.id);
      const blob = new Blob([JSON.stringify(snapshot, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `project-${project.id}-snapshot.json`;
      link.click();
      URL.revokeObjectURL(url);
      showSuccess('项目快照已导出');
    } catch (error: unknown) {
      showError(getErrorMessage(error, '项目导出失败'));
    } finally {
      setIsExporting(false);
    }
  };

  const handleDelete = async () => {
    if (!window.confirm(`确认删除项目「${project.name}」吗？该操作不可恢复，建议先导出快照。`)) {
      return;
    }
    try {
      setIsSaving(true);
      await projectService.deleteProject(project.id);
      showSuccess('项目已删除');
      onDeleted();
    } catch (error: unknown) {
      showError(getErrorMessage(error, '项目删除失败'));
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="项目设置" size="md">
      <div className="space-y-5">
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-text mb-1">项目名称</label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary"
              placeholder="项目名称"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-text mb-1">项目描述</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              className="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary resize-none"
              placeholder="项目描述（可选）"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-text mb-1">敏捷模式</label>
            <select
              value={agileMode}
              onChange={(e) => setAgileMode(e.target.value as Project['agile_mode'])}
              className="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary"
            >
              <option value="kanban">看板模式</option>
              <option value="scrum">Scrum 模式</option>
            </select>
          </div>
        </div>

        <div className="flex justify-end gap-2 border-t border-border pt-4">
          <Button variant="secondary" onClick={onClose} disabled={isSaving}>
            取消
          </Button>
          <Button onClick={() => void handleSave()} isLoading={isSaving}>
            保存设置
          </Button>
        </div>

        <div className="space-y-3 border-t border-border pt-4">
          <div className="text-sm font-medium text-text">数据与生命周期</div>
          <div className="flex flex-col gap-2 sm:flex-row">
            <Button
              variant="secondary"
              onClick={() => void handleExport()}
              isLoading={isExporting}
              className="sm:w-auto"
            >
              导出 JSON 快照
            </Button>
            <Button
              variant="secondary"
              onClick={() => void handleToggleArchive()}
              isLoading={isArchiving}
              className="sm:w-auto"
            >
              {project.archived ? '还原归档' : '归档项目'}
            </Button>
            <Button variant="danger" onClick={() => void handleDelete()} className="sm:w-auto">
              删除项目
            </Button>
          </div>
          <p className="text-xs text-text-light">
            归档不会删除数据；删除不可恢复，操作前建议先导出快照。
          </p>
        </div>
      </div>
    </Modal>
  );
}
