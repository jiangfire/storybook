export default function LoadingSpinner() {
  return (
    <div className="flex min-h-[40vh] items-center justify-center px-4">
      <div className="surface-card rounded-[1.6rem] px-8 py-10 text-center">
        <div className="mx-auto mb-4 h-12 w-12 animate-spin rounded-full border-b-2 border-primary" />
        <p className="text-text-light">加载中...</p>
      </div>
    </div>
  );
}
