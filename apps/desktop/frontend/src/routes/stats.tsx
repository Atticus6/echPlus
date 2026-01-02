import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { trafficStatsOptions } from "@/querys/proxy";
import { ArrowUp, ArrowDown } from "lucide-react";

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

function formatSpeed(bytesPerSec: number): string {
  return formatBytes(bytesPerSec) + "/s";
}

export const Route = createFileRoute("/stats")({
  component: StatsPage,
});

function StatsPage() {
  const { data: stats } = useQuery(trafficStatsOptions());

  return (
    <div className="p-6 h-full flex flex-col">
      <h1 className="text-xl font-semibold mb-6">流量统计</h1>

      {/* 总流量卡片 */}
      <div className="grid grid-cols-2 gap-4 mb-6 shrink-0">
        <div className="bg-green-50 dark:bg-green-900/20 rounded-xl p-4">
          <div className="flex items-center gap-2 text-green-600 dark:text-green-400 mb-2">
            <ArrowUp className="w-5 h-5" />
            <span>上传</span>
          </div>
          <div className="text-2xl font-bold text-green-700 dark:text-green-300">
            {formatBytes(stats?.totalUpload || 0)}
          </div>
          <div className="text-sm text-green-600/70 dark:text-green-400/70 mt-1">
            {formatSpeed(stats?.uploadSpeed || 0)}
          </div>
        </div>
        <div className="bg-blue-50 dark:bg-blue-900/20 rounded-xl p-4">
          <div className="flex items-center gap-2 text-blue-600 dark:text-blue-400 mb-2">
            <ArrowDown className="w-5 h-5" />
            <span>下载</span>
          </div>
          <div className="text-2xl font-bold text-blue-700 dark:text-blue-300">
            {formatBytes(stats?.totalDownload || 0)}
          </div>
          <div className="text-sm text-blue-600/70 dark:text-blue-400/70 mt-1">
            {formatSpeed(stats?.downloadSpeed || 0)}
          </div>
        </div>
      </div>

      {/* 站点列表 */}
      <div className="flex-1 min-h-0 flex flex-col">
        <h2 className="text-sm text-gray-500 dark:text-gray-400 mb-3">
          站点流量排行
        </h2>
        <div className="flex-1 overflow-y-auto bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
          {stats?.sites && stats.sites.length > 0 ? (
            stats.sites.map((site, index) => (
              <div
                key={site.host}
                className="flex items-center justify-between px-4 py-3 border-b border-gray-100 dark:border-gray-700 last:border-0"
              >
                <div className="flex items-center gap-3 min-w-0">
                  <span className="text-sm text-gray-400 w-6">{index + 1}</span>
                  <span className="text-sm text-gray-700 dark:text-gray-300 truncate">
                    {site.host}
                  </span>
                </div>
                <div className="flex items-center gap-4 text-sm shrink-0">
                  <span className="text-green-600 dark:text-green-400">
                    ↑ {formatBytes(site.upload || 0)}
                  </span>
                  <span className="text-blue-600 dark:text-blue-400">
                    ↓ {formatBytes(site.download || 0)}
                  </span>
                  <span className="text-gray-500 w-20 text-right">
                    {formatBytes((site.upload || 0) + (site.download || 0))}
                  </span>
                </div>
              </div>
            ))
          ) : (
            <div className="text-center text-gray-400 py-8">暂无流量数据</div>
          )}
        </div>
      </div>
    </div>
  );
}
