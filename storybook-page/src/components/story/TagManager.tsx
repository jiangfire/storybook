import { useState, ChangeEvent } from 'react';
import { XIcon } from '../ui/AppIcon';

interface TagManagerProps {
  tags: string[];
  onChange: (tags: string[]) => void;
}

export const TagManager = ({ tags, onChange }: TagManagerProps) => {
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

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
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
    </div>
  );
};
