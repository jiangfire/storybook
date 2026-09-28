import { useState } from 'react';
import { aiService } from '../../services/aiService';
import { storyService } from '../../services/storyService';
import { useToast } from '../ui/Toast';
import type { AIDoRCheckData, AITranslateData } from '../../types/api';
import { cn } from '../../utils/cn';
import Button from '../ui/Button';

interface StoryAIPanelProps {
  storyId: number;
  /** 仅产品经理/管理员可优化 AC（后端同口径） */
  canRefineAC: boolean;
  /** 应用 AC 建议后刷新故事 */
  onACChanged?: () => void;
}

export const StoryAIPanel = ({ storyId, canRefineAC, onACChanged }: StoryAIPanelProps) => {
  const { showError, showSuccess } = useToast();

  const [dor, setDor] = useState<AIDoRCheckData | null>(null);
  const [isDorLoading, setIsDorLoading] = useState(false);

  const [summary, setSummary] = useState<string | null>(null);
  const [isSummaryLoading, setIsSummaryLoading] = useState(false);

  const [translation, setTranslation] = useState<AITranslateData | null>(null);
  const [translateLang, setTranslateLang] = useState<'en' | 'zh'>('en');
  const [isTranslateLoading, setIsTranslateLoading] = useState(false);

  const [refineFeedback, setRefineFeedback] = useState('');
  const [refineSuggestions, setRefineSuggestions] = useState<string[] | null>(null);
  const [appliedIndices, setAppliedIndices] = useState<number[]>([]);
  const [isRefineLoading, setIsRefineLoading] = useState(false);

  const handleDoRCheck = async () => {
    try {
      setIsDorLoading(true);
      setDor(await aiService.checkDoR(storyId));
    } catch (error) {
      showError(error instanceof Error ? error.message : 'DoR 检查失败');
    } finally {
      setIsDorLoading(false);
    }
  };

  const handleSummarize = async () => {
    try {
      setIsSummaryLoading(true);
      const data = await aiService.summarizeStory(storyId);
      setSummary(data.summary);
    } catch (error) {
      showError(error instanceof Error ? error.message : '摘要生成失败');
    } finally {
      setIsSummaryLoading(false);
    }
  };

  const handleTranslate = async () => {
    try {
      setIsTranslateLoading(true);
      setTranslation(await aiService.translateStory(storyId, translateLang));
    } catch (error) {
      showError(error instanceof Error ? error.message : '翻译失败');
    } finally {
      setIsTranslateLoading(false);
    }
  };

  const handleRefineAC = async () => {
    const feedback = refineFeedback.trim();
    if (feedback.length < 2) {
      showError('请先描述优化方向（至少 2 个字符）');
      return;
    }
    try {
      setIsRefineLoading(true);
      const data = await aiService.refineAC(storyId, feedback);
      setRefineSuggestions(data.suggested);
      setAppliedIndices([]);
    } catch (error) {
      showError(error instanceof Error ? error.message : 'AC 优化失败');
    } finally {
      setIsRefineLoading(false);
    }
  };

  const handleApplySuggestion = async (index: number, description: string) => {
    try {
      await storyService.addAC(storyId, { description });
      setAppliedIndices((prev) => [...prev, index]);
      showSuccess('已添加该验收标准');
      onACChanged?.();
    } catch (error) {
      showError(error instanceof Error ? error.message : '添加验收标准失败');
    }
  };

  return (
    <div className="bg-white rounded-xl border border-primary-100 p-6 space-y-5">
      <div className="space-y-1">
        <h2 className="text-lg font-semibold text-text">AI 助手</h2>
        <p className="text-sm text-text-light">DoR 检查、干系人摘要、翻译与验收标准优化。</p>
      </div>

      {/* DoR 检查 */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-medium text-text">准备就绪检查（DoR）</h3>
          <Button size="sm" variant="secondary" onClick={() => void handleDoRCheck()} isLoading={isDorLoading}>
            检查
          </Button>
        </div>
        {dor && (
          <div className="space-y-1.5 text-sm">
            <div className={cn('font-medium', dor.ready ? 'text-green-700' : 'text-yellow-700')}>
              {dor.ready ? '已就绪' : '尚未就绪'} · {dor.score} 分（{dor.passed}/{dor.total} 项通过）
            </div>
            <ul className="space-y-1">
              {dor.checks.map((check) => (
                <li key={check.key} className="flex items-center gap-2">
                  <span className={check.pass ? 'text-green-600' : 'text-red-500'}>
                    {check.pass ? '✓' : '✗'}
                  </span>
                  <span className={cn(check.pass ? 'text-text' : 'text-danger')}>
                    {check.title}
                  </span>
                </li>
              ))}
            </ul>
            {dor.suggestions.length > 0 && (
              <div className="text-xs text-text-light">建议：{dor.suggestions.join('；')}</div>
            )}
          </div>
        )}
      </div>

      {/* 干系人摘要 */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-medium text-text">干系人摘要</h3>
          <Button
            size="sm"
            variant="secondary"
            onClick={() => void handleSummarize()}
            isLoading={isSummaryLoading}
          >
            生成
          </Button>
        </div>
        {summary && <p className="text-sm text-text whitespace-pre-wrap">{summary}</p>}
      </div>

      {/* 翻译 */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-medium text-text">翻译</h3>
          <div className="flex items-center gap-2">
            <select
              value={translateLang}
              onChange={(e) => setTranslateLang(e.target.value as 'en' | 'zh')}
              className="border border-border rounded-lg px-2 py-1 text-sm"
              aria-label="目标语言"
            >
              <option value="en">英文</option>
              <option value="zh">中文</option>
            </select>
            <Button
              size="sm"
              variant="secondary"
              onClick={() => void handleTranslate()}
              isLoading={isTranslateLoading}
            >
              翻译
            </Button>
          </div>
        </div>
        {translation && (
          <div className="text-sm space-y-1.5">
            <p className="font-medium text-text">{translation.translated.title}</p>
            <p className="text-text-light whitespace-pre-wrap">
              {translation.translated.description}
            </p>
            {translation.translated.ac.length > 0 && (
              <ul className="text-text-light list-disc pl-4">
                {translation.translated.ac.map((ac, i) => (
                  <li key={i}>{ac}</li>
                ))}
              </ul>
            )}
          </div>
        )}
      </div>

      {/* AC 优化 */}
      {canRefineAC && (
        <div className="space-y-2">
          <h3 className="text-sm font-medium text-text">优化验收标准</h3>
          <textarea
            value={refineFeedback}
            onChange={(e) => setRefineFeedback(e.target.value)}
            placeholder="描述优化方向，如：把每条 AC 改成 Given/When/Then 格式，并补充异常场景"
            className="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary resize-none"
            rows={2}
          />
          <Button size="sm" onClick={() => void handleRefineAC()} isLoading={isRefineLoading}>
            生成建议
          </Button>
          {refineSuggestions && (
            <div className="space-y-2">
              {refineSuggestions.length === 0 ? (
                <p className="text-sm text-text-light">模型未返回有效建议，请调整反馈后重试。</p>
              ) : (
                refineSuggestions.map((suggestion, index) => (
                  <div
                    key={index}
                    className="flex items-start justify-between gap-2 rounded-lg border border-border p-2"
                  >
                    <p className="text-sm text-text flex-1">{suggestion}</p>
                    <Button
                      size="sm"
                      variant="secondary"
                      disabled={appliedIndices.includes(index)}
                      onClick={() => void handleApplySuggestion(index, suggestion)}
                    >
                      {appliedIndices.includes(index) ? '已添加' : '添加'}
                    </Button>
                  </div>
                ))
              )}
              <p className="text-xs text-text-light">建议仅供参考，逐条确认后添加。</p>
            </div>
          )}
        </div>
      )}
    </div>
  );
};
