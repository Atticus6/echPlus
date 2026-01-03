import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { LogService } from "../../bindings/github.com/atticus6/echPlus/apps/desktop/services";
import { cn } from "@/lib/utils";
import { RefreshCw } from "lucide-react";

export const Route = createFileRoute("/logs")({
  component: LogsPage,
});

const logTypes = [
  { value: "info", label: "Info" },
  { value: "error", label: "Error" },
  { value: "debug", label: "Debug" },
];

function LogsPage() {
  const [logType, setLogType] = useState("info");
  const [lines, setLines] = useState(100);

  const { data: logs, isLoading, refetch } = useQuery({
    queryKey: ["logs", logType, lines],
    queryFn: () => LogService.GetTodayLogs(logType, lines),
    refetchInterval: 5000,
  });

  const getLevelColor = (level: string) => {
    switch (level?.toUpperCase()) {
      case "ERROR":
        return "text-red-500";
      case "WARN":
        return "text-yellow-500";
      case "DEBUG":
        return "text-gray-400";
      default:
        return "text-blue-400";
    }
  };

  return (
    <div className="p-6 h-full flex flex-col">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-xl font-semibold">日志</h1>
        <div className="flex items-center gap-3">
          <select
            value={logType}
            onChange={(e) => setLogType(e.target.value)}
            className="px-3 py-1.5 rounded-md bg-gray-100 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 text-sm"
          >
            {logTypes.map((t) => (
              <option key={t.value} value={t.value}>{t.label}</option>
            ))}
          </select>
          <select
            value={lines}
            onChange={(e) => setLines(Number(e.target.value))}
            className="px-3 py-1.5 rounded-md bg-gray-100 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 text-sm"
          >
            <option value={50}>50 行</option>
            <option value={100}>100 行</option>
            <option value={200}>200 行</option>
            <option value={500}>500 行</option>
          </select>
          <button
            onClick={() => refetch()}
            className="p-2 rounded-md hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
            title="刷新"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
        </div>
      </div>

      <div className="flex-1 bg-gray-900 rounded-lg overflow-hidden">
        <div className="h-full overflow-auto p-4 font-mono text-sm">
          {isLoading ? (
            <div className="text-gray-500">加载中...</div>
          ) : logs && logs.length > 0 ? (
            <div className="space-y-1">
              {logs.map((log, i) => (
                <div key={i} className="flex gap-2 hover:bg-gray-800/50 px-2 py-0.5 rounded">
                  <span className="text-gray-500 shrink-0">{log.time}</span>
                  <span className={cn("shrink-0 w-14", getLevelColor(log.level))}>
                    [{log.level}]
                  </span>
                  <span className="text-gray-300 break-all">{log.message}</span>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-gray-500">暂无日志</div>
          )}
        </div>
      </div>
    </div>
  );
}
