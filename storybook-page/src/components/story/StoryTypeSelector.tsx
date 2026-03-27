import type { StoryType } from '../../types/models';
import { SparklesIcon } from '../ui/AppIcon';
import { BugIcon } from '../ui/AppIcon';
import { ArchiveIcon } from '../ui/AppIcon';

interface StoryTypeOption {
  value: StoryType;
  label: string;
  icon: typeof SparklesIcon;
  activeClass: string;
}

const storyTypeOptions: StoryTypeOption[] = [
  {
    value: 'feature',
    label: '功能',
    icon: SparklesIcon,
    activeClass: 'border-blue-500 bg-blue-50 text-blue-700',
  },
  {
    value: 'bug',
    label: 'Bug',
    icon: BugIcon,
    activeClass: 'border-red-500 bg-red-50 text-red-700',
  },
  {
    value: 'chore',
    label: '杂项',
    icon: ArchiveIcon,
    activeClass: 'border-primary-500 bg-secondary-50 text-text',
  },
];

interface StoryTypeSelectorProps {
  value: StoryType;
  onChange: (value: StoryType) => void;
  label?: string;
  className?: string;
}

export const StoryTypeSelector = ({
  value,
  onChange,
  label = '故事类型',
  className = '',
}: StoryTypeSelectorProps) => {
  return (
    <div className={className}>
      {label && (
        <label className="block text-sm font-medium text-text mb-2">
          {label}
        </label>
      )}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        {storyTypeOptions.map((type) => {
          const Icon = type.icon;
          const isSelected = value === type.value;

          return (
            <button
              key={type.value}
              type="button"
              onClick={() => onChange(type.value)}
              className={`rounded-lg border-2 p-3 text-left transition-all ${
                isSelected ? type.activeClass : 'border-border bg-white hover:border-primary-300'
              }`}
            >
              <div className="mb-2 inline-flex h-9 w-9 items-center justify-center rounded-lg bg-white/80">
                <Icon size={18} />
              </div>
              <div className="text-sm font-medium">{type.label}</div>
            </button>
          );
        })}
      </div>
    </div>
  );
};
