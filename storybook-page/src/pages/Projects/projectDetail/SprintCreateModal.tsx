import Button from '../../../components/ui/Button';
import Modal from '../../../components/ui/Modal';
import type { CreateSprintRequest } from '../../../types/api';

interface SprintCreateModalProps {
  isOpen: boolean;
  sprintForm: CreateSprintRequest;
  sprintFormError: string;
  isSubmitting: boolean;
  onClose: () => void;
  onChange: (field: keyof CreateSprintRequest, value: string) => void;
  onSubmit: () => void;
}

export function SprintCreateModal({
  isOpen,
  sprintForm,
  sprintFormError,
  isSubmitting,
  onClose,
  onChange,
  onSubmit,
}: SprintCreateModalProps) {
  return (
    <Modal isOpen={isOpen} onClose={onClose} title="新建冲刺" size="md">
      <div className="space-y-4">
        <div>
          <label className="mb-2 block text-sm font-medium text-text">冲刺名称</label>
          <input
            type="text"
            value={sprintForm.name}
            onChange={(event) => onChange('name', event.target.value)}
            className="field-control"
            placeholder="例如：Sprint 1"
            maxLength={120}
          />
        </div>
        <div>
          <label className="mb-2 block text-sm font-medium text-text">目标（可选）</label>
          <textarea
            value={sprintForm.goal}
            onChange={(event) => onChange('goal', event.target.value)}
            className="field-control"
            rows={3}
            maxLength={500}
            placeholder="本次冲刺要达成什么"
          />
        </div>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <label className="mb-2 block text-sm font-medium text-text">开始日期</label>
            <input
              type="date"
              value={sprintForm.start_date}
              onChange={(event) => onChange('start_date', event.target.value)}
              className="field-control"
            />
          </div>
          <div>
            <label className="mb-2 block text-sm font-medium text-text">结束日期</label>
            <input
              type="date"
              value={sprintForm.end_date}
              onChange={(event) => onChange('end_date', event.target.value)}
              className="field-control"
            />
          </div>
        </div>
        {sprintFormError && <div className="state-panel state-panel-error">{sprintFormError}</div>}
        <div className="flex flex-col-reverse gap-3 pt-2 sm:flex-row sm:justify-end">
          <Button variant="secondary" onClick={onClose} disabled={isSubmitting}>
            取消
          </Button>
          <Button onClick={onSubmit} isLoading={isSubmitting}>
            创建冲刺
          </Button>
        </div>
      </div>
    </Modal>
  );
}
