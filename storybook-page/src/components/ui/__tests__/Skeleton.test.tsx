import { render, screen } from '@testing-library/react';
import { BoardSkeleton, CardSkeleton, ProjectListSkeleton, Skeleton, StoryDetailSkeleton } from '../Skeleton';

describe('Skeleton', () => {
  it('基础骨架屏支持自定义 className', () => {
    const { container } = render(<Skeleton className="h-10 w-10" />);

    expect(screen.getByRole('status', { name: '加载中' })).toBeInTheDocument();
    expect(container.querySelector('.h-10.w-10')).toBeInTheDocument();
  });

  it('CardSkeleton / BoardSkeleton / StoryDetailSkeleton / ProjectListSkeleton 均可渲染', () => {
    const { rerender } = render(<CardSkeleton />);
    expect(screen.getAllByRole('status').length).toBeGreaterThan(0);

    rerender(<BoardSkeleton />);
    expect(screen.getAllByRole('status').length).toBeGreaterThan(10);

    rerender(<StoryDetailSkeleton />);
    expect(screen.getAllByRole('status').length).toBeGreaterThan(5);

    rerender(<ProjectListSkeleton />);
    expect(screen.getAllByRole('status').length).toBeGreaterThan(10);
  });
});
