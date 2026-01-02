import { queryOptions } from "@tanstack/react-query";
import { ProxyServerDesktop } from "../../bindings/github.com/atticus6/echPlus/apps/desktop/services";
export const isRunningoptions = () =>
  queryOptions({
    queryKey: ["isRunning"],
    queryFn: () => ProxyServerDesktop.IsRunning(),
  });

export const trafficStatsOptions = () =>
  queryOptions({
    queryKey: ["trafficStats"],
    queryFn: () => ProxyServerDesktop.GetTrafficStats(),
    refetchInterval: 1000, // 每1秒刷新
  });
