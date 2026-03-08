import { useCallback, useEffect, useState } from 'react';
import Button from '../ui/Button';
import { testCaseService } from '../../services/testCaseService';
import type { TestCaseItem } from '../../types/api';
import { getErrorMessage } from '../../utils/error';
import { useToast } from '../ui/Toast';

interface StoryTestCasesPanelProps {
  storyId: number;
}

export default function StoryTestCasesPanel({ storyId }: StoryTestCasesPanelProps) {
  const { showError, showSuccess } = useToast();
  const [cases, setCases] = useState<TestCaseItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState({
    title: '',
    description: '',
    stepsText: '',
    expected_result: '',
  });

  const loadCases = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const data = await testCaseService.getStoryTestCases(storyId);
      setCases(data.test_cases || []);
    } catch (err: unknown) {
      setCases([]);
      setError(getErrorMessage(err, '测试用例加载失败'));
    } finally {
      setLoading(false);
    }
  }, [storyId]);

  useEffect(() => {
    void loadCases();
  }, [loadCases]);

  const handleCreate = async () => {
    const title = form.title.trim();
    if (!title) {
      showError('请输入测试用例标题');
      return;
    }
    const steps = form.stepsText
      .split('\n')
      .map((line) => line.trim())
      .filter(Boolean);
    if (steps.length === 0) {
      showError('至少输入一个测试步骤（每行一条）');
      return;
    }

    try {
      setCreating(true);
      await testCaseService.createTestCase(storyId, {
        title,
        description: form.description.trim() || undefined,
        steps,
        expected_result: form.expected_result.trim() || undefined,
      });
      setForm({ title: '', description: '', stepsText: '', expected_result: '' });
      showSuccess('测试用例创建成功');
      await loadCases();
    } catch (err: unknown) {
      showError(getErrorMessage(err, '测试用例创建失败'));
    } finally {
      setCreating(false);
    }
  };

  const handleStatusChange = async (id: number, status: TestCaseItem['status']) => {
    try {
      await testCaseService.updateTestCaseStatus(id, { status });
      setCases((prev) => prev.map((item) => (item.id === id ? { ...item, status } : item)));
    } catch (err: unknown) {
      showError(getErrorMessage(err, '测试用例状态更新失败'));
    }
  };

  return (
    <div className="bg-white rounded-xl border border-border p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-text">测试用例</h2>
        <Button size="sm" variant="secondary" onClick={() => void loadCases()} disabled={loading}>
          刷新
        </Button>
      </div>

      <div className="space-y-2">
        <input
          value={form.title}
          onChange={(e) => setForm((prev) => ({ ...prev, title: e.target.value }))}
          placeholder="测试用例标题"
          className="w-full px-3 py-2 border border-border rounded-lg"
        />
        <textarea
          value={form.description}
          onChange={(e) => setForm((prev) => ({ ...prev, description: e.target.value }))}
          placeholder="描述（可选）"
          rows={2}
          className="w-full px-3 py-2 border border-border rounded-lg resize-none"
        />
        <textarea
          value={form.stepsText}
          onChange={(e) => setForm((prev) => ({ ...prev, stepsText: e.target.value }))}
          placeholder="测试步骤（每行一条）"
          rows={3}
          className="w-full px-3 py-2 border border-border rounded-lg resize-none"
        />
        <input
          value={form.expected_result}
          onChange={(e) => setForm((prev) => ({ ...prev, expected_result: e.target.value }))}
          placeholder="预期结果（可选）"
          className="w-full px-3 py-2 border border-border rounded-lg"
        />
        <div className="flex justify-end">
          <Button size="sm" onClick={handleCreate} isLoading={creating}>
            新建测试用例
          </Button>
        </div>
      </div>

      {error && <div className="text-sm text-danger">{error}</div>}
      {loading ? (
        <div className="text-sm text-text-light">测试用例加载中...</div>
      ) : cases.length === 0 ? (
        <div className="text-sm text-text-light">暂无测试用例</div>
      ) : (
        <div className="space-y-2">
          {cases.map((item) => (
            <div key={item.id} className="border border-border rounded-lg p-3">
              <div className="flex flex-col md:flex-row md:items-center gap-2 md:justify-between">
                <div className="min-w-0">
                  <div className="font-medium text-text truncate">{item.title}</div>
                  <div className="text-xs text-text-light">
                    步骤 {item.steps?.length || 0} 条 · 创建于{' '}
                    {new Date(item.created_at).toLocaleString()}
                  </div>
                </div>
                <select
                  value={item.status}
                  onChange={(e) =>
                    void handleStatusChange(
                      item.id,
                      e.target.value as 'pending' | 'passed' | 'failed'
                    )
                  }
                  className="px-2 py-1 border border-border rounded text-sm"
                >
                  <option value="pending">待验证</option>
                  <option value="passed">通过</option>
                  <option value="failed">失败</option>
                </select>
              </div>
              {item.description && <div className="text-sm text-text mt-2">{item.description}</div>}
              {item.expected_result && (
                <div className="text-sm text-text-light mt-1">预期：{item.expected_result}</div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

