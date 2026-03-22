import { useCallback, useEffect, useState } from 'react';
import { aiService } from '../../services/aiService';
import type { AIConfigTestResponse, AIConfigUpdateRequest } from '../../types/api';
import { getErrorMessage } from '../../utils/error';
import LoadingSpinner from '../../components/ui/LoadingSpinner';
import Button from '../../components/ui/Button';
import { useToast } from '../../components/ui/Toast';

type ConfigForm = {
  api_key: string;
  model: string;
  temperature: string;
  max_tokens: string;
  enabled: boolean;
  api_key_masked: string;
  is_configured: boolean;
};

const defaultForm: ConfigForm = {
  api_key: '',
  model: 'gpt-4o-mini',
  temperature: '0.2',
  max_tokens: '1200',
  enabled: false,
  api_key_masked: '',
  is_configured: false,
};

export default function AIConfigPage() {
  const { showSuccess, showError } = useToast();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [form, setForm] = useState<ConfigForm>(defaultForm);
  const [testResult, setTestResult] = useState<AIConfigTestResponse | null>(null);

  const loadConfig = useCallback(async () => {
    try {
      setLoading(true);
      const data = await aiService.getConfig();
      setForm({
        api_key: '',
        model: data.config.model || 'gpt-4o-mini',
        temperature: String(data.config.temperature ?? 0.2),
        max_tokens: String(data.config.max_tokens ?? 1200),
        enabled: Boolean(data.config.enabled),
        api_key_masked: data.config.api_key_masked || '',
        is_configured: Boolean(data.config.is_configured),
      });
    } catch (error: unknown) {
      showError(getErrorMessage(error, '加载 AI 配置失败'));
    } finally {
      setLoading(false);
    }
  }, [showError]);

  useEffect(() => {
    void loadConfig();
  }, [loadConfig]);

  const buildPayload = (): AIConfigUpdateRequest => ({
    api_key: form.api_key.trim() || undefined,
    model: form.model.trim(),
    temperature: Number(form.temperature),
    max_tokens: Number(form.max_tokens),
    enabled: form.enabled,
  });

  const handleSave = async () => {
    try {
      setSaving(true);
      const data = await aiService.updateConfig(buildPayload());
      setForm((prev) => ({
        ...prev,
        api_key: '',
        api_key_masked: data.config.api_key_masked,
        is_configured: data.config.is_configured,
      }));
      showSuccess('AI 配置已保存');
    } catch (error: unknown) {
      showError(getErrorMessage(error, '保存 AI 配置失败'));
    } finally {
      setSaving(false);
    }
  };

  const handleTest = async () => {
    try {
      setTesting(true);
      const data = await aiService.testConfig(buildPayload());
      setTestResult(data);
      showSuccess('AI 配置测试通过');
    } catch (error: unknown) {
      setTestResult(null);
      showError(getErrorMessage(error, 'AI 配置测试失败'));
    } finally {
      setTesting(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <LoadingSpinner />
      </div>
    );
  }

  return (
    <div className="container mx-auto px-4 py-6 space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-text mb-2">AI 配置</h1>
          <p className="text-text-light">配置 OpenAI，用于根据需求自动生成并回填用户故事表单。</p>
        </div>
        <div
          className={`px-3 py-1.5 rounded-full text-xs font-medium ${
            form.enabled && form.is_configured
              ? 'bg-green-100 text-green-800'
              : 'bg-yellow-100 text-yellow-800'
          }`}
        >
          {form.enabled && form.is_configured ? '已启用 OpenAI' : '当前将回退到规则草稿'}
        </div>
      </div>

      <div className="bg-white rounded-lg shadow-sm border border-border p-6 space-y-5">
        <div className="grid gap-5 md:grid-cols-2">
          <div>
            <label className="block text-sm font-medium text-text mb-2">Provider</label>
            <input
              value="OpenAI"
              readOnly
              className="w-full px-3 py-2 border border-border rounded-lg bg-secondary-50 text-text-light"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-text mb-2">模型</label>
            <input
              value={form.model}
              onChange={(event) => setForm((prev) => ({ ...prev, model: event.target.value }))}
              placeholder="例如 gpt-4o-mini"
              className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
            />
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-text mb-2">OpenAI API Key</label>
          <input
            type="password"
            value={form.api_key}
            onChange={(event) => setForm((prev) => ({ ...prev, api_key: event.target.value }))}
            placeholder={form.api_key_masked || '输入新的 API Key'}
            className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
          />
          <p className="mt-2 text-xs text-text-light">
            {form.api_key_masked
              ? `当前已保存密钥：${form.api_key_masked}。留空表示沿用现有值。`
              : '当前未保存 API Key。'}
          </p>
        </div>

        <div className="grid gap-5 md:grid-cols-2">
          <div>
            <label className="block text-sm font-medium text-text mb-2">Temperature</label>
            <input
              type="number"
              min="0"
              max="2"
              step="0.1"
              value={form.temperature}
              onChange={(event) =>
                setForm((prev) => ({ ...prev, temperature: event.target.value }))
              }
              className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-text mb-2">Max Tokens</label>
            <input
              type="number"
              min="256"
              max="8192"
              step="1"
              value={form.max_tokens}
              onChange={(event) => setForm((prev) => ({ ...prev, max_tokens: event.target.value }))}
              className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
            />
          </div>
        </div>

        <label className="flex items-center justify-between gap-3 rounded-lg border border-border p-4">
          <div>
            <div className="font-medium text-text">启用 AI 生成功能</div>
            <div className="text-xs text-text-light mt-1">
              关闭后，故事表单仍可使用内置规则生成草稿，但不会调用 OpenAI。
            </div>
          </div>
          <input
            type="checkbox"
            checked={form.enabled}
            onChange={(event) => setForm((prev) => ({ ...prev, enabled: event.target.checked }))}
            className="h-4 w-4"
          />
        </label>

        <div className="flex flex-wrap justify-end gap-3">
          <Button
            variant="secondary"
            onClick={() => void loadConfig()}
            disabled={saving || testing}
          >
            重新加载
          </Button>
          <Button variant="secondary" onClick={handleTest} disabled={saving} isLoading={testing}>
            测试连接
          </Button>
          <Button onClick={handleSave} disabled={testing} isLoading={saving}>
            保存配置
          </Button>
        </div>
      </div>

      {testResult && (
        <div className="bg-white rounded-lg shadow-sm border border-border p-6 space-y-4">
          <div className="flex items-center justify-between gap-3">
            <div>
              <h2 className="text-lg font-semibold text-text">测试结果</h2>
              <p className="text-sm text-text-light">
                Provider: {testResult.provider} · Model: {testResult.model}
              </p>
            </div>
            <span className="px-3 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800">
              可用
            </span>
          </div>

          <div className="rounded-lg border border-border bg-secondary-50 p-4 space-y-2 text-sm">
            <div>
              <span className="text-text-light">标题：</span>
              <span className="text-text font-medium">{testResult.preview.title}</span>
            </div>
            <div>
              <span className="text-text-light">故事：</span>
              <span className="text-text">{testResult.preview.user_story}</span>
            </div>
            <div className="flex flex-wrap gap-3">
              <span className="text-text-light">类型：{testResult.preview.story_type}</span>
              <span className="text-text-light">优先级：{testResult.preview.priority}</span>
              <span className="text-text-light">故事点：{testResult.preview.story_points}</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
