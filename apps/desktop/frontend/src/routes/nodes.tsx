import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/nodes")({
  component: NodesPage,
});

function NodesPage() {
  return (
    <div className="p-6">
      <h1 className="text-xl font-semibold mb-6">节点管理</h1>
      <div className="text-gray-400">开发中...</div>
    </div>
  );
}
