import { Button } from "@/components/ui/button";
import {
  NodeService,
  ConfigService,
  ProxyServerDesktop,
} from "../../bindings/github.com/atticus6/echPlus/apps/desktop/services";
import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { nodesQueryOptions } from "@/querys/nodes";
import {
  useSuspenseQuery,
  useQueryClient,
  useMutation,
} from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";

import { Check, ChevronsUpDown, CirclePlus, Plus } from "lucide-react";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { cn } from "@/lib/utils";
import { ButtonGroup } from "@/components/ui/button-group";
import { configOptions } from "@/querys/config";
import { ConfigType } from "bindings/github.com/atticus6/echPlus/apps/desktop/config/models";
import { RoutingMode } from "../../bindings/github.com/atticus6/echPlus/apps/client/core/models";
import { isRunningoptions } from "@/querys/proxy";
import { TrafficStats } from "@/components/TrafficStats";

const formSchema = z.object({
  name: z.string().min(1, "名称不能为空"),
  token: z.string().min(1, "Token不能为空"),
  address: z.string().min(1, "地址不能为空"),
  serverIP: z.string(),
  port: z.number().min(1).max(65535),
});

type FormValues = z.infer<typeof formSchema>;

export const Route = createFileRoute("/")({
  component: RouteComponent,
  loader: ({ context: { queryClient } }) => {
    return Promise.all([
      queryClient.ensureQueryData(nodesQueryOptions()),
      queryClient.ensureQueryData(configOptions()),
      queryClient.ensureQueryData(isRunningoptions()),
    ]);
  },
});

function RouteComponent() {
  const { data: nodes } = useSuspenseQuery(nodesQueryOptions());
  const { data: config } = useSuspenseQuery(configOptions());
  const { data: isRunning } = useSuspenseQuery(isRunningoptions());
  console.log(config);

  const queryClient = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);

  const [open, setOpen] = useState(false);

  const { mutate: ChangeConfig } = useMutation({
    mutationKey: ["config", "ChangeValue"],
    mutationFn: (v: Partial<ConfigType>) => {
      return ConfigService.ChangeValue(v as any);
    },
    onSuccess() {
      queryClient.invalidateQueries({ queryKey: configOptions().queryKey });
    },
  });

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
      token: "",
      address: "",
      serverIP: "",
      port: 443,
    },
  });

  const onSubmit = async (values: FormValues) => {
    try {
      await NodeService.CreateNode(
        values.name,
        values.token,
        values.address,
        values.serverIP || "",
        values.port
      );
      setShowCreate(false);
      form.reset();
      queryClient.invalidateQueries({ queryKey: ["nodes"] });
    } catch (error) {
      console.error("创建节点失败:", error);
    }
  };

  return (
    <div className="h-full flex justify-center items-center">
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>创建节点</DialogTitle>
            <DialogDescription>
              填写节点信息，点击创建完成添加。
            </DialogDescription>
          </DialogHeader>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>名称</FormLabel>
                    <FormControl>
                      <Input placeholder="节点名称" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="token"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Token</FormLabel>
                    <FormControl>
                      <Input placeholder="访问令牌" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="address"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>地址</FormLabel>
                    <FormControl>
                      <Input placeholder="例如: ech.example.com" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="serverIP"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>服务器IP</FormLabel>
                    <FormControl>
                      <Input placeholder="可选" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="port"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>端口</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        placeholder="443"
                        {...field}
                        onChange={(e) => field.onChange(Number(e.target.value))}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <DialogFooter>
                <DialogClose asChild>
                  <Button type="button" variant="outline">
                    取消
                  </Button>
                </DialogClose>
                <Button type="submit">创建</Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>

      {!nodes.length ? (
        <Button
          size="icon"
          className="rounded-full"
          onClick={() => setShowCreate(true)}
        >
          <CirclePlus />
        </Button>
      ) : (
        <div className="flex flex-col items-center gap-6">
          <Switch
            checked={isRunning}
            onCheckedChange={async (v) => {
              if (v) {
                await ProxyServerDesktop.Start();
              } else {
                await ProxyServerDesktop.Stop();
              }
              queryClient.invalidateQueries({
                queryKey: isRunningoptions().queryKey,
              });
            }}
          />
          
          {/* 流量统计 */}
          {isRunning && <TrafficStats />}
          
          <div className="flex gap-1 p-1 bg-gray-100 dark:bg-gray-800 rounded-lg">
            {[
              { value: RoutingMode.RoutingModeGlobal, label: "全局" },
              { value: RoutingMode.RoutingModeBypassCN, label: "中国大陆" },
              { value: RoutingMode.RoutingModeNone, label: "直连" },
            ].map((mode) => (
              <button
                key={mode.value}
                onClick={() => ChangeConfig({ RoutingMode: mode.value })}
                className={cn(
                  "px-4 py-2 text-sm font-medium rounded-md transition-all duration-200",
                  config.RoutingMode === mode.value
                    ? "bg-white dark:bg-gray-700 text-gray-900 dark:text-white shadow-sm"
                    : "text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300"
                )}
              >
                {mode.label}
              </button>
            ))}
          </div>
          <div className="flex gap-2">
            <Popover open={open} onOpenChange={setOpen}>
              <ButtonGroup>
                <PopoverTrigger asChild>
                  <Button
                    variant="outline"
                    role="combobox"
                    aria-expanded={open}
                    className="w-[200px] justify-between"
                  >
                    {nodes.find((item) => item.id === config.SelectNodeId)
                      ?.name || "请选择节点"}
                    <ChevronsUpDown className="opacity-50" />
                  </Button>
                </PopoverTrigger>
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => {
                    setShowCreate(true);
                  }}
                >
                  <Plus />
                </Button>
              </ButtonGroup>

              <PopoverContent className="w-[200px] p-0">
                <Command>
                  <CommandInput
                    placeholder="Search framework..."
                    className="h-9"
                  />
                  <CommandList>
                    <CommandEmpty>No framework found.</CommandEmpty>
                    <CommandGroup>
                      {nodes.map((node) => (
                        <CommandItem
                          key={node.id}
                          value={String(node.id)}
                          onSelect={(currentValue) => {
                            ChangeConfig({
                              SelectNodeId: node.id,
                            });

                            // setValue(currentValue === value ? "" : currentValue);
                            setOpen(false);
                          }}
                        >
                          {node.name}
                          <Check
                            className={cn(
                              "ml-auto",
                              config.SelectNodeId === node.id
                                ? "opacity-100"
                                : "opacity-0"
                            )}
                          />
                        </CommandItem>
                      ))}
                    </CommandGroup>
                  </CommandList>
                </Command>
              </PopoverContent>
            </Popover>
          </div>
        </div>
      )}
    </div>
  );
}
