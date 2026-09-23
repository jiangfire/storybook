import { useRef, useState } from 'react';
import type { ChangeEvent, KeyboardEvent } from 'react';
import { aiService } from '../../services/aiService';
import { getErrorMessage } from '../../utils/error';
import type { AIChatMessage, AIFormDraft } from '../../types/api';

interface StoryChatPanelProps {
  /** 当前表单草稿，随对话上下文发送给 AI，保证增量完善而不是每次重来 */
  currentDraft: AIFormDraft | null;
  onApply: (draft: AIFormDraft, strategy: 'replace' | 'fill_empty') => void;
  strategy: 'replace' | 'fill_empty';
}

export function StoryChatPanel({ currentDraft, onApply, strategy }: StoryChatPanelProps) {
  const [messages, setMessages] = useState<AIChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [isSending, setIsSending] = useState(false);
  const [error, setError] = useState('');
  const [pendingDraft, setPendingDraft] = useState<AIFormDraft | null>(null);
  const [applied, setApplied] = useState(false);
  const scrollAnchorRef = useRef<HTMLDivElement | null>(null);

  const handleInputChange = (e: ChangeEvent<HTMLTextAreaElement>) => {
    setInput(e.target.value);
    if (error) {
      setError('');
    }
  };

  const scrollToBottom = () => {
    requestAnimationFrame(() => {
      // jsdom 等测试环境未实现 scrollIntoView，用可选调用兜底
      scrollAnchorRef.current?.scrollIntoView?.({ block: 'end' });
    });
  };

  const handleSend = async () => {
    const content = input.trim();
    if (!content || isSending) {
      return;
    }

    const nextMessages: AIChatMessage[] = [...messages, { role: 'user', content }];
    setMessages(nextMessages);
    setInput('');
    setIsSending(true);
    setError('');
    setPendingDraft(null);
    setApplied(false);
    scrollToBottom();

    try {
      const data = await aiService.storyChat({
        messages: nextMessages,
        current_draft: currentDraft,
      });
      setMessages([...nextMessages, { role: 'assistant', content: data.reply }]);
      setPendingDraft(data.form_draft ?? null);
      scrollToBottom();
    } catch (err: unknown) {
      setError(getErrorMessage(err, 'AI 回复失败，请稍后重试'));
      setMessages(nextMessages.slice(0, -1));
      setInput(content);
    } finally {
      setIsSending(false);
    }
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
      e.preventDefault();
      void handleSend();
    }
  };

  const handleApply = () => {
    if (!pendingDraft) {
      return;
    }
    onApply(pendingDraft, strategy);
    setApplied(true);
  };

  return (
    <section
      aria-label="对话式完善"
      className="rounded-2xl border border-border bg-white p-5 shadow-sm"
    >
      <div className="space-y-1">
        <h3 className="text-lg font-semibold text-text">对话式完善</h3>
        <p className="text-sm text-text-light">
          和 AI 聊几句把需求聊清楚，每一轮都会更新草稿，满意后一键写入下方表单。
        </p>
      </div>

      <div className="mt-4 max-h-64 space-y-3 overflow-y-auto rounded-xl bg-secondary-50 p-3">
        {messages.length === 0 && !isSending ? (
          <p className="py-4 text-center text-sm text-text-light">
            在下方输入框描述需求，例如“做一个手机号验证码登录，主要给APP用户用”
          </p>
        ) : (
          messages.map((msg, index) => (
            <div
              key={index}
              className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
            >
              <div
                className={`max-w-[85%] whitespace-pre-wrap rounded-2xl px-3 py-2 text-sm ${
                  msg.role === 'user'
                    ? 'rounded-br-sm bg-primary text-white'
                    : 'rounded-bl-sm border border-border bg-white text-text'
                }`}
              >
                {msg.content}
              </div>
            </div>
          ))
        )}
        {isSending && (
          <div className="flex justify-start">
            <div className="rounded-2xl rounded-bl-sm border border-border bg-white px-3 py-2 text-sm text-text-light">
              AI 正在思考…
            </div>
          </div>
        )}
        <div ref={scrollAnchorRef} />
      </div>

      {error && (
        <div role="alert" className="mt-3 rounded-lg border border-danger/30 bg-red-50 px-3 py-2 text-xs text-danger">
          {error}
        </div>
      )}

      {pendingDraft && (
        <div className="mt-3 flex flex-col gap-2 rounded-xl border border-primary-100 bg-primary-50/60 px-3 py-2.5 text-xs text-text-light sm:flex-row sm:items-center sm:justify-between">
          <span>
            本轮草稿已就绪
            {applied ? '，已写入下方表单' : '，可继续对话或写入表单'}
          </span>
          {!applied && (
            <button
              type="button"
              onClick={handleApply}
              className="w-fit rounded-xl bg-primary px-4 py-1.5 text-xs font-medium text-white transition-colors hover:bg-primary-700"
            >
              应用到表单
            </button>
          )}
        </div>
      )}

      <div className="mt-3">
        <label htmlFor="story-chat-input" className="sr-only">
          对话输入
        </label>
        <textarea
          id="story-chat-input"
          value={input}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          rows={2}
          maxLength={2000}
          placeholder="输入消息，Enter 发送，Shift+Enter 换行"
          className="w-full rounded-xl border border-border bg-white px-3 py-2.5 text-sm leading-6 text-text outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/20 resize-none"
        />
        <div className="mt-2 flex justify-end">
          <button
            type="button"
            onClick={() => void handleSend()}
            disabled={!input.trim() || isSending}
            className="inline-flex items-center justify-center rounded-xl border border-transparent bg-primary px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-primary-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isSending ? '发送中…' : '发送'}
          </button>
        </div>
      </div>
    </section>
  );
}

export default StoryChatPanel;
