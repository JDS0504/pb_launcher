import { useQueryClient } from "@tanstack/react-query";
import { type FC } from "react";
import classNames from "classnames";
import { ServiceForm } from "../forms/ServiceForm";
import type { ServiceDto } from "../../../services/services";
import { useModal } from "../../../components/modal/hook";
import { ChangePasswordModal } from "../components/ChangePasswordModal";
import {
  Play,
  Square,
  RotateCcw,
  Cpu,
  Copy,
  HardDrive,
  Trash2,
  KeyRound,
} from "lucide-react";
import { useServiceActions } from "../../../hooks/useServiceActions";

type Props = {
  service_id: string;
  service?: ServiceDto;
};

export const GeneralSection: FC<Props> = ({ service_id, service }) => {
  const queryClient = useQueryClient();
  const { openModal } = useModal();

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ["services", service?.name] });
    queryClient.invalidateQueries({ queryKey: ["services"] });
  };

  const {
    handleStart,
    handleStop,
    handleRestart,
    handleDelete,
    handleBackup,
    handleChangeVersion,
    handleClone,
    isCommandPending,
    isBackupPending,
  } = useServiceActions(invalidate);

  if (service == null) {
    return <div className="p-4">Loading...</div>;
  }

  const status = service.status.charAt(0).toUpperCase() + service.status.slice(1);
  const isRunning = service.status === "running" || service.status === "sleeping";
  const isStopped = service.status === "stopped" || service.status === "idle";
  const isPending = service.status === "pending";
  const canEditConfig = isStopped;

  const openChangePassword = () => {
    openModal(
      <ChangePasswordModal service_id={service_id} />,
      { title: "Cambiar contraseña del superuser" },
    );
  };

  return (
    <div className="space-y-6">
      {/* ── Estado y controles de proceso ─────────────────────────── */}
      <div className="rounded-box bg-base-200 p-4 space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span
              className={classNames("badge badge-sm", {
                "badge-success": service.status === "running",
                "badge-info": service.status === "sleeping",
                "badge-warning": service.status === "pending" || service.status === "idle",
                "badge-error": service.status === "failure",
                "badge-neutral": !["running", "pending", "idle", "failure", "sleeping"].includes(
                  service.status,
                ),
              })}
            >
              {status}
            </span>
            {service.release_version && (
              <span className="badge badge-ghost badge-sm font-mono">
                v{service.release_version}
              </span>
            )}
            {service.release_version && (
              <span className="text-xs text-base-content/50">{service.repository}</span>
            )}
          </div>
        </div>

        {/* Controles Start / Stop / Restart */}
        <div className="flex flex-wrap gap-2">
          <button
            id="btn-service-start"
            className="btn btn-sm btn-success gap-2"
            disabled={isRunning || isPending}
            onClick={() => handleStart(service_id)}
          >
            <Play className="w-4 h-4" />
            Start
          </button>
          <button
            id="btn-service-restart"
            className="btn btn-sm btn-warning gap-2"
            disabled={!isRunning || isPending}
            onClick={() => handleRestart(service_id)}
          >
            <RotateCcw className="w-4 h-4" />
            Restart
          </button>
          <button
            id="btn-service-stop"
            className="btn btn-sm btn-error gap-2"
            disabled={!isRunning || isPending}
            onClick={() => handleStop(service_id)}
          >
            <Square className="w-4 h-4" />
            Stop
          </button>
          {isPending && (
            <span className="flex items-center gap-1 text-xs text-base-content/50">
              <span className="loading loading-spinner loading-xs" />
              Procesando…
            </span>
          )}
        </div>

        {/* Error message */}
        {service.error_message && (
          <div className="alert alert-error text-xs p-2">
            {service.error_message}
          </div>
        )}
      </div>

      {/* ── Configuración de instancia ────────────────────────────── */}
      <div className="rounded-box bg-base-200 p-4">
        <div className="flex items-center justify-between mb-4">
          <h4 className="font-semibold text-sm">Configuración</h4>
          {!canEditConfig && (
            <span className="text-xs text-warning flex items-center gap-1">
              Detén el servicio para editar
            </span>
          )}
        </div>
        <fieldset disabled={!canEditConfig} className={classNames({ "opacity-60": !canEditConfig })}>
          <ServiceForm
            record={service}
            onSaveRecord={invalidate}
          />
        </fieldset>
      </div>

      {/* ── Acciones avanzadas (solo disponibles con servicio parado) ─ */}
      <div className="rounded-box bg-base-200 p-4 space-y-3">
        <h4 className="font-semibold text-sm mb-2">Acciones avanzadas</h4>
        {!isStopped && (
          <p className="text-xs text-base-content/50">
            Detén el servicio para acceder a estas opciones.
          </p>
        )}
        <div className="flex flex-wrap gap-2">
          <button
            id="btn-service-change-version"
            className="btn btn-sm btn-outline gap-2"
            disabled={!isStopped || isCommandPending}
            onClick={() => handleChangeVersion(service)}
            title="Cambiar la versión de PocketBase (upgrade o downgrade)"
          >
            <Cpu className="w-4 h-4" />
            Cambiar versión
          </button>
          <button
            id="btn-service-clone"
            className="btn btn-sm btn-outline gap-2"
            disabled={!isStopped}
            onClick={() => handleClone(service)}
            title="Clonar esta instancia con todos sus datos"
          >
            <Copy className="w-4 h-4" />
            Clonar
          </button>
          <button
            id="btn-service-backup"
            className="btn btn-sm btn-outline gap-2"
            disabled={!isStopped || isBackupPending}
            onClick={() => handleBackup(service_id)}
            title="Descargar backup de los datos de esta instancia"
          >
            {isBackupPending ? (
              <span className="loading loading-spinner loading-xs" />
            ) : (
              <HardDrive className="w-4 h-4" />
            )}
            Backup
          </button>
        </div>
      </div>

      {/* ── Seguridad y datos ────────────────────────────────────────── */}
      <div className="rounded-box bg-base-200 p-4 space-y-3">
        <h4 className="font-semibold text-sm mb-2">Seguridad</h4>
        <div className="flex flex-wrap gap-2">
          <button
            id="btn-change-password"
            className="btn btn-sm btn-outline gap-2"
            onClick={openChangePassword}
            title="Cambiar la contraseña del superusuario de esta instancia PocketBase"
          >
            <KeyRound className="w-4 h-4" />
            Cambiar contraseña superuser
          </button>
        </div>
      </div>

      {/* ── Zona peligrosa ───────────────────────────────────────────── */}
      <div className="rounded-box border border-error/30 p-4 space-y-3">
        <h4 className="font-semibold text-sm text-error mb-2">Zona peligrosa</h4>
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
