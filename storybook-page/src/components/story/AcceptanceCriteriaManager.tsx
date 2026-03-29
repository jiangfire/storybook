import { useState } from 'react';
import type { ChangeEvent, KeyboardEvent } from 'react';
import { XIcon } from '../ui/AppIcon';

export interface AcceptanceCriterion {
  description: string;
  order: number;
}

interface AcceptanceCriteriaManagerProps {
  criteria: AcceptanceCriterion[];
  onChange: (criteria: AcceptanceCriterion[]) => void;
}

export const AcceptanceCriteriaManager = ({
  criteria,
  onChange,
}: AcceptanceCriteriaManagerProps) => {
  const [newACText, setNewACText] = useState('');

  const handleInputChange = (e: ChangeEvent<HTMLInputElement>) => {
    setNewACText(e.target.value);
  };

  const handleAdd = () => {
    const trimmed = newACText.trim();
    if (!trimmed) return;

    const newAC: AcceptanceCriterion = {
      description: trimmed,
      order: criteria.length,
    };

    onChange([...criteria, newAC]);
    setNewACText('');
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleAdd();
    }
  };

  const handleRemove = (index: number) => {
    onChange(criteria.filter((_, i) => i !== index));
  };

  return (
    <div>
      <label className="block text-sm font-medium text-text mb-2">
        验收标准 (AC)
      </label>
      <div className="mb-3 space-y-2">
        {criteria.map((ac, index) => (
          <div
            key={index}
            className="flex items-start space-x-2 rounded-lg bg-secondary-50 p-3"
          >
            <span className="mt-1 text-text-light">{index + 1}.</span>
            <span className="flex-1 text-sm">{ac.description}</span>
            <button
              type="button"
              onClick={() => handleRemove(index)}
              aria-label="删除验收标准"
              className="text-danger hover:text-danger-700"
            >
              <XIcon size={14} />
            </button>
          </div>
        ))}
      </div>
      <div className="flex flex-col gap-2 sm:flex-row">
        <input
          type="text"
          value={newACText}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          className="flex-1 rounded-lg border border-border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
          placeholder="添加验收标准...（按 Enter 快速添加）"
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
