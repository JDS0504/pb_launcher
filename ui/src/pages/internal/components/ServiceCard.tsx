import { type FC } from "react";
import type { ServiceDto } from "../../../services/services";
import { Eye, ExternalLink, Play, Square } from "lucide-react";
import classNames from "classnames";
import type { ProxyConfigsResponse } from "../../../services/config";
import { useServiceUrls } from "../../../hooks/useServiceUrls";

type Props = {
  proxyInfo: ProxyConfigsResponse;
  service: ServiceDto;
  onDetails: () => void;
  onStart: () => void;
  onStop: () => void;
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
}) => {
  const serviceUrls = useServiceUrls(service, proxyInfo);
  const adminUrl = serviceUrls.length > 0 ? serviceUrls[0] : null;

  const isRunning = service.status === "running" || service.status === "sleeping";
  const isPending = service.status === "pending";
  const isStopped = service.status === "stopped" || service.status === "idle";

  const statusLabel = STATUS_LABELS[service.status] ?? service.status;

  return (
    <div className="card bg-base-100 shadow border border-base-300">
      <div className="card-body gap-3">
        {/* Header: nombre + badge de estado */}
        <div className="flex items-start justify-between gap-2">
          <h2
            className="card-title text-base-content text-base leading-tight truncate"
            title={service.name}
          >
            {service.name}
          </h2>
          <span
            className={classNames("badge badge-sm shrink-0", {
              "badge-success": service.status === "running",
              "badge-info": service.status === "sleeping",
              "badge-warning": service.status === "pending" || service.status === "idle",
              "badge-error": service.status === "failure",
              "badge-neutral": !["running", "pending", "idle", "failure", "sleeping"].includes(
                service.status,
              ),
            })}
          >
            {statusLabel}
          </span>
        </div>

        {/* Info */}
        <div className="text-xs text-base-content/60 space-y-0.5">
          <div>
            {service.repository} v{service.release_version}
          </div>
          <div className="capitalize">Restart: {service.restart_policy}</div>
        </div>

        {/* Acciones */}
        <div className="card-actions mt-1 flex gap-2 w-full">
          {/* Start / Stop */}
          {isStopped && (
            <button
              id={`btn-start-${service.id}`}
              className="btn btn-xs btn-success gap-1 flex-1"
              onClick={onStart}
              disabled={isPending}
              title="Iniciar"
            >
              <Play className="w-3 h-3" />
              Start
            </button>
          )}
          {isRunning && (
            <button
              id={`btn-stop-${service.id}`}
              className="btn btn-xs btn-error gap-1 flex-1"
              onClick={onStop}
              disabled={isPending}
              title="Detener"
            >
              <Square className="w-3 h-3" />
              Stop
            </button>
          )}
          {isPending && (
            <button className="btn btn-xs flex-1" disabled>
              <span className="loading loading-spinner loading-xs" />
              Pending
            </button>
          )}

          {/* Details */}
          <button
            id={`btn-details-${service.id}`}
            className="btn btn-xs btn-neutral gap-1 flex-1"
            onClick={onDetails}
            title="Ver detalle"
          >
            <Eye className="w-3 h-3" />
            Details
          </button>

          {/* Abrir Admin */}
          {adminUrl && (
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
