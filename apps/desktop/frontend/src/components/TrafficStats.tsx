import { useQuery } from "@tanstack/react-query";
import { trafficStatsOptions } from "@/querys/proxy";
import { ArrowUp, ArrowDown, ChartBar } from "lucide-react";
import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

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

export function TrafficStats() {
  const { data: stats } = useQuery(trafficStatsOptions());
  const [open, setOpen] = useState(false);

  if (!stats) return null;

  return (
    <>
      {/* 简要流量显示 + 查看按钮 */}
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-4 text-sm">
          <div className="flex items-center gap-1">
            <ArrowUp className="w-4 h-4 text-green-500" />
            <span className="text-gray-600 dark:text-gray-400">
              {formatSpeed(stats.uploadSpeed || 0)}
            </span>
          </div>
          <div className="flex items-center gap-1">
            <ArrowDown className="w-4 h-4 text-blue-500" />
            <span className="text-gray-600 dark:text-gray-400">
              {formatSpeed(stats.downloadSpeed || 0)}
            </span>
          </div>
        </div>

        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button variant="ghost" size="icon" className="h-8 w-8">
              <ChartBar className="w-4 h-4" />
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>流量统计</DialogTitle>
            </DialogHeader>
            <div className="space-y-4">
              {/* 总流量统计 */}
              <div className="grid grid-cols-2 gap-4">
                <div className="bg-green-50 dark:bg-green-900/20 rounded-lg p-3">
                  <div className="flex items-center gap-2 text-green-600 dark:text-green-400 mb-1">
                    <ArrowUp className="w-4 h-4" />
                    <span className="text-sm">上传</span>
                  </div>
                  <div className="text-lg font-semibold text-green-700 dark:text-green-300">
                    {formatBytes(stats.totalUpload || 0)}
                  </div>
                </div>
                <div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-3">
                  <div className="flex items-center gap-2 text-blue-600 dark:text-blue-400 mb-1">
                    <ArrowDown className="w-4 h-4" />
                    <span className="text-sm">下载</span>
                  </div>
                  <div className="text-lg font-semibold text-blue-700 dark:text-blue-300">
                    {formatBytes(stats.totalDownload || 0)}
                  </div>
                </div>
              </div>

              {/* 站点列表 */}
              {stats.sites && stats.sites.length > 0 && (
                <div>
                  <div className="text-sm text-gray-500 dark:text-gray-400 mb-2">
                    站点流量排行
                  </div>
                  <div className="bg-gray-50 dark:bg-gray-800/50 rounded-lg max-h-64 overflow-y-auto">
                    {stats.sites.map((site, index) => (
                      <div
                        key={site.host}
                        className="flex items-center justify-between px-3 py-2 border-b border-gray-100 dark:border-gray-700 last:border-0"
                      >
                        <div className="flex items-center gap-2 min-w-0">
                          <span className="text-xs text-gray-400 w-5">
                            {index + 1}
                          </span>
                          <span className="text-sm text-gray-700 dark:text-gray-300 truncate">
                            {site.host}
                          </span>
                        </div>
                        <div className="flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400 shrink-0">
                          <span className="text-green-600 dark:text-green-400">
                            ↑{formatBytes(site.upload || 0)}
                          </span>
                          <span className="text-blue-600 dark:text-blue-400">
                            ↓{formatBytes(site.download || 0)}
                          </span>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {(!stats.sites || stats.sites.length === 0) && (
                <div className="text-center text-gray-400 py-4">
                  暂无流量数据
                </div>
              )}
            </div>
          </DialogContent>
        </Dialog>
      </div>
    </>
  );
}
