import type { ReactNode } from 'react';
import { render, screen } from '@testing-library/react';
import App from '../App';

vi.mock('../components/ErrorBoundary', () => ({
  default: ({ children }: { children: ReactNode }) => (
    <div data-testid="app-error-boundary">{children}</div>
  ),
}));

vi.mock('../components/ui/Toast', () => ({
  ToastProvider: ({ children }: { children: ReactNode }) => (
    <div data-testid="app-toast-provider">{children}</div>
  ),
}));

vi.mock('../router', () => ({
  default: () => <div>router content</div>,
}));

describe('App', () => {
  it('会用根级错误边界和 Toast Provider 包裹路由', () => {
    render(<App />);

    const boundary = screen.getByTestId('app-error-boundary');
    const toastProvider = screen.getByTestId('app-toast-provider');

    expect(boundary).toContainElement(toastProvider);
    expect(screen.getByText('router content')).toBeInTheDocument();
  });
});
