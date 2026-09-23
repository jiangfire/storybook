import { useState } from 'react';
import type { ChangeEvent, FormEvent } from 'react';
import { aiService } from '../../services/aiService';
import { getErrorMessage } from '../../utils/error';
import type { AIGeneratedStoryResponse, AIFormDraft } from '../../types/api';

interface AICreatorProps {
  onGenerated: (draft: AIFormDraft, strategy: 'replace' | 'fill_empty') => void;
  strategy?: 'replace' | 'fill_empty';
}

const MIN_REQUIREMENT_LENGTH = 5;
const MAX_REQUIREMENT_LENGTH = 2000;

export const AICreator = ({ onGenerated, strategy = 'replace' }: AICreatorProps) => {
  const [requirement, setRequirement] = useState('');
  const [isGenerating, setIsGenerating] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState<AIGeneratedStoryResponse | null>(null);

  const handleInputChange = (e: ChangeEvent<HTMLTextAreaElement>) => {
    setRequirement(e.target.value);
    if (error) {
      setError('');
    }
  };

  const handleGenerate = async (event?: FormEvent) => {
    event?.preventDefault();
    const text = requirement.trim();
    if (text.length < MIN_REQUIREMENT_LENGTH) {
      setError(`再多描述几个字（至少 ${MIN_REQUIREMENT_LENGTH} 个字），AI 才能理解你的需求`);
      return;
    }

    setIsGenerating(true);
    setError('');

    try {
      const data = await aiService.generateStory({ requirement: text });
      onGenerated(data.form_draft, strategy);
      setResult(data);
    } catch (err: unknown) {
      setResult(null);
      setError(getErrorMessage(err, 'AI 生成失败，请稍后重试'));
    } finally {
      setIsGenerating(false);
    }
  };

  const trimmedLength = requirement.trim().length;
  const isDisabled = trimmedLength < MIN_REQUIREMENT_LENGTH || isGenerating;
  const sourceLabel =
    result?.source === 'openai' ? 'AI 模型生成' : result ? '规则草稿（未调用大模型）' : '';

  return (
    <section
      aria-label="一句话创建故事"
      className="rounded-2xl border border-primary-200 bg-gradient-to-br from-primary-50 via-white to-accent-50 p-5 shadow-sm"
    >
      <div className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
        <div className="space-y-1">
          <h3 className="text-lg font-semibold text-text">一句话创建故事</h3>
          <p className="text-sm text-text-light">
            用一句话描述需求，AI 会生成标题、描述、验收标准等完整草稿并填入下方表单，生成后可继续手动修改。
          </p>
        </div>
        {result && (
          <span
            className={`inline-flex w-fit shrink-0 items-center gap-1 rounded-full px-3 py-1 text-xs font-medium ${
              result.source === 'openai'
                ? 'bg-primary-100 text-primary'
                : 'bg-amber-100 text-amber-700'
            }`}
          >
            来源：{sourceLabel}
          </span>
        )}
      </div>

      <form className="mt-4 space-y-3" onSubmit={(e) => void handleGenerate(e)}>
        <div>
          <label htmlFor="ai-requirement" className="mb-2 block text-sm font-medium text-text">
            需求描述
          </label>
          <textarea
            id="ai-requirement"
            value={requirement}
            onChange={handleInputChange}
            rows={3}
            autoFocus
            placeholder="例：支持用户用手机号和短信验证码登录"
            maxLength={MAX_REQUIREMENT_LENGTH}
            className="w-full rounded-xl border border-primary-200 bg-white/90 px-3 py-3 text-sm leading-6 text-text shadow-sm outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/20 resize-none"
          />
        </div>

        {error && (
          <div role="alert" className="rounded-lg border border-danger/30 bg-red-50 px-3 py-2 text-xs text-danger">
            {error}
          </div>
        )}

        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <span className="text-xs text-text-light">
            {trimmedLength}/{MAX_REQUIREMENT_LENGTH}（至少 {MIN_REQUIREMENT_LENGTH} 字）
            {isGenerating ? ' · 生成最长约 45 秒，请稍候' : ''}
          </span>
          <button
            type="submit"
            disabled={isDisabled}
            className="inline-flex items-center justify-center rounded-xl border border-transparent bg-primary px-5 py-2.5 text-sm font-medium text-white shadow-sm hover:bg-primary-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isGenerating ? '生成中…' : '生成完整草稿'}
          </button>
        </div>
      </form>

      {result && (
        <div className="mt-3 rounded-xl border border-success/30 bg-white/80 px-3 py-2.5 text-xs text-text-light">
          <span className="font-medium text-text">草稿已填入下方表单</span>
          ，请检查各字段后保存。
          {result.warnings && result.warnings.length > 0 && (
            <span className="ml-1 text-amber-700">{result.warnings[0]}</span>
          )}
        </div>
      )}
    </section>
  );
};
