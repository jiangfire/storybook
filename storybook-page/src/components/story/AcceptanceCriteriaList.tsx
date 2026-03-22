import { useState } from 'react';
import type { AcceptanceCriteria, ACStatus } from '../../types/models';
import { useStoryStore } from '../../stores/storyStore';
import { cn } from '../../utils/cn';
import { CheckCircleIcon, ClipboardIcon, XIcon } from '../ui/AppIcon';
import Button from '../ui/Button';

interface AcceptanceCriteriaListProps {
  storyId: number;
  criteria: AcceptanceCriteria[];
  editable?: boolean;
}

const statusConfig: Record<
  ACStatus,
  {
    label: string;
    color: string;
    bgColor: string;
    icon: typeof ClipboardIcon;
  }
> = {
  pending: {
    label: '待验收',
    color: 'text-text',
    bgColor: 'bg-secondary-100',
    icon: ClipboardIcon,
  },
  passed: {
    label: '已通过',
    color: 'text-green-700',
    bgColor: 'bg-green-100',
    icon: CheckCircleIcon,
  },
  failed: {
    label: '未通过',
    color: 'text-red-700',
    bgColor: 'bg-red-100',
    icon: XIcon,
  },
};

export default function AcceptanceCriteriaList({
  storyId,
  criteria,
  editable = true,
}: AcceptanceCriteriaListProps) {
  const { updateACStatus, isUpdating } = useStoryStore();
  const [editingAC, setEditingAC] = useState<string | null>(null);
  const [evidence, setEvidence] = useState('');

  const handleStatusChange = async (acId: string, newStatus: ACStatus) => {
    try {
      await updateACStatus(storyId, acId, newStatus, '');
    } catch (error) {
      console.error('Failed to update AC status:', error);
    }
  };

  const handleSaveEvidence = async (acId: string) => {
    try {
      await updateACStatus(storyId, acId, 'passed', evidence);
      setEditingAC(null);
      setEvidence('');
    } catch (error) {
      console.error('Failed to save evidence:', error);
    }
  };

  return (
    <div className="space-y-3">
      {criteria.map((ac) => {
        const config = statusConfig[ac.status];
        const isEditing = editingAC === ac.id;
        const StatusIcon = config.icon;

        return (
          <div
            key={ac.id}
            className={cn(
              'p-4 rounded-lg border transition-all',
              ac.status === 'passed' && 'border-green-200 bg-green-50',
              ac.status === 'failed' && 'border-red-200 bg-red-50',
              ac.status === 'pending' && 'border-border'
            )}
          >
            <div className="flex items-start space-x-3">
              {/* 状态指示器 */}
              {editable ? (
                <div className="flex-shrink-0 pt-1 flex flex-col gap-1">
                  {(['pending', 'passed', 'failed'] as ACStatus[]).map((status) => (
                    <button
                      key={status}
                      onClick={() => handleStatusChange(ac.id, status)}
                      disabled={isUpdating}
                      className={cn(
                        'px-2 py-0.5 rounded text-xs border transition-colors',
                        status === 'pending' &&
                          ac.status === 'pending' &&
                          'bg-secondary-200 border-secondary-300 text-text',
                        status === 'passed' &&
                          ac.status === 'passed' &&
                          'bg-green-200 border-green-300 text-green-800',
                        status === 'failed' &&
                          ac.status === 'failed' &&
                          'bg-red-200 border-red-300 text-red-800',
                        ac.status !== status &&
                          'bg-white border-border text-text-light hover:border-primary'
                      )}
                    >
                      {status === 'pending' ? '待' : status === 'passed' ? '过' : '未'}
                    </button>
                  ))}
                </div>
              ) : (
                <div
                  className={cn(
                    'w-6 h-6 rounded-full flex items-center justify-center',
                    config.bgColor
                  )}
                >
                  <StatusIcon size={14} className={cn('font-medium', config.color)} />
                </div>
              )}

              {/* 内容 */}
              <div className="flex-1 min-w-0">
                {/* 描述 */}
                <p className="text-text mb-2">{ac.description}</p>

                {/* AC 引用 */}
                {ac.ref && (
                  <div className="text-xs text-text-light bg-white px-2 py-1 rounded inline-block">
                    {ac.ref}
                  </div>
                )}

                {/* 证据 */}
                {ac.evidence && (
                  <div className="mt-2 p-2 bg-white rounded border border-border text-sm">
                    <div className="text-xs text-text-light mb-1">证据:</div>
                    <code className="text-xs text-primary">{ac.evidence}</code>
                  </div>
                )}

                {/* 编辑证据输入框 */}
                {isEditing && (
                  <div className="mt-3 space-y-2">
                    <textarea
                      value={evidence}
                      onChange={(e) => setEvidence(e.target.value)}
                      placeholder="添加证据（如代码路径、测试截图等）..."
                      className="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary resize-none"
                      rows={2}
                    />
                    <div className="flex space-x-2">
                      <Button
                        size="sm"
                        onClick={() => handleSaveEvidence(ac.id)}
                      >
                        保存
                      </Button>
                      <Button
                        size="sm"
                        variant="secondary"
                        onClick={() => {
                          setEditingAC(null);
                          setEvidence('');
                        }}
                      >
                        取消
                      </Button>
                    </div>
                  </div>
                )}

                {/* 添加证据按钮 */}
                {editable && ac.status === 'passed' && !ac.evidence && !isEditing && (
                  <button
                    onClick={() => setEditingAC(ac.id)}
                    className="mt-2 text-sm text-primary transition-colors hover:text-primary-700"
                  >
                    添加证据
                  </button>
                )}
              </div>
            </div>
          </div>
        );
      })}

      {/* 统计信息 */}
      <div className="pt-3 border-t border-border">
        <div className="flex items-center justify-between text-sm">
          <span className="text-text-light">完成进度</span>
          <span className="font-medium">
            {criteria.filter((ac) => ac.status === 'passed').length} / {criteria.length}
          </span>
        </div>
        <div className="w-full bg-secondary-200 rounded-full h-2 mt-2">
          <div
            className="bg-success h-2 rounded-full transition-all duration-300"
            style={{
              width: `${criteria.length > 0 ? (criteria.filter((ac) => ac.status === 'passed').length / criteria.length) * 100 : 0}%`,
            }}
          />
        </div>
      </div>
    </div>
  );
}
