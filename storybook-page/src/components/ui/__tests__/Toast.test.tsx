import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { ToastProvider, useToast } from '../Toast';

function ToastHarness() {
  const { showSuccess, showError, showWarning, showInfo } = useToast();

  return (
    <div>
      <button onClick={() => showSuccess('保存成功', 0)}>success</button>
      <button onClick={() => showError('保存失败', 0)}>error</button>
      <button onClick={() => showWarning('需要关注', 0)}>warning</button>
      <button onClick={() => showInfo('系统提示', 50)}>info</button>
    </div>
  );
}

describe('Toast', () => {
  it('支持展示不同类型的 toast，并可手动关闭', async () => {
    render(
      <ToastProvider>
        <ToastHarness />
      </ToastProvider>
    );

    fireEvent.click(screen.getByRole('button', { name: 'success' }));
    fireEvent.click(screen.getByRole('button', { name: 'error' }));
    fireEvent.click(screen.getByRole('button', { name: 'warning' }));

    expect(screen.getByText('保存成功')).toBeInTheDocument();
    expect(screen.getByText('保存失败')).toBeInTheDocument();
    expect(screen.getByText('需要关注')).toBeInTheDocument();

    fireEvent.click(screen.getAllByRole('button', { name: '关闭提示' })[0]);

    await waitFor(() => {
      expect(screen.queryByText('保存成功')).not.toBeInTheDocument();
    });
  });

  it('带 duration 的 toast 会自动消失', async () => {
    render(
      <ToastProvider>
        <ToastHarness />
      </ToastProvider>
    );

    fireEvent.click(screen.getByRole('button', { name: 'info' }));
    expect(screen.getByText('系统提示')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.queryByText('系统提示')).not.toBeInTheDocument();
    }, { timeout: 1000 });
  });
});
