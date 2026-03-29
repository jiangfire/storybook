import { render, screen } from '@testing-library/react';
import LoadingSpinner from '../LoadingSpinner';

describe('LoadingSpinner', () => {
  it('显示统一加载文案和动画容器', () => {
    const { container } = render(<LoadingSpinner />);

    expect(screen.getByText('加载中...')).toBeInTheDocument();
    expect(container.querySelector('.animate-spin')).toBeInTheDocument();
  });
});
