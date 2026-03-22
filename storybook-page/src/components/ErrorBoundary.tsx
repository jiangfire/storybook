import { Component, type ErrorInfo, type ReactNode } from 'react';

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export default class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('ErrorBoundary caught:', error, errorInfo);
  }

  handleReload = () => {
    window.location.reload();
  };

  render() {
    if (this.state.hasError) {
      return (
        <div className="min-h-screen flex items-center justify-center bg-secondary-50 px-4">
          <div className="max-w-md w-full bg-white rounded-lg shadow-lg p-8 text-center">
            <div className="text-6xl mb-4">⚠️</div>
            <h1 className="text-2xl font-bold text-text mb-2">出错了</h1>
            <p className="text-text-light mb-6">应用遇到了一个错误，请刷新页面重试</p>
            {this.state.error && (
              <div className="bg-secondary-50 rounded p-3 mb-6 text-left">
                <p className="text-xs text-text-light font-mono break-all">
                  {this.state.error.message}
                </p>
              </div>
            )}
            <button onClick={this.handleReload} className="btn btn-primary w-full">
              刷新页面
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
