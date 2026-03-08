import { useCallback, useEffect, useState } from 'react';
import Modal from '../ui/Modal';
import Button from '../ui/Button';
import { taskService } from '../../services/taskService';
import { getErrorMessage } from '../../utils/error';
import { useToast } from '../ui/Toast';
import { useAuthStore } from '../../stores/authStore';
import type { TaskItem } from '../../types/api';

interface StoryTasksPanelProps {
  storyId: number;
}

const taskStatusOptions: Array<{ value: TaskItem['status']; label: string }> = [
  { value: 'todo', label: '待办' },
  { value: 'in_progress', label: '进行中' },
  { value: 'blocked', label: '阻塞' },
  { value: 'done', label: '已完成' },
];

function resolveAssignedID(task: TaskItem): number | null {
  if (!task.assigned_to) {
    return null;
  }
  if (typeof task.assigned_to === 'number') {
    return task.assigned_to;
  }
  return task.assigned_to.id;
}

export default function StoryTasksPanel({ storyId }: StoryTasksPanelProps) {
  const { user } = useAuthStore();
  const { showError, showSuccess } = useToast();

  const [tasks, setTasks] = useState<TaskItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [creating, setCreating] = useState(false);
  const [splitting, setSplitting] = useState(false);
  const [taskForm, setTaskForm] = useState({
    title: '',
    description: '',
    priority: 2,
    estimated_hours: '',
  });

  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailSaving, setDetailSaving] = useState(false);
  const [selectedTask, setSelectedTask] = useState<TaskItem | null>(null);
  const [codeRef, setCodeRef] = useState('');

  const loadTasks = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const data = await taskService.getStoryTasks(storyId);
      setTasks(data.tasks || []);
    } catch (err: unknown) {
      setTasks([]);
      setError(getErrorMessage(err, '任务列表加载失败'));
    } finally {
      setLoading(false);
    }
  }, [storyId]);

  useEffect(() => {
    void loadTasks();
  }, [loadTasks]);

  const handleCreateTask = async () => {
    const title = taskForm.title.trim();
    if (!title) {
      showError('请输入任务标题');
      return;
    }
    try {
      setCreating(true);
      await taskService.createTask(storyId, {
        title,
        description: taskForm.description.trim() || undefined,
        priority: Number(taskForm.priority),
        estimated_hours: taskForm.estimated_hours ? Number(taskForm.estimated_hours) : undefined,
      });
      setTaskForm({ title: '', description: '', priority: 2, estimated_hours: '' });
      showSuccess('任务创建成功');
      await loadTasks();
    } catch (err: unknown) {
      showError(getErrorMessage(err, '任务创建失败'));
    } finally {
      setCreating(false);
    }
  };

  const handleSplitFromAC = async () => {
    try {
      setSplitting(true);
      const data = await taskService.splitFromAC(storyId);
      showSuccess(`已自动拆分 ${data.created_count || 0} 个子任务`);
      await loadTasks();
    } catch (err: unknown) {
      showError(getErrorMessage(err, 'AC拆分失败'));
    } finally {
      setSplitting(false);
    }
  };

  const updateLocalTask = (next: TaskItem) => {
    setTasks((prev) => prev.map((item) => (item.id === next.id ? { ...item, ...next } : item)));
    setSelectedTask((prev) => (prev && prev.id === next.id ? { ...prev, ...next } : prev));
  };

  const handleStatusChange = async (task: TaskItem, status: TaskItem['status']) => {
    try {
      const updated = await taskService.updateTaskStatus(task.id, { status });
      updateLocalTask(updated);
      showSuccess('任务状态已更新');
    } catch (err: unknown) {
      showError(getErrorMessage(err, '更新任务状态失败'));
    }
  };

  const handleProgressChange = async (task: TaskItem, progress: number) => {
    try {
      const updated = await taskService.updateTaskProgress(task.id, { progress });
      updateLocalTask(updated);
    } catch (err: unknown) {
      showError(getErrorMessage(err, '更新任务进度失败'));
    }
  };

  const handleClaimOrRelease = async (task: TaskItem) => {
    const assignedTo = resolveAssignedID(task);
    try {
      if (!assignedTo) {
        const updated = await taskService.claimTask(task.id);
        updateLocalTask(updated);
        showSuccess('任务领取成功');
        return;
      }
      if (assignedTo === user?.id || user?.role === 'product' || user?.role === 'admin') {
        const updated = await taskService.releaseTask(task.id);
        updateLocalTask(updated);
        showSuccess('任务已释放');
      }
    } catch (err: unknown) {
      showError(getErrorMessage(err, '任务操作失败'));
    }
  };

  const handleDeleteTask = async (task: TaskItem) => {
    if (!confirm(`确认删除任务「${task.title}」吗？`)) {
      return;
    }
    try {
      await taskService.deleteTask(task.id);
      setTasks((prev) => prev.filter((item) => item.id !== task.id));
      if (selectedTask?.id === task.id) {
        setDetailOpen(false);
        setSelectedTask(null);
      }
      showSuccess('任务删除成功');
    } catch (err: unknown) {
      showError(getErrorMessage(err, '任务删除失败'));
    }
  };

  const openTaskDetail = async (taskID: number) => {
    try {
      setDetailOpen(true);
      setDetailLoading(true);
      const detail = await taskService.getTask(taskID);
      setSelectedTask(detail);
      setCodeRef('');
    } catch (err: unknown) {
      showError(getErrorMessage(err, '任务详情加载失败'));
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  };

  const handleSaveTaskDetail = async () => {
    if (!selectedTask) {
      return;
    }
    try {
      setDetailSaving(true);
      const updated = await taskService.updateTask(selectedTask.id, {
        title: selectedTask.title,
        description: selectedTask.description || '',
        priority: selectedTask.priority,
        estimated_hours: selectedTask.estimated_hours,
      });
      updateLocalTask(updated);
      showSuccess('任务详情已保存');
    } catch (err: unknown) {
      showError(getErrorMessage(err, '保存失败'));
    } finally {
      setDetailSaving(false);
    }
  };

  const handleAddCodeRef = async () => {
    if (!selectedTask) {
      return;
    }
    const ref = codeRef.trim();
    if (!ref) {
      showError('请输入代码引用');
      return;
    }
    try {
      const data = await taskService.addCodeRef(selectedTask.id, ref);
      const next: TaskItem = {
        ...selectedTask,
        code_references: data.code_references,
      };
      setSelectedTask(next);
      updateLocalTask(next);
      setCodeRef('');
      showSuccess('代码引用已添加');
    } catch (err: unknown) {
      showError(getErrorMessage(err, '添加代码引用失败'));
    }
  };

  return (
    <div className="bg-white rounded-xl border border-border p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-text">子任务</h2>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="secondary" onClick={() => void loadTasks()} disabled={loading}>
            刷新
          </Button>
          <Button size="sm" onClick={handleSplitFromAC} isLoading={splitting}>
            AC自动拆分
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-2">
        <input
          value={taskForm.title}
          onChange={(e) => setTaskForm((prev) => ({ ...prev, title: e.target.value }))}
          placeholder="任务标题"
          className="md:col-span-2 px-3 py-2 border border-border rounded-lg"
        />
        <select
          value={taskForm.priority}
          onChange={(e) => setTaskForm((prev) => ({ ...prev, priority: Number(e.target.value) }))}
          className="px-3 py-2 border border-border rounded-lg"
        >
          <option value={0}>优先级0</option>
          <option value={1}>优先级1</option>
          <option value={2}>优先级2</option>
          <option value={3}>优先级3</option>
          <option value={4}>优先级4</option>
        </select>
        <input
          type="number"
          min={0}
          max={500}
          value={taskForm.estimated_hours}
          onChange={(e) => setTaskForm((prev) => ({ ...prev, estimated_hours: e.target.value }))}
          placeholder="预估工时"
          className="px-3 py-2 border border-border rounded-lg"
        />
      </div>
      <textarea
        value={taskForm.description}
        onChange={(e) => setTaskForm((prev) => ({ ...prev, description: e.target.value }))}
        placeholder="任务描述（可选）"
        rows={2}
        className="w-full px-3 py-2 border border-border rounded-lg resize-none"
      />
      <div className="flex justify-end">
        <Button size="sm" onClick={handleCreateTask} isLoading={creating}>
          新建任务
        </Button>
      </div>

      {error && <div className="text-sm text-danger">{error}</div>}
      {loading ? (
        <div className="text-sm text-text-light">任务加载中...</div>
      ) : tasks.length === 0 ? (
        <div className="text-sm text-text-light">暂无任务</div>
      ) : (
        <div className="space-y-2">
          {tasks.map((task) => {
            const assignedTo = resolveAssignedID(task);
            const canRelease =
              assignedTo === user?.id || user?.role === 'product' || user?.role === 'admin';
            return (
              <div
                key={task.id}
                className="border border-border rounded-lg p-3 flex flex-col md:flex-row md:items-center gap-3"
              >
                <div className="flex-1 min-w-0">
                  <div className="font-medium text-text truncate">{task.title}</div>
                  <div className="text-xs text-text-light">
                    P{task.priority} · 工时 {task.estimated_hours || 0}h · 进度 {task.progress || 0}%
                  </div>
                </div>
                <select
                  value={task.status}
                  onChange={(e) => void handleStatusChange(task, e.target.value as TaskItem['status'])}
                  className="px-2 py-1 border border-border rounded text-sm"
                >
                  {taskStatusOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
                <input
                  type="number"
                  min={0}
                  max={100}
                  value={task.progress || 0}
                  onChange={(e) => void handleProgressChange(task, Number(e.target.value))}
                  className="w-20 px-2 py-1 border border-border rounded text-sm"
                />
                <Button size="sm" variant="secondary" onClick={() => void openTaskDetail(task.id)}>
                  详情
                </Button>
                <Button
                  size="sm"
                  variant={assignedTo ? 'secondary' : 'primary'}
                  disabled={assignedTo !== null && !canRelease}
                  onClick={() => void handleClaimOrRelease(task)}
                >
                  {assignedTo ? '释放' : '领取'}
                </Button>
                <Button size="sm" variant="danger" onClick={() => void handleDeleteTask(task)}>
                  删除
                </Button>
              </div>
            );
          })}
        </div>
      )}

      <Modal
        isOpen={detailOpen}
        onClose={() => {
          if (!detailSaving) {
            setDetailOpen(false);
            setSelectedTask(null);
          }
        }}
        title="任务详情"
        size="md"
      >
        {detailLoading || !selectedTask ? (
          <div className="text-sm text-text-light">加载任务详情中...</div>
        ) : (
          <div className="space-y-3">
            <input
              value={selectedTask.title}
              onChange={(e) => setSelectedTask((prev) => (prev ? { ...prev, title: e.target.value } : prev))}
              className="w-full px-3 py-2 border border-border rounded-lg"
              placeholder="任务标题"
            />
            <textarea
              value={selectedTask.description || ''}
              onChange={(e) =>
                setSelectedTask((prev) => (prev ? { ...prev, description: e.target.value } : prev))
              }
              rows={3}
              className="w-full px-3 py-2 border border-border rounded-lg resize-none"
              placeholder="任务描述"
            />
            <div className="grid grid-cols-2 gap-2">
              <input
                type="number"
                min={0}
                max={4}
                value={selectedTask.priority}
                onChange={(e) =>
                  setSelectedTask((prev) => (prev ? { ...prev, priority: Number(e.target.value) } : prev))
                }
                className="px-3 py-2 border border-border rounded-lg"
                placeholder="优先级"
              />
              <input
                type="number"
                min={0}
                max={500}
                value={selectedTask.estimated_hours || 0}
                onChange={(e) =>
                  setSelectedTask((prev) =>
                    prev ? { ...prev, estimated_hours: Number(e.target.value) } : prev
                  )
                }
                className="px-3 py-2 border border-border rounded-lg"
                placeholder="预估工时"
              />
            </div>
            <div className="space-y-2">
              <div className="text-sm font-medium text-text">代码引用</div>
              <div className="text-xs text-text-light">
                {selectedTask.code_references && selectedTask.code_references.length > 0
                  ? selectedTask.code_references.join(' | ')
                  : '暂无代码引用'}
              </div>
              <div className="flex gap-2">
                <input
                  value={codeRef}
                  onChange={(e) => setCodeRef(e.target.value)}
                  className="flex-1 px-3 py-2 border border-border rounded-lg"
                  placeholder="例如 src/components/Task.tsx:42"
                />
                <Button size="sm" onClick={handleAddCodeRef}>
                  添加
                </Button>
              </div>
            </div>
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="secondary" onClick={() => setDetailOpen(false)} disabled={detailSaving}>
                关闭
              </Button>
              <Button onClick={handleSaveTaskDetail} isLoading={detailSaving}>
                保存
              </Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
