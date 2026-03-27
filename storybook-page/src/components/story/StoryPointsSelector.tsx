interface StoryPointsSelectorProps {
  value: number | undefined;
  onChange: (value: number | undefined) => void;
  label?: string;
  className?: string;
}

const STORY_POINTS = [1, 2, 3, 5, 8, 13] as const;

export const StoryPointsSelector = ({
  value,
  onChange,
  label = '故事点',
  className = '',
}: StoryPointsSelectorProps) => {
  const handleSelect = (points: number) => {
    // 如果点击已选中的值，则取消选择
    onChange(value === points ? undefined : points);
  };

  return (
    <div className={className}>
      {label && (
        <label className="block text-sm font-medium text-text mb-2">
          {label}
        </label>
      )}
      <div className="grid grid-cols-3 gap-2 sm:grid-cols-6">
        {STORY_POINTS.map((points) => (
          <button
            key={points}
            type="button"
            onClick={() => handleSelect(points)}
            className={`rounded-lg border px-3 py-2 text-sm font-medium transition-all ${
              value === points
                ? 'border-primary bg-primary text-white'
                : 'border-border bg-white text-text hover:border-primary-300'
            }`}
          >
            {points}
          </button>
        ))}
      </div>
    </div>
  );
};
