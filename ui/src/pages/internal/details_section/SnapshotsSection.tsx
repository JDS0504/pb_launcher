import { useMutation, useQuery } from "@tanstack/react-query";
import { Camera, RotateCcw, Trash2 } from "lucide-react";
import type { FC } from "react";
import toast from "react-hot-toast";
import { ErrorFallback } from "../../../components/helpers/ErrorFallback";
import { useConfirmModal } from "../../../hooks/useConfirmModal";
import { backupService, type SnapshotInfo } from "../../../services/backup";
import { serviceService } from "../../../services/services";
import { getErrorMessage } from "../../../utils/errors";

type Props = {
  service_id: string;
};

const formatSize = (size: number) => {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / 1024 / 1024).toFixed(1)} MB`;
};

export const SnapshotsSection: FC<Props> = ({ service_id }) => {
  const confirm = useConfirmModal();
  const serviceQuery = useQuery({
    queryKey: ["services", service_id],
    queryFn: () => serviceService.fetchServiceByID(service_id),
  });
  const snapshotsQuery = useQuery({
    queryKey: ["snapshots", service_id],
    queryFn: () => backupService.listSnapshots(service_id),
  });

  const createMutation = useMutation({
    mutationFn: backupService.createSnapshot,
    onSuccess: () => {
      toast.success("Snapshot created successfully");
      setTimeout(() => snapshotsQuery.refetch());
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const restoreMutation = useMutation({
    mutationFn: backupService.restoreSnapshot,
    onSuccess: () => toast.success("Snapshot restore started successfully"),
    onError: error => toast.error(getErrorMessage(error)),
  });

  const deleteMutation = useMutation({
    mutationFn: backupService.deleteSnapshot,
    onSuccess: () => {
      toast.success("Snapshot deleted successfully");
      setTimeout(() => snapshotsQuery.refetch());
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const createSnapshot = () => {
    const name = window.prompt("Snapshot name");
    if (name == null) return;
    const trimmedName = name.trim();
    if (trimmedName === "") {
      toast.error("Enter a snapshot name");
      return;
    }
    createMutation.mutate({ serviceID: service_id, name: trimmedName });
  };

  const restoreSnapshot = (snapshot: SnapshotInfo) => {
    const name = window.prompt("Restored instance name", `${snapshot.source_service} snapshot`);
    if (name == null) return;
    const trimmedName = name.trim();
    if (trimmedName === "") {
      toast.error("Enter an instance name");
      return;
    }
    restoreMutation.mutate({
      serviceID: service_id,
      snapshotID: snapshot.id,
      name: trimmedName,
    });
  };

  const deleteSnapshot = async (snapshot: SnapshotInfo) => {
    const ok = await confirm(
      "Delete snapshot",
      `Are you sure you want to delete ${snapshot.name}?`,
    );
    if (!ok) return;
    deleteMutation.mutate({ serviceID: service_id, snapshotID: snapshot.id });
  };

  if (serviceQuery.isLoading || snapshotsQuery.isLoading) {
    return <div className="p-4">Loading...</div>;
  }
  if (serviceQuery.isError) {
    return (
      <ErrorFallback
        error={serviceQuery.error}
        onRetry={() => setTimeout(serviceQuery.refetch)}
      />
    );
  }
  if (snapshotsQuery.isError) {
    return (
      <ErrorFallback
        error={snapshotsQuery.error}
        onRetry={() => setTimeout(snapshotsQuery.refetch)}
      />
    );
  }

  const service = serviceQuery.data;
  const snapshots = snapshotsQuery.data ?? [];
  const canCreate = service?.status === "stopped";

  return (
    <div className="space-y-4">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-3">
        <div className="text-sm text-base-content/70">
          Snapshots are local point-in-time ZIP copies. Creating a snapshot requires
          the service to be stopped. Restoring creates a new instance.
        </div>
        <button
          className="btn btn-sm btn-primary gap-2"
          disabled={!canCreate || createMutation.isPending}
          onClick={createSnapshot}
          title={canCreate ? "Create snapshot" : "Stop the service before creating a snapshot"}
        >
          <Camera className="w-4 h-4" />
          Snapshot
        </button>
      </div>

      {snapshots.length === 0 ? (
        <div className="text-sm text-base-content/70">No snapshots yet.</div>
      ) : (
        <div className="overflow-x-auto">
          <table className="table table-sm">
            <thead>
              <tr>
                <th>Name</th>
                <th>Version</th>
                <th>Created</th>
                <th>Size</th>
                <th className="text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {snapshots.map(snapshot => (
                <tr key={snapshot.id}>
                  <td>
                    <div className="font-medium">{snapshot.name}</div>
                    <div className="text-xs text-base-content/60">{snapshot.id}</div>
                  </td>
                  <td>{snapshot.version}</td>
                  <td className="whitespace-nowrap">
                    {new Date(snapshot.created_at).toLocaleString()}
                  </td>
                  <td>{formatSize(snapshot.size)}</td>
                  <td>
                    <div className="flex justify-end gap-2">
                      <button
                        className="btn btn-xs btn-ghost"
                        disabled={restoreMutation.isPending}
                        onClick={() => restoreSnapshot(snapshot)}
                        title="Restore as new instance"
                      >
                        <RotateCcw className="w-4 h-4" />
                      </button>
                      <button
                        className="btn btn-xs btn-ghost text-error"
                        disabled={deleteMutation.isPending}
                        onClick={() => deleteSnapshot(snapshot)}
                        title="Delete snapshot"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
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
