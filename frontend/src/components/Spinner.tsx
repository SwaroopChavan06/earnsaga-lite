export function Spinner({ className = "h-8 w-8" }: { className?: string }) {
  return (
    <div className={`${className} animate-spin rounded-full border-4 border-slate-600 border-t-indigo-500`} />
  );
}
