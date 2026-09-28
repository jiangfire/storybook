import { useState } from 'react';
import type { ChangeEvent, KeyboardEvent } from 'react';
import { XIcon } from '../ui/AppIcon';

interface TagManagerProps {
  tags: string[];
  onChange: (tags: string[]) => void;
  /** 可选：来自相似故事的推荐标签（依赖向量服务，未启用时父级不传） */
  suggestions?: string[];
  /** 可选：请求推荐标签 */
  onRequestSuggestions?: () => void;
}

export const TagManager = ({ tags, onChange, suggestions, onRequestSuggestions }: TagManagerProps) => {
  const [newTag, setNewTag] = useState('');

  const handleInputChange = (e: ChangeEvent<HTMLInputElement>) => {
    setNewTag(e.target.value);
  };

  const handleAdd = () => {
    const trimmed = newTag.trim();
    if (!trimmed || tags.includes(trimmed)) return;

    onChange([...tags, trimmed]);
    setNewTag('');
  };

  const handleAddSuggestion = (tag: string) => {
    if (tags.includes(tag)) return;
    onChange([...tags, tag]);
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleAdd();
    }
  };

  const handleRemove = (tag: string) => {
    onChange(tags.filter((t) => t !== tag));
  };

  return (
    <div>
      <label className="block text-sm font-medium text-text mb-2">标签</label>
      <div className="mb-3 flex flex-wrap gap-2">
        {tags.map((tag) => (
          <span
            key={tag}
            className="inline-flex items-center rounded-full bg-primary-50 px-3 py-1 text-sm text-primary"
          >
            {tag}
            <button
              type="button"
              onClick={() => handleRemove(tag)}
              aria-label={`删除标签${tag}`}
              className="ml-2 text-primary hover:text-primary-700"
            >
              <XIcon size={12} />
            </button>
          </span>
        ))}
      </div>
      <div className="flex flex-col gap-2 sm:flex-row">
        <input
          type="text"
          value={newTag}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          className="flex-1 rounded-lg border border-border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
          placeholder="添加标签...（按 Enter 快速添加）"
        />
        <button
          type="button"
          onClick={handleAdd}
          className="rounded-lg bg-secondary px-4 py-2 text-text transition-colors hover:bg-primary-50"
        >
          添加
        </button>
      </div>

      {onRequestSuggestions && (
        <button
          type="button"
          onClick={onRequestSuggestions}
          className="mt-2 text-xs text-text-light transition-colors hover:text-primary"
        >
          根据标题与描述推荐标签
        </button>
      )}

      {suggestions && suggestions.length > 0 && (
        <div className="mt-2 flex flex-wrap items-center gap-2">
          <span className="text-xs text-text-light">推荐：</span>
          {suggestions
            .filter((tag) => !tags.includes(tag))
            .map((tag) => (
              <button
                key={tag}
                type="button"
                onClick={() => handleAddSuggestion(tag)}
                className="rounded-full border border-border px-2.5 py-0.5 text-xs text-text-light transition-colors hover:border-primary hover:text-primary"
              >
                + {tag}
              </button>
            ))}
        </div>
      )}
    </div>
  );
};
