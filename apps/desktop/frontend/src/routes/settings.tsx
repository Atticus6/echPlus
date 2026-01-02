import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/settings")({
  component: SettingsPage,
});

function SettingsPage() {
  return (
    <div className="p-6">
      <h1 className="text-xl font-semibold mb-6">设置</h1>
      <div className="text-gray-400">开发中...</div>
    </div>
  );
}
