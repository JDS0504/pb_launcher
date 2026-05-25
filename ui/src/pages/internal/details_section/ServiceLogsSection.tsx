import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState, type FC } from "react";
import { serviceService, type ServiceLog, type ServiceDto } from "../../../services/services";
import classNames from "classnames";
import { Play, RotateCw, Square, Download } from "lucide-react";
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
    <div className="space-y-4 min-w-0">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div className="text-sm text-base-content/70">
          Status: <span className="font-medium capitalize">{status}</span>
        </div>
        <div className="flex flex-wrap gap-2">
          <div
            title={
              isRunning || isPending
                ? "El servicio ya está en ejecución"
                : undefined
            }
          >
            <button
              className="btn btn-sm btn-success gap-2"
              disabled={isRunning || isPending || commandMutation.isPending}
              onClick={() => executeCommand("start")}
            >
              <Play className="w-4 h-4" />
              Start
            </button>
          </div>
          <div
            title={
              !isRunning ? "El servicio no está en ejecución" : undefined
            }
          >
            <button
              className="btn btn-sm btn-warning gap-2"
              disabled={!isRunning || commandMutation.isPending}
              onClick={() => executeCommand("restart")}
            >
              <RotateCw className="w-4 h-4" />
              Restart
            </button>
          </div>
          <div
            title={
              !isRunning ? "El servicio no está en ejecución" : undefined
            }
          >
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
      </div>
      <LogsView service_id={service_id} />
    </div>
  );
};

type LogsViewProps = {
  service_id: string;
};

type StreamFilter = "all" | "stdout" | "stderr";

const LogsView: FC<LogsViewProps> = ({ service_id }) => {
  const [logs, setLogs] = useState<ServiceLog[]>([]);
  const [connected, setConnectionState] = useState<boolean>(true);
  const [streamFilter, setStreamFilter] = useState<StreamFilter>("all");
  const containerRef = useRef<HTMLDivElement>(null);
  const shouldScrollToBottomRef = useRef<boolean>(true);

  useEffect(() => {
    if (containerRef.current && shouldScrollToBottomRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [logs]);

  const handleScroll = () => {
    if (!containerRef.current) return;
    const { scrollTop, clientHeight, scrollHeight } = containerRef.current;
    shouldScrollToBottomRef.current = scrollTop + clientHeight >= scrollHeight - 30;
  };

  useEffect(() => {
    const controller = new AbortController();
    let isActive = true;
    let timeoutId: NodeJS.Timeout;

    serviceService
      .fetchServiceLogs(controller.signal, service_id, -1)
      .then(initLogs => {
        if (!isActive) return;
        const safeInitLogs = Array.isArray(initLogs) ? initLogs : [];
        shouldScrollToBottomRef.current = true;
        setLogs(safeInitLogs);
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

        if (!isActive) return;

        const safeLogs = Array.isArray(newLogs) ? newLogs : [];

        if (containerRef.current) {
          const { scrollTop, clientHeight, scrollHeight } = containerRef.current;
          shouldScrollToBottomRef.current = scrollTop + clientHeight >= scrollHeight - 30;
        }

        setLogs(prev => {
          const merged = mergeLogsUnique(prev, safeLogs);
          if (prev.length === 0 && merged.length > 0) {
            shouldScrollToBottomRef.current = true;
          }
          return merged;
        });

        setConnectionState(true);
      } catch (err) {
        if (!isActive) return;
        if (err instanceof Error && err.name !== "AbortError") {
          console.warn("Logs fetch error:", err);
        }
        setConnectionState(false);
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

  const filteredLogs = streamFilter === "all"
    ? logs
    : logs.filter(l => l.stream === streamFilter);

  const handleExport = () => {
    const text = logs
      .map(l => `[${l.stream.toUpperCase()}] ${l.message}`)
      .join("\n");
    const blob = new Blob([text], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `logs-${service_id}.txt`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="space-y-2 min-w-0">
      {/* Controles de filtro y exportar */}
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex gap-1">
          {(["all", "stdout", "stderr"] as StreamFilter[]).map(f => (
            <button
              key={f}
              className={classNames("btn btn-xs", {
                "btn-neutral": streamFilter === f,
                "btn-ghost": streamFilter !== f,
              })}
              onClick={() => setStreamFilter(f)}
            >
              {f === "all" ? "Todos" : f}
            </button>
          ))}
        </div>
        <div className="flex items-center gap-2">
          <div
            className={classNames("w-2 h-2 rounded-full", {
              "bg-green-500": connected,
              "bg-red-500": !connected,
            })}
            title={connected ? "Conectado" : "Sin conexión"}
          />
          <button
            className="btn btn-xs btn-ghost gap-1"
            onClick={handleExport}
            disabled={logs.length === 0}
            title="Descargar logs como .txt"
          >
            <Download className="w-3 h-3" />
            Exportar
          </button>
        </div>
      </div>

      {/* Contenedor de logs con altura responsiva */}
      <div
        className="text-base-content overflow-y-auto font-mono text-sm bg-base-300/40 rounded-lg p-3 min-h-[200px] max-h-[calc(100vh-340px)]"
        ref={containerRef}
        onScroll={handleScroll}
      >
        <div className="whitespace-pre-wrap space-y-1">
          {filteredLogs.length === 0 ? (
            <span className="text-base-content/40 text-xs">
              {logs.length === 0 ? "Sin logs disponibles..." : "Sin logs para el filtro seleccionado."}
            </span>
          ) : (
            filteredLogs.map(log => (
              <div
                key={log.id}
                className={
                  log.stream === "stderr" ? "text-error" : "text-success"
                }
              >
                {log.message}
              </div>
            ))
          )}
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
