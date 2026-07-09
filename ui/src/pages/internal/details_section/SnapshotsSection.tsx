import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Camera, Download, RotateCcw, Trash2, Play, Square, RefreshCw } from "lucide-react";
import type { FC } from "react";
import toast from "react-hot-toast";
import { ErrorFallback } from "../../../components/helpers/ErrorFallback";
import { useModal } from "../../../components/modal/hook";
import { useConfirmModal } from "../../../hooks/useConfirmModal";
import { backupService, type SnapshotInfo } from "../../../services/backup";
import { serviceService, type ServiceDto } from "../../../services/services";
import { getErrorMessage } from "../../../utils/errors";
import { SnapshotNameForm } from "../forms/SnapshotNameForm";

type Props = {
  service_id: string;
  service?: ServiceDto;
};

const formatSize = (size: number) => {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / 1024 / 1024).toFixed(1)} MB`;
};

export const SnapshotsSection: FC<Props> = ({ service_id, service }) => {
  const { openModal } = useModal();
  const confirm = useConfirmModal();
  const queryClient = useQueryClient();

  const commandMutation = useMutation({
    mutationFn: serviceService.executeServiceCommand,
    onSuccess: (_, variables) => {
      toast.success(`Comando '${variables.action}' enviado con éxito`);
      queryClient.invalidateQueries({ queryKey: ["services", service_id] });
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const handleStartService = () => {
    commandMutation.mutate({ service_id, action: "start" });
  };

  const handleStopService = () => {
    commandMutation.mutate({ service_id, action: "stop" });
  };

  const handleRestartService = () => {
    commandMutation.mutate({ service_id, action: "restart" });
  };

  const snapshotsQuery = useQuery({
    queryKey: ["snapshots", service_id],
    queryFn: () => backupService.listSnapshots(service_id),
  });

  const createMutation = useMutation({
    mutationFn: backupService.createSnapshot,
    onSuccess: () => {
      toast.success("Snapshot creado exitosamente");
      setTimeout(() => snapshotsQuery.refetch());
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const restoreMutation = useMutation({
    mutationFn: backupService.restoreSnapshot,
    onSuccess: () => {
      toast.success("Snapshot restaurado como nueva instancia");
      queryClient.invalidateQueries({ queryKey: ["services"] });
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const downloadMutation = useMutation({
    mutationFn: ({ serviceID, snapshotID }: { serviceID: string; snapshotID: string }) =>
      backupService.downloadSnapshot(serviceID, snapshotID),
    onSuccess: () => toast.success("Snapshot descargado"),
    onError: error => toast.error(getErrorMessage(error)),
  });

  const deleteMutation = useMutation({
    mutationFn: backupService.deleteSnapshot,
    onSuccess: () => {
      toast.success("Snapshot eliminado exitosamente");
      setTimeout(() => snapshotsQuery.refetch());
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const createSnapshot = () => {
    const d = new Date();
    const pad = (n: number) => String(n).padStart(2, "0");
    const dateStr = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}-${pad(d.getHours())}-${pad(d.getMinutes())}`;
    const defaultSnapshotName = service ? `${service.name}-${dateStr}` : "";

    openModal(
      <SnapshotNameForm
        defaultName={defaultSnapshotName}
        description="Crea una copia ZIP local de esta instancia detenida."
        label="Nombre del snapshot"
        submitLabel="Crear snapshot"
        emptyMessage="Ingresa un nombre para el snapshot"
        onSubmit={async name => {
          await createMutation.mutateAsync({ serviceID: service_id, name });
        }}
      />,
      { title: "Crear Snapshot", width: 420 },
    );
  };

  const restoreSnapshot = async (snapshot: SnapshotInfo) => {
    const ok = await confirm(
      "¿Restaurar este snapshot?",
      `Se creará una nueva instancia a partir de "${snapshot.name}" (${snapshot.version}, ${new Date(snapshot.created_at).toLocaleString()}). La instancia actual no se modifica.`,
    );
    if (!ok) return;

    const d = new Date();
    const pad = (n: number) => String(n).padStart(2, "0");
    const dateStr = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
    const name = service ? `${service.name}-restored-${dateStr}` : `restored-${dateStr}`;

    restoreMutation.mutate({ serviceID: service_id, snapshotID: snapshot.id, name });
  };

  const deleteSnapshot = async (snapshot: SnapshotInfo) => {
    const ok = await confirm(
      "Eliminar snapshot",
      `¿Seguro que quieres eliminar "${snapshot.name}"?`,
    );
    if (!ok) return;
    deleteMutation.mutate({ serviceID: service_id, snapshotID: snapshot.id });
  };

  if (service == null || snapshotsQuery.isLoading) {
    return <div className="p-4">Cargando...</div>;
  }
  if (snapshotsQuery.isError) {
    return (
      <ErrorFallback
        error={snapshotsQuery.error}
        onRetry={() => setTimeout(snapshotsQuery.refetch)}
      />
    );
  }

  const snapshots = snapshotsQuery.data ?? [];
  const canCreate = service?.status === "stopped";

  return (
    <div className="space-y-4 min-w-0">
      <div className="flex flex-col sm:flex-row sm:items-start justify-between gap-3">
        <div className="text-sm text-base-content/70 max-w-prose">
          Los snapshots son copias ZIP locales en un punto del tiempo. Crear un snapshot
          requiere que el servicio esté detenido. Restaurar crea una nueva instancia.
        </div>
        <div className="flex flex-wrap gap-2 items-center justify-end shrink-0">
          {/* Botones de Control de Servicio */}
          {!canCreate ? (
            <button
              type="button"
              onClick={handleStopService}
              className="btn btn-sm btn-error gap-1"
              disabled={commandMutation.isPending || service?.status === "pending"}
            >
              <Square className="w-3 h-3 fill-current" />
              Detener Servicio
            </button>
          ) : (
            <button
              type="button"
              onClick={handleStartService}
              className="btn btn-sm btn-success gap-1"
              disabled={commandMutation.isPending || service?.status === "pending"}
            >
              <Play className="w-3 h-3 fill-current" />
              Iniciar Servicio
            </button>
          )}
          <button
            type="button"
            onClick={handleRestartService}
            className="btn btn-sm btn-neutral gap-1"
            disabled={commandMutation.isPending || service?.status === "pending" || service?.status !== "running"}
          >
            <RefreshCw className="w-3 h-3" />
            Reiniciar
          </button>

          <div
            title={canCreate ? undefined : "Detén el servicio antes de crear un snapshot"}
          >
            <button
              className="btn btn-sm btn-primary gap-2"
              disabled={!canCreate || createMutation.isPending}
              onClick={createSnapshot}
            >
              <Camera className="w-4 h-4" />
              Crear snapshot
            </button>
          </div>
        </div>
      </div>

      {!canCreate && (
        <div className="alert alert-info py-2 text-sm flex justify-between items-center gap-3">
          <span>Detén la instancia antes de crear un snapshot.</span>
          <button
            type="button"
            onClick={handleStopService}
            className="btn btn-xs btn-neutral"
            disabled={commandMutation.isPending}
          >
            Detener Ahora
          </button>
        </div>
      )}

      {snapshots.length === 0 ? (
        <div className="rounded-lg border border-dashed border-base-300 p-6 text-center">
          <p className="text-sm text-base-content/60">Sin snapshots. Detén el servicio y crea tu primer snapshot.</p>
        </div>
      ) : (
        <div className="overflow-x-auto rounded-lg">
          <table className="table table-sm w-full">
            <thead>
              <tr>
                <th>Nombre</th>
                <th className="hidden sm:table-cell">Versión</th>
                <th className="hidden md:table-cell">Creado</th>
                <th className="hidden sm:table-cell">Tamaño</th>
                <th className="text-right">Acciones</th>
              </tr>
            </thead>
            <tbody>
              {snapshots.map(snapshot => (
                <tr key={snapshot.id}>
                  <td className="min-w-0">
                    <div className="font-medium truncate max-w-[160px]">{snapshot.name}</div>
                    <div className="text-xs text-base-content/60 sm:hidden">
                      {snapshot.version} · {formatSize(snapshot.size)}
                    </div>
                    <div className="text-xs text-base-content/40 md:hidden hidden sm:block">
                      {new Date(snapshot.created_at).toLocaleDateString()}
                    </div>
                  </td>
                  <td className="hidden sm:table-cell whitespace-nowrap">{snapshot.version}</td>
                  <td className="hidden md:table-cell whitespace-nowrap">
                    {new Date(snapshot.created_at).toLocaleString()}
                  </td>
                  <td className="hidden sm:table-cell whitespace-nowrap">{formatSize(snapshot.size)}</td>
                  <td>
                    <div className="flex justify-end gap-1 flex-wrap">
                      <div title="Descargar snapshot como ZIP">
                        <button
                          className="btn btn-xs btn-ghost gap-1"
                          disabled={downloadMutation.isPending}
                          onClick={() =>
                            downloadMutation.mutate({
                              serviceID: service_id,
                              snapshotID: snapshot.id,
                            })
                          }
                        >
                          <Download className="w-4 h-4" />
                          <span className="hidden sm:inline">Descargar</span>
                        </button>
                      </div>
                      <div title="Restaurar como nueva instancia en este punto">
                        <button
                          className="btn btn-xs btn-ghost gap-1"
                          disabled={restoreMutation.isPending}
                          onClick={() => restoreSnapshot(snapshot)}
                        >
                          <RotateCcw className="w-4 h-4" />
                          <span className="hidden sm:inline">Restaurar</span>
                        </button>
                      </div>
                      <div title="Eliminar snapshot">
                        <button
                          className="btn btn-xs btn-ghost gap-1 text-error"
                          disabled={deleteMutation.isPending}
                          onClick={() => deleteSnapshot(snapshot)}
                        >
                          <Trash2 className="w-4 h-4" />
                          <span className="hidden sm:inline">Eliminar</span>
                        </button>
                      </div>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};
