import Button from '../../../components/ui/Button';
import Modal from '../../../components/ui/Modal';
import type { ConfirmActionState } from './types';

interface ConfirmActionModalProps {
  confirmAction: ConfirmActionState | null;
  isSubmitting: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}

export function ConfirmActionModal({
  confirmAction,
  isSubmitting,
  onCancel,
  onConfirm,
}: ConfirmActionModalProps) {
  return (
    <Modal isOpen={Boolean(confirmAction)} onClose={onCancel} title={confirmAction?.title} size="sm">
      <div className="space-y-4">
        <p className="text-sm text-text">{confirmAction?.message}</p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={onCancel} disabled={isSubmitting}>
            取消
          </Button>
          <Button variant="danger" onClick={onConfirm} isLoading={isSubmitting}>
            确认移除
          </Button>
        </div>
      </div>
    </Modal>
  );
}
