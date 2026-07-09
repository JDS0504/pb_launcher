import { type FC, useMemo } from "react";
import type { ServiceDto } from "../../../services/services";
import { Eye, ExternalLink, Play, Square, Copy } from "lucide-react";
import classNames from "classnames";
import type { ProxyConfigsResponse } from "../../../services/config";
import { useServiceUrls } from "../../../hooks/useServiceUrls";

type Props = {
  proxyInfo: ProxyConfigsResponse;
  service: ServiceDto;
  onDetails: () => void;
  onStart: () => void;
  onStop: () => void;
  onClone: () => void;
};

const STATUS_LABELS: Record<string, string> = {
  running: "Running",
  sleeping: "Sleeping",
  pending: "Pending",
  idle: "Idle",
  failure: "Failure",
  stopped: "Stopped",
  restoring: "Restoring",
};

export const ServiceCard: FC<Props> = ({
  proxyInfo,
  service,
  onDetails,
  onStart,
  onStop,
  onClone,
}) => {
  const serviceUrls = useServiceUrls(service, proxyInfo);
  const adminUrl = serviceUrls.length > 0 ? serviceUrls[0] : null;

  const hasPendingCert = useMemo(() => {
    return (service.domains ?? []).some(
      d => d.use_https === "yes" && (d.cert_status === "pending" || d.x_cert_request_state === "pending")
    );
  }, [service.domains]);

  const isRunning = (service.status === "running" || service.status === "sleeping") && !hasPendingCert;
  const isPending = service.status === "pending" || ((service.status === "running" || service.status === "sleeping") && hasPendingCert);
  const isStopped = service.status === "stopped" || service.status === "idle";
  const statusLabel = hasPendingCert && (service.status === "running" || service.status === "sleeping")
    ? "Pending SSL"
    : (STATUS_LABELS[service.status] ?? service.status);

  return (
    <div className="card bg-base-100 shadow border border-base-300">
      <div className="card-body gap-3 p-4">

        {/* Fila superior: nombre + badge + botón Start/Stop */}
        <div className="flex items-center justify-between gap-2 min-w-0">
          <h2
            className="card-title text-base-content text-base leading-tight truncate min-w-0"
            title={service.name}
          >
            {service.name}
          </h2>

          <div className="flex items-center gap-1.5 shrink-0">
            {/* Badge de estado */}
            <span
              className={classNames("badge badge-sm", {
                "badge-success": service.status === "running" && !hasPendingCert,
                "badge-info":    service.status === "sleeping" && !hasPendingCert,
                "badge-warning": service.status === "pending" || service.status === "idle" || hasPendingCert,
                "badge-error":   service.status === "failure",
                "badge-neutral": !["running", "pending", "idle", "failure", "sleeping"].includes(service.status) && !hasPendingCert,
              })}
            >
              {statusLabel}
            </span>

            {/* Clonar (Antes del control de arranque) */}
            <button
              id={`btn-clone-${service.id}`}
              className={classNames("btn btn-xs", {
                "btn-ghost text-base-content/30 cursor-not-allowed": !isStopped,
                "btn-neutral": isStopped,
              })}
              onClick={isStopped ? onClone : undefined}
              disabled={!isStopped}
              title={!isStopped ? "Detén el servicio para clonarlo" : "Clonar instancia"}
            >
              <Copy className="w-3 h-3" />
            </button>

            {/* Start / Stop al lado del badge */}
            {isStopped && (
              <button
                id={`btn-start-${service.id}`}
                className="btn btn-xs btn-success"
                onClick={onStart}
                title="Iniciar"
              >
                <Play className="w-3 h-3" />
              </button>
            )}
            {isRunning && (
              <button
                id={`btn-stop-${service.id}`}
                className="btn btn-xs btn-error"
                onClick={onStop}
                title="Detener"
              >
                <Square className="w-3 h-3" />
              </button>
            )}
            {isPending && (
              <span className="loading loading-spinner loading-xs text-warning" />
            )}
          </div>
        </div>

        {/* Info secundaria */}
        <div className="text-xs text-base-content/60 space-y-0.5">
          <div>{service.repository} v{service.release_version}</div>
          <div className="capitalize">Restart: {service.restart_policy}</div>
        </div>

        {/* Acciones de navegación */}
        <div className="card-actions mt-1 flex gap-2 w-full">
          <button
            id={`btn-details-${service.id}`}
            className="btn btn-xs btn-neutral gap-1 flex-1"
            onClick={onDetails}
          >
            <Eye className="w-3 h-3" />
            Details
          </button>

          {adminUrl && isRunning && (
            <a
              id={`btn-open-admin-${service.id}`}
              href={adminUrl}
              target="_blank"
              rel="noreferrer"
              className="btn btn-xs btn-primary gap-1 flex-1"
              title="Abrir PocketBase Admin"
            >
              <ExternalLink className="w-3 h-3" />
              Admin
            </a>
          )}
        </div>
      </div>
    </div>
  );
};
