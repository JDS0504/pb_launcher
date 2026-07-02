import { useQueryClient } from "@tanstack/react-query";
import { type FC } from "react";
import { useNavigate } from "react-router-dom";
import classNames from "classnames";
import { ServiceForm } from "../forms/ServiceForm";
import type { ServiceDto } from "../../../services/services";
import { useModal } from "../../../components/modal/hook";
import { ChangePasswordModal } from "../components/ChangePasswordModal";
import {
  Play,
  Square,
  RotateCcw,
  Copy,
  HardDrive,
  KeyRound,
  Trash2,
} from "lucide-react";
import { useServiceActions } from "../../../hooks/useServiceActions";

type Props = {
  service_id: string;
  service?: ServiceDto;
};

export const GeneralSection: FC<Props> = ({ service_id, service }) => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { openModal } = useModal();

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ["services"] });
  };

  const {
    handleStart,
    handleStop,
    handleRestart,
    handleBackup,
    handleClone,
    isCommandPending,
    isBackupPending,
  } = useServiceActions(invalidate);

  const { handleDelete } = useServiceActions(() => navigate("/"));

  if (service == null) {
    return <div className="p-4">Loading...</div>;
  }

  const statusLabel = service.status.charAt(0).toUpperCase() + service.status.slice(1);
  const isRunning  = service.status === "running" || service.status === "sleeping";
  const isStopped  = service.status === "stopped" || service.status === "idle" || service.status === "failure";
  const isPending  = service.status === "pending";

  const openChangePassword = () => {
    openModal(
      <ChangePasswordModal service_id={service_id} />,
      { title: "Cambiar contraseña del superuser" },
    );
  };

  // Navega a la nueva URL si el nombre cambió, o solo invalida si no cambió
  const handleSaveRecord = (newName?: string) => {
    queryClient.invalidateQueries({ queryKey: ["services"] });
    if (newName && newName !== service.name) {
      navigate(`/services/${newName}?section=general`);
    }
  };

  return (
    <div className="space-y-6">

      {/* ── Fila de estado + controles ─────────────────────────────────── */}
      <div className="flex items-center justify-between flex-wrap gap-3">

        {/* Izquierda: badge + versión */}
        <div className="flex items-center gap-2 flex-wrap">
          <span
            className={classNames("badge badge-sm", {
              "badge-success": service.status === "running",
              "badge-info":    service.status === "sleeping",
              "badge-warning": service.status === "pending" || service.status === "idle",
              "badge-error":   service.status === "failure",
              "badge-neutral": !["running", "pending", "idle", "failure", "sleeping"].includes(service.status),
            })}
          >
            {statusLabel}
          </span>
          {service.release_version && (
            <span className="badge badge-ghost badge-sm font-mono">
              v{service.release_version}
            </span>
          )}
          {service.repository && (
            <span className="text-xs text-base-content/50">{service.repository}</span>
          )}
          {isPending && (
            <span className="loading loading-spinner loading-xs text-warning" />
          )}
        </div>

        {/* Derecha: controles de proceso + acciones secundarias */}
        <div className="flex items-center gap-2 flex-wrap">
          {/* Start / Stop / Restart — solo los relevantes */}
          {isStopped && (
            <button
              id="btn-service-start"
              className="btn btn-xs btn-success gap-1"
              disabled={isCommandPending}
              onClick={() => handleStart(service_id)}
            >
              <Play className="w-3.5 h-3.5" />
              Start
            </button>
          )}
          {isRunning && (
            <>
              <button
                id="btn-service-restart"
                className="btn btn-xs btn-warning gap-1"
                disabled={isCommandPending}
                onClick={() => handleRestart(service_id)}
              >
                <RotateCcw className="w-3.5 h-3.5" />
                Restart
              </button>
              <button
                id="btn-service-stop"
                className="btn btn-xs btn-error gap-1"
                disabled={isCommandPending}
                onClick={() => handleStop(service_id)}
              >
                <Square className="w-3.5 h-3.5" />
                Stop
              </button>
            </>
          )}

          {/* Separador */}
          <div className="w-px h-5 bg-base-300 hidden sm:block" />

          {/* Acciones secundarias: siempre visibles, deshabilitadas si running */}
          <button
            id="btn-service-clone"
            className="btn btn-xs btn-ghost gap-1"
            disabled={!isStopped}
            onClick={() => handleClone(service)}
            title={isStopped ? "Clonar instancia" : "Detén el servicio para clonar"}
          >
            <Copy className="w-3.5 h-3.5" />
            Clonar
          </button>
          <button
            id="btn-service-backup"
            className="btn btn-xs btn-ghost gap-1"
            disabled={!isStopped || isBackupPending}
            onClick={() => handleBackup(service_id)}
            title={isStopped ? "Descargar backup" : "Detén el servicio para hacer backup"}
          >
            {isBackupPending
              ? <span className="loading loading-spinner loading-xs" />
              : <HardDrive className="w-3.5 h-3.5" />
            }
            Backup
          </button>
          <button
            id="btn-change-password"
            className="btn btn-xs btn-ghost gap-1"
            onClick={openChangePassword}
            title="Cambiar contraseña del superusuario"
          >
            <KeyRound className="w-3.5 h-3.5" />
            Password
          </button>
        </div>
      </div>

      {/* ── Error message ──────────────────────────────────────────────── */}
      {service.error_message && (
        <div className="alert alert-error text-xs p-3">
          {service.error_message}
        </div>
      )}

      {/* ── Aviso de edición ───────────────────────────────────────────── */}
      {isRunning && (
        <div className="text-xs text-base-content/50 italic">
          Detén el servicio para editar el nombre o la versión.
        </div>
      )}

      {/* ── Formulario de configuración ────────────────────────────────── */}
      <ServiceForm
        record={service}
        canChangeVersion={isStopped}
        onSaveRecord={handleSaveRecord}
      />

      {/* ── Zona peligrosa ───────────────────────────────────────────────── */}
      <div className="rounded-box border border-error/30 bg-base-100 p-4 space-y-3 mt-4">
        <h4 className="font-semibold text-sm text-error">Zona peligrosa</h4>
        <p className="text-xs text-base-content/60">
          Eliminar el servicio borrará permanentemente la instancia y todos sus datos del servidor.
          Esta acción no se puede deshacer.
        </p>
        <button
          id="btn-service-delete"
          className="btn btn-sm btn-error btn-outline gap-2"
          onClick={() => handleDelete(service_id)}
        >
          <Trash2 className="w-4 h-4" />
          Eliminar servicio
        </button>
      </div>
    </div>
  );
};
