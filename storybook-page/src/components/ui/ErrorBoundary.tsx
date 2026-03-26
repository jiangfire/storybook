import React, { Component, ErrorInfo, ReactNode } from 'react';

interface ErrorBoundaryProps {
	children: ReactNode;
	fallback?: ReactNode;
	onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface ErrorBoundaryState {
	hasError: boolean;
	error: Error | null;
}

/**
 * Error Boundary组件
 * 捕获子组件树中的JavaScript错误，记录错误日志，并显示fallback UI
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
	constructor(props: ErrorBoundaryProps) {
		super(props);
		this.state = { hasError: false, error: null };
	}

	static getDerivedStateFromError(error: Error): ErrorBoundaryState {
		// 更新state使下一次渲染能够显示降级后的UI
		return { hasError: true, error };
	}

	componentDidCatch(error: Error, errorInfo: ErrorInfo) {
		// 记录错误到日志系统
		console.error('ErrorBoundary caught an error:', error, errorInfo);

		// 调用自定义错误处理函数
		if (this.props.onError) {
			this.props.onError(error, errorInfo);
		}
	}

	render() {
		if (this.state.hasError) {
			// 自定义fallback UI
			if (this.props.fallback) {
				return this.props.fallback;
			}

			// 默认fallback UI
			return (
				<div className="min-h-[400px] flex items-center justify-center">
					<div className="text-center">
						<div className="text-6xl mb-4">⚠️</div>
						<h2 className="text-xl font-semibold text-text mb-2">
							出错了
						</h2>
						<p className="text-text-light mb-4">
							页面遇到了一些问题，请刷新页面重试
						</p>
						<button
							onClick={() => window.location.reload()}
							className="inline-flex items-center justify-center rounded-xl border border-transparent bg-primary px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-primary-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2"
						>
							刷新页面
						</button>
						{process.env.NODE_ENV === 'development' && this.state.error && (
							<details className="mt-4 text-left">
								<summary className="cursor-pointer text-sm text-text-light hover:text-text">
									错误详情
								</summary>
								<pre className="mt-2 bg-gray-100 p-4 rounded-lg text-xs overflow-auto max-w-2xl mx-auto">
									{this.state.error.toString()}
									{this.state.error.stack}
								</pre>
							</details>
						)}
					</div>
				</div>
			);
		}

		return this.props.children;
	}
}

/**
 * Async Error Boundary
 * 专门用于捕获异步操作中的错误
 */
export class AsyncErrorBoundary extends Component<
	Omit<ErrorBoundaryProps, 'fallback'>,
	ErrorBoundaryState
> {
	constructor(props: Omit<ErrorBoundaryProps, 'fallback'>) {
		super(props);
		this.state = { hasError: false, error: null };
	}

	static getDerivedStateFromError(error: Error): ErrorBoundaryState {
		return { hasError: true, error };
	}

	componentDidCatch(error: Error, errorInfo: ErrorInfo) {
		console.error('AsyncErrorBoundary caught an error:', error, errorInfo);
		if (this.props.onError) {
			this.props.onError(error, errorInfo);
		}
	}

	render() {
		if (this.state.hasError) {
			return (
				<div className="min-h-[200px] flex items-center justify-center bg-error/10">
					<div className="text-center">
						<div className="text-4xl mb-2">⚠️</div>
						<p className="text-text text-sm">
							异步操作出错，请重试
						</p>
					</div>
				</div>
			);
		}

		return this.props.children;
	}
}

/**
 * Hook版本的错误边界（函数组件）
 * 注意：React目前没有官方的Hook版本Error Boundary
 * 这是使用包装器的模拟实现
 */
export function withErrorBoundary<P extends object>(
	Component: React.ComponentType<P>,
	errorBoundaryProps?: Omit<ErrorBoundaryProps, 'children'>
): React.ComponentType<P> {
	return function WithErrorBoundaryWrapper(props: P) {
		return (
			<ErrorBoundary {...errorBoundaryProps}>
				<Component {...props} />
			</ErrorBoundary>
		);
	};
}
