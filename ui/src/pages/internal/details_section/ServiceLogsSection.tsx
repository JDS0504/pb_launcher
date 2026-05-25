import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState, type FC } from "react";
import { serviceService, type ServiceLog, type ServiceDto } from "../../../services/services";
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

  const commandMutation = useMutation({
    mutationFn: serviceService.executeServiceCommand,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["services", service_id] });
      queryClient.invalidateQueries({ queryKey: ["services"] });
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

  if (service == null) {
    return <div className="p-4">Loading...</div>;
  }

  const status = service.status;
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
      <LogsView service_id={service_id} />
    </div>
  );
};

type LogsViewProps = {
  service_id: string;
};

const LogsView: FC<LogsViewProps> = ({ service_id }) => {
  const viewHeight = useViewportHeight();
  const [logs, setLogs] = useState<ServiceLog[]>([]);
  const [connected, setConnectionState] = useState<boolean>(true);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const controller = new AbortController();
    let isActive = true;
    let timeoutId: NodeJS.Timeout;

    // Carga inicial de logs históricos
    serviceService
      .fetchServiceLogs(controller.signal, service_id, -1)
      .then(initLogs => {
        if (!isActive) return;
        const safeInitLogs = Array.isArray(initLogs) ? initLogs : [];
        setLogs(safeInitLogs);
        if (safeInitLogs.length > 0) {
          setTimeout(() => {
            if (containerRef.current) {
              containerRef.current.scrollTop = containerRef.current.scrollHeight;
            }
          }, 50);
        }
      })
      .catch(() => {
        // ignorar error de abort en carga inicial
      });

    const fetchLoop = async () => {
      try {
        const newLogs = await serviceService.fetchServiceLogs(
          controller.signal,
          service_id,
          10,
        );

        // Verificar isActive antes de cualquier setState
        if (!isActive) return;

        const safeLogs = Array.isArray(newLogs) ? newLogs : [];
        let shouldScrollToBottom = false;

        if (containerRef.current) {
          const { scrollTop, clientHeight, scrollHeight } = containerRef.current;
          // Si está a menos de 30px del fondo, asumimos que quiere auto-scroll
          shouldScrollToBottom = scrollTop + clientHeight >= scrollHeight - 30;
        }

        setLogs(prev => {
          const merged = mergeLogsUnique(prev, safeLogs);
          // Si antes estaba vacío y ahora tiene elementos, forzar auto-scroll
          if (prev.length === 0 && merged.length > 0) {
            shouldScrollToBottom = true;
          }
          return merged;
        });

        if (shouldScrollToBottom) {
          setTimeout(() => {
            if (containerRef.current) {
              containerRef.current.scrollTop = containerRef.current.scrollHeight;
            }
          }, 50);
        }
        setConnectionState(true);
      } catch (err) {
        // Si el componente ya se desmontó o el signal fue abortado, ignorar
        if (!isActive) return;
        // Solo loguear errores reales, no aborts
        if (err instanceof Error && err.name !== "AbortError") {
          console.warn("Logs fetch error:", err);
        }
        setConnectionState(false);
      }

      if (isActive) {
        timeoutId = setTimeout(fetchLoop, 1000);
      }
    };

    // Primera iteración con delay
    timeoutId = setTimeout(fetchLoop, 2000);

    return () => {
      isActive = false;
      controller.abort();
      clearTimeout(timeoutId);
    };
  }, [service_id]);

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
          {logs.map(log => (
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
