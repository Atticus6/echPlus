import { queryOptions } from "@tanstack/react-query";
import { NodeService } from "../../bindings/github.com/atticus6/echPlus/apps/desktop/services";

export const nodesQueryOptions = () =>
  queryOptions({
    queryKey: ["nodes"],
    queryFn: () => NodeService.GetNodes(),
  });
