import { AlertCircle } from "lucide-react";

export function ErrorCard({ message }: { message: string }) {
  return (
    <div className="flex items-center gap-3 bg-red-950 border border-red-700 text-red-300 rounded-lg p-4">
      <AlertCircle className="w-5 h-5 shrink-0" />
      <p className="text-sm">{message}</p>
    </div>
  );
}
