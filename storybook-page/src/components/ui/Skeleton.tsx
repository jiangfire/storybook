interface SkeletonProps {
  className?: string;
}

export function Skeleton({ className = '' }: SkeletonProps) {
  return (
    <div
      className={`animate-pulse bg-gray-200 rounded ${className}`}
      role="status"
      aria-label="加载中"
    >
      <span className="sr-only">加载中...</span>
    </div>
  );
}

export function CardSkeleton() {
  return (
    <div className="bg-white rounded-lg border border-gray-200 p-4 space-y-3">
      <Skeleton className="h-4 w-3/4" />
      <Skeleton className="h-3 w-1/2" />
      <Skeleton className="h-3 w-1/3" />
    </div>
  );
}

export function BoardSkeleton() {
  return (
    <div className="grid grid-cols-5 gap-4 p-4">
      {[1, 2, 3, 4, 5].map((i) => (
        <div key={i} className="bg-gray-50 rounded-lg border border-gray-200 p-4 space-y-3">
          <Skeleton className="h-6 w-20" />
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-24 w-full" />
        </div>
      ))}
    </div>
  );
}

export function StoryDetailSkeleton() {
  return (
    <div className="max-w-6xl mx-auto p-8 space-y-6">
      <div className="space-y-4">
        <Skeleton className="h-8 w-1/3" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-2/3" />
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        <div className="lg:col-span-2 space-y-6">
          <div className="bg-white rounded-lg border border-gray-200 p-6 space-y-3">
            <Skeleton className="h-6 w-24" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-3/4" />
          </div>
          <div className="bg-white rounded-lg border border-gray-200 p-6 space-y-3">
            <Skeleton className="h-6 w-24" />
            <Skeleton className="h-20 w-full" />
          </div>
        </div>
        <div className="space-y-6">
          <div className="bg-white rounded-lg border border-gray-200 p-6 space-y-3">
            <Skeleton className="h-6 w-24" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-full" />
          </div>
        </div>
      </div>
    </div>
  );
}

export function ProjectListSkeleton() {
  return (
    <div className="p-8">
      {/* 头部骨架屏 */}
      <div className="flex items-center justify-between mb-8">
        <div className="space-y-2">
          <Skeleton className="h-8 w-24" />
          <Skeleton className="h-4 w-48" />
        </div>
        <Skeleton className="h-10 w-24" />
      </div>

      {/* 搜索框骨架屏 */}
      <div className="mb-6">
        <Skeleton className="h-10 w-full max-w-md" />
      </div>

      {/* 项目卡片骨架屏 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {[1, 2, 3, 4, 5, 6].map((i) => (
          <div key={i} className="bg-white rounded-lg border border-gray-200 p-6 space-y-4">
            <div className="flex items-center space-x-3">
              <Skeleton className="h-12 w-12 rounded-lg" />
              <div className="flex-1 space-y-2">
                <Skeleton className="h-5 w-3/4" />
                <Skeleton className="h-4 w-1/2" />
              </div>
            </div>
            <div className="space-y-2">
              <Skeleton className="h-3 w-full" />
              <Skeleton className="h-3 w-2/3" />
            </div>
            <div className="flex items-center justify-between pt-4 border-t border-gray-100">
              <div className="flex items-center space-x-2">
                <Skeleton className="h-6 w-6 rounded-full" />
                <Skeleton className="h-4 w-20" />
              </div>
              <Skeleton className="h-8 w-16" />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
