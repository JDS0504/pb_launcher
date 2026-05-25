import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState, type FC } from "react";
import { serviceService, type ServiceLog, type ServiceDto } from "../../../services/services";
import { ErrorFallback } from "../../../components/helpers/ErrorFallback";
import { useViewportHeight } from "../../../hooks/useViewportHeight";
import classNames from "classnames";
import { Play, RotateCw, Square } from "lucide-react";
import toast from "react-hot-toast";
import { getErrorMessage } from "../../../utils/errors";
import { useConfirmModal } from "../../../hooks/useConfirmModal";

type Props = {
  service_id: string;
  service?: ServiceDto;
};

export const ServiceLogsSection: FC<Props> = ({ service_id, service }) => {
  const confirm = useConfirmModal();
  const queryClient = useQueryClient();

  const initLogsQuery = useQuery({
    queryKey: ["services", service_id],
    queryFn: ({ signal }) =>
      serviceService.fetchServiceLogs(signal, service_id, -1),
    refetchOnMount: true,
  });

  const commandMutation = useMutation({
    mutationFn: serviceService.executeServiceCommand,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["services", service_id] });
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const executeCommand = async (action: "start" | "stop" | "restart") => {
    if (action !== "start") {
      const ok = await confirm(
        `${action.charAt(0).toUpperCase()}${action.slice(1)} service`,
        `Are you sure you want to ${action} this service?`,
      );
      if (!ok) return;
    }
    commandMutation.mutate({ service_id, action });
  };

  if (initLogsQuery.isFetching || service == null) {
    return <div className="p-4">Loading...</div>;
  }

  if (initLogsQuery.isError)
    return (
      <ErrorFallback
        error={initLogsQuery.error}
        onRetry={() => setTimeout(initLogsQuery.refetch)}
      />
    );

  const status = service?.status;
  const isRunning = status === "running";
  const isPending = status === "pending";

  return (
    <div className="space-y-4">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-3">
        <div className="text-sm text-base-content/70">
          Status: <span className="font-medium capitalize">{status}</span>
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            className="btn btn-sm btn-success gap-2"
            disabled={isRunning || isPending || commandMutation.isPending}
            onClick={() => executeCommand("start")}
          >
            <Play className="w-4 h-4" />
            Start
          </button>
          <button
            className="btn btn-sm btn-warning gap-2"
            disabled={!isRunning || commandMutation.isPending}
            onClick={() => executeCommand("restart")}
          >
            <RotateCw className="w-4 h-4" />
            Restart
          </button>
          <button
            className="btn btn-sm btn-error gap-2"
            disabled={!isRunning || commandMutation.isPending}
            onClick={() => executeCommand("stop")}
          >
            <Square className="w-4 h-4" />
            Stop
          </button>
        </div>
      </div>
      <LogsView initLogs={initLogsQuery.data ?? []} service_id={service_id} />
    </div>
  );
};

type LogsViewProps = {
  service_id: string;
  initLogs: ServiceLog[];
};

const LogsView: FC<LogsViewProps> = ({ initLogs, service_id }) => {
  const viewHeight = useViewportHeight();
  const [logs, setLogs] = useState<ServiceLog[]>(initLogs);
  const [connected, setConnectionState] = useState<boolean>(true);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const controller = new AbortController();
    let isActive = true;
    let timeoutId: NodeJS.Timeout;

    const fetchLoop = async () => {
      try {
        const newLogs = await serviceService.fetchServiceLogs(
          controller.signal,
          service_id,
          10,
        );
        if (!isActive) return;
        let isScrolledToBottom = false;
        if (containerRef.current) {
          isScrolledToBottom =
            containerRef.current.scrollTop +
              containerRef.current.clientHeight >=
            containerRef.current.scrollHeight - 5;
        }
        setLogs(prev => mergeLogsUnique(prev, newLogs));
        if (isScrolledToBottom) {
          setTimeout(() => {
            if (containerRef.current) {
              containerRef.current.scrollTop =
                containerRef.current.scrollHeight;
            }
          }, 10);
        }
        setConnectionState(true);
      } catch (err) {
        console.log(err);
        setConnectionState(false);
        if (!isActive) return;
      }

      if (isActive) {
        timeoutId = setTimeout(fetchLoop, 1000);
      }
    };
    timeoutId = setTimeout(fetchLoop, 2000);
    return () => {
      isActive = false;
      controller.abort();
      clearTimeout(timeoutId);
    };
  }, [service_id]);

  useEffect(() => {
    if (containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, []);

  return (
    <div style={{ height: viewHeight - 270 }} className="relative">
      <div
        className={classNames("w-4 h-4 rounded-full absolute top-0 right-0", {
          "bg-green-600": connected,
          "bg-red-500": !connected,
        })}
      />
      <div
        style={{ height: viewHeight - 270 }}
        className="text-base-content overflow-y-auto font-mono text-sm"
        ref={containerRef}
      >
        <div className="whitespace-pre-wrap space-y-1">
          {(Array.isArray(logs) ? logs : []).map(log => (
            <div
              key={log.id}
              className={
                log.stream === "stderr" ? "text-error" : "text-success"
              }
            >
              {log.message}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

const mergeLogsUnique = (
  current: ServiceLog[],
  incoming: ServiceLog[],
): ServiceLog[] => {
  const existingIds = new Set(current.map(log => log.id));
  const newLogs = incoming.filter(log => !existingIds.has(log.id));
  return [...current, ...newLogs];
};
