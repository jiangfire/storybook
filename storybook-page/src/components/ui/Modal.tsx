import { useEffect, ReactNode, MouseEvent } from 'react';
import { createPortal } from 'react-dom';
import { cn } from '../../utils/cn';

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
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm animate-fadeIn"
      onClick={handleBackdropClick}
    >
      <div className={cn('bg-white rounded-2xl shadow-xl w-full animate-slideUp', sizes[size])}>
        {/* 头部 */}
        {(title || showCloseButton) && (
          <div className="flex items-center justify-between p-6 border-b border-border">
            {title && <h2 className="text-xl font-semibold text-text">{title}</h2>}
            {showCloseButton && (
              <button
                onClick={onClose}
                className="p-2 rounded-lg hover:bg-gray-100 transition-colors"
              >
                ✕
              </button>
            )}
          </div>
        )}

        {/* 内容 */}
        <div className="p-6 max-h-[70vh] overflow-auto">{children}</div>
      </div>
    </div>,
    document.body
  );
}
