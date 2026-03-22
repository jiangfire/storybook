import { useEffect, type ReactNode, type MouseEvent } from 'react';
import { createPortal } from 'react-dom';
import { cn } from '../../utils/cn';
import { XIcon } from './AppIcon';

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  children: ReactNode;
  size?: 'sm' | 'md' | 'lg' | 'xl';
  showCloseButton?: boolean;
}

export default function Modal({
  isOpen,
  onClose,
  title,
  children,
  size = 'md',
  showCloseButton = true,
}: ModalProps) {
  useEffect(() => {
    if (isOpen) {
      // 禁止背景滚动
      document.body.style.overflow = 'hidden';
    } else {
      // 恢复滚动
      document.body.style.overflow = '';
    }

    return () => {
      document.body.style.overflow = '';
    };
  }, [isOpen]);

  // ESC 键关闭
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };

    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const sizes = {
    sm: 'max-w-md',
    md: 'max-w-lg',
    lg: 'max-w-2xl',
    xl: 'max-w-4xl',
  };

  const handleBackdropClick = (e: MouseEvent<HTMLDivElement>) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  return createPortal(
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/18 p-4 animate-fadeIn"
      onClick={handleBackdropClick}
    >
      <div
        className={cn(
          'surface-card w-full rounded-[1.5rem] border border-border bg-white shadow-[0_28px_60px_-36px_rgba(16,42,67,0.45)] animate-slideUp',
          sizes[size]
        )}
      >
        {/* 头部 */}
        {(title || showCloseButton) && (
          <div className="flex items-center justify-between border-b border-border px-5 py-4">
            {title && <h2 className="text-lg font-semibold text-text">{title}</h2>}
            {showCloseButton && (
              <button
                onClick={onClose}
                aria-label="关闭弹窗"
                className="rounded-xl p-2 transition-colors hover:bg-primary-50"
              >
                <XIcon size={16} />
              </button>
            )}
          </div>
        )}

        {/* 内容 */}
        <div className="max-h-[72vh] overflow-auto px-5 py-5">{children}</div>
      </div>
    </div>,
    document.body
  );
}
