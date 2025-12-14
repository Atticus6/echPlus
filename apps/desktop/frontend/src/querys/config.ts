import { queryOptions } from "@tanstack/react-query";
import { ConfigService } from "../../bindings/github.com/atticus6/echPlus/apps/desktop/services";

export const configOptions = () =>
  queryOptions({
    queryKey: ["config"],
    queryFn: () => ConfigService.GetValue(),
  });
