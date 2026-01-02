import { cn } from "@/lib/utils";
import { Home, ChartBar, Server, Settings } from "lucide-react";
import { Link, useLocation } from "@tanstack/react-router";

const menuItems = [
  { icon: Home, label: "首页", path: "/" },
  { icon: ChartBar, label: "流量统计", path: "/stats" },
  { icon: Server, label: "节点管理", path: "/nodes" },
  { icon: Settings, label: "设置", path: "/settings" },
];

export function Sidebar() {
  const location = useLocation();

  return (
    <div className="w-16 h-full bg-gray-50 dark:bg-gray-900 border-r border-gray-200 dark:border-gray-800 flex flex-col items-center py-4 gap-2">
      {/* 拖拽区域 */}
      <div className="h-8 w-full" style={{ WebkitAppRegion: "drag" } as React.CSSProperties} />
      
      {menuItems.map((item) => {
        const isActive = location.pathname === item.path;
        return (
          <Link
            key={item.path}
            to={item.path}
            className={cn(
              "w-10 h-10 flex items-center justify-center rounded-lg transition-colors",
              isActive
                ? "bg-blue-100 dark:bg-blue-900/50 text-blue-600 dark:text-blue-400"
                : "text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800"
            )}
            title={item.label}
          >
            <item.icon className="w-5 h-5" />
          </Link>
        );
      })}
    </div>
  );
}
