export default function LoadingSpinner() {
  return (
    <div className="flex items-center justify-center h-screen">
      <div className="text-center">
        <div className="animate-spin rounded-full h-16 w-16 border-b-2 border-primary mx-auto mb-4"></div>
        <p className="text-text-light">加载中...</p>
      </div>
    </div>
  );
}
