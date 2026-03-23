import type { ReactNode } from 'react';
import { cn } from '../../utils/cn';

interface PageContainerProps {
  children: ReactNode;
  className?: string;
}

interface PageHeroProps {
  children: ReactNode;
  className?: string;
}

export function PageContainer({ children, className }: PageContainerProps) {
  return <div className={cn('space-y-3 pb-4', className)}>{children}</div>;
}

export function PageHero({ children, className }: PageHeroProps) {
  return <section className={cn('surface-card overflow-hidden rounded-[2rem]', className)}>{children}</section>;
}
