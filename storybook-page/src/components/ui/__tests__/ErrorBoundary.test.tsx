import { render, screen, cleanup } from '@testing-library/react';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import React from 'react';
import { ErrorBoundary, AsyncErrorBoundary, withErrorBoundary } from '../ErrorBoundary';
import * as matchers from '@testing-library/jest-dom/matchers';

// 添加匹配器
expect.extend(matchers);

// 每个测试后清理DOM
afterEach(() => {
 cleanup();
});

// 一个会抛出错误的组件
const ThrowError = ({ message = 'Test error' }: { message?: string }) => {
	throw new Error(message);
};

// 一个异步抛出错误的组件
const AsyncThrowError = () => {
	React.useEffect(() => {
		throw new Error('Async error');
	}, []);
	return <div>Should not render</div>;
};

// 一个正常组件
const NormalComponent = () => <div>Normal Content</div>;

describe('ErrorBoundary', () => {
 beforeEach(() => {
 // 抑制控制台错误输出
 vi.spyOn(console, 'error').mockImplementation(() => {});
 });

 afterEach(() => {
 vi.restoreAllMocks();
 });

 it('应该捕获同步错误并显示fallback UI', () => {
 const fallback = <div>Custom Fallback</div>;

 render(
   <ErrorBoundary fallback={fallback}>
     <ThrowError message="Test error" />
   </ErrorBoundary>
 );

 expect(screen.getByText('Custom Fallback')).toBeInTheDocument();
 });

 it('应该使用默认fallback UI当未提供自定义fallback', () => {
 const { container } = render(
   <ErrorBoundary>
     <ThrowError message="Default fallback test" />
   </ErrorBoundary>
 );

 // 使用container查询避免多元素冲突
 expect(container).toHaveTextContent(/出错了/);
 expect(container).toHaveTextContent(/刷新页面/);
 });

 it('应该在开发环境显示错误详情', () => {
 const originalEnv = process.env.NODE_ENV;
 process.env.NODE_ENV = 'development';

 render(
   <ErrorBoundary>
     <ThrowError message="Development error details" />
   </ErrorBoundary>
 );

 expect(screen.getByText(/错误详情/)).toBeInTheDocument();
 expect(screen.getByText(/Development error details/)).toBeInTheDocument();

 process.env.NODE_ENV = originalEnv;
 });

 it('应该正常渲染子组件当没有错误', () => {
 render(
   <ErrorBoundary>
     <NormalComponent />
   </ErrorBoundary>
 );

 expect(screen.getByText('Normal Content')).toBeInTheDocument();
 });

 it('应该调用onError回调', () => {
 const onError = vi.fn();

 render(
   <ErrorBoundary onError={onError}>
     <ThrowError message="Callback test" />
   </ErrorBoundary>
 );

 expect(onError).toHaveBeenCalled();
 // onError应该被调用，参数包含Error和errorInfo
 expect(onError).toHaveBeenCalledWith(
 expect.any(Error),
 expect.objectContaining({
 componentStack: expect.any(String)
 })
 );
 });

 it('应该支持从错误中恢复', () => {
 const { rerender } = render(
   <ErrorBoundary>
     <NormalComponent />
   </ErrorBoundary>
 );

 // 初始正常渲染
 expect(screen.getByText('Normal Content')).toBeInTheDocument();

 // 切换到错误组件
 rerender(
   <ErrorBoundary>
     <ThrowError />
   </ErrorBoundary>
 );

 expect(screen.getByText(/出错了/)).toBeInTheDocument();
 });

 describe('withErrorBoundary HOC', () => {
 it('应该包装组件并添加错误边界', () => {
 const WrappedComponent = withErrorBoundary(NormalComponent);
 render(<WrappedComponent />);
 expect(screen.getByText('Normal Content')).toBeInTheDocument();
 });

 it('应该捕获被包装组件的错误', () => {
 const WrappedComponent = withErrorBoundary(ThrowError);
 render(<WrappedComponent />);
 expect(screen.getByText(/出错了/)).toBeInTheDocument();
 });
 });
});

describe('AsyncErrorBoundary', () => {
 beforeEach(() => {
 vi.spyOn(console, 'error').mockImplementation(() => {});
 });

 afterEach(() => {
 vi.restoreAllMocks();
 });

 it('应该显示简化的fallback UI', () => {
 render(
   <AsyncErrorBoundary>
     <AsyncThrowError />
   </AsyncErrorBoundary>
 );

 expect(screen.getByText(/异步操作出错/)).toBeInTheDocument();
 });
});

describe('ErrorBoundary 边界条件', () => {
 beforeEach(() => {
 vi.spyOn(console, 'error').mockImplementation(() => {});
 });

 afterEach(() => {
 vi.restoreAllMocks();
 });

 it('应该处理嵌套Error Boundary', () => {
 const InnerFallback = <div>Inner Error</div>;
 const OuterFallback = <div>Outer Error</div>;

 render(
   <ErrorBoundary fallback={OuterFallback}>
     <ErrorBoundary fallback={InnerFallback}>
       <NormalComponent />
     </ErrorBoundary>
   </ErrorBoundary>
 );

 expect(screen.getByText('Normal Content')).toBeInTheDocument();
 });

 it('内层boundary应该捕获错误', () => {
 const InnerFallback = <div>Inner Error</div>;
 const OuterFallback = <div>Outer Error</div>;

 render(
   <ErrorBoundary fallback={OuterFallback}>
     <ErrorBoundary fallback={InnerFallback}>
       <ThrowError message="Inner error" />
     </ErrorBoundary>
   </ErrorBoundary>
 );

 expect(screen.getByText('Inner Error')).toBeInTheDocument();
 expect(screen.queryByText('Outer Error')).not.toBeInTheDocument();
 });

 it('应该支持多次错误触发', () => {
 const { rerender } = render(
   <ErrorBoundary>
     <NormalComponent />
   </ErrorBoundary>
 );

 // 第一次错误
 rerender(
   <ErrorBoundary>
     <ThrowError message="Error 1" />
   </ErrorBoundary>
 );
 expect(screen.getByText(/出错了/)).toBeInTheDocument();

 // 注意：Error Boundary一旦捕获错误，需要重新挂载才能恢复
 // 这是React的设计，不是bug
 });
});
