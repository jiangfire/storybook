interface PrioritySelectorProps {
  value: number;
  onChange: (value: number) => void;
  label?: string;
  className?: string;
}

const priorityOptions = [
  { value: 0, label: '无' },
  { value: 1, label: '低' },
  { value: 2, label: '中' },
  { value: 3, label: '高' },
  { value: 4, label: '紧急' },
];

const getPriorityStyles = (value: number, isSelected: boolean) => {
  if (!isSelected) {
    return 'bg-secondary-100 text-text-light hover:bg-secondary-200';
  }

  switch (value) {
    case 4:
      return 'bg-danger text-white';
    case 3:
      return 'bg-warning text-text';
    case 2:
      return 'bg-warning-light text-warning';
    case 1:
      return 'bg-info-light text-info';
    default:
      return 'bg-secondary-200 text-text';
  }
};

export const PrioritySelector = ({
  value,
  onChange,
  label = '优先级',
  className = '',
}: PrioritySelectorProps) => {
  return (
    <div className={className}>
      {label && (
        <label className="block text-sm font-medium text-text mb-2">
          {label}
        </label>
      )}
      <div className="flex flex-wrap items-center gap-2">
        {priorityOptions.map((option) => (
          <button
            key={option.value}
            type="button"
            onClick={() => onChange(option.value)}
            className={`rounded-full px-3 py-1.5 text-sm transition-all ${getPriorityStyles(
              option.value,
              value === option.value
            )}`}
          >
            {option.label}
          </button>
        ))}
      </div>
    </div>
  );
};
