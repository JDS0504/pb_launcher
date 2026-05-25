import { useMutation, useQuery } from "@tanstack/react-query";
import { Camera, RotateCcw, Trash2 } from "lucide-react";
import type { FC } from "react";
import toast from "react-hot-toast";
import { useNavigate } from "react-router-dom";
import { ErrorFallback } from "../../../components/helpers/ErrorFallback";
import { useModal } from "../../../components/modal/hook";
import { useConfirmModal } from "../../../hooks/useConfirmModal";
import { backupService, type SnapshotInfo } from "../../../services/backup";
import type { ServiceDto } from "../../../services/services";
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
  const navigate = useNavigate();
  const { openModal } = useModal();
  const confirm = useConfirmModal();

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
    onSuccess: data => {
      toast.success("Snapshot restored as a new instance");
      navigate(`/services/${data.service_id}`);
    },
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
    openModal(
      <SnapshotNameForm
        description="Create a local point-in-time ZIP snapshot of this stopped instance."
        label="Snapshot Name"
        submitLabel="Create snapshot"
        emptyMessage="Enter a snapshot name"
        onSubmit={async name => {
          await createMutation.mutateAsync({ serviceID: service_id, name });
        }}
      />,
      { title: "Create Snapshot", width: 420 },
    );
  };

  const restoreSnapshot = (snapshot: SnapshotInfo) => {
    openModal(
      <SnapshotNameForm
        defaultName={`${snapshot.source_service} snapshot`}
        description="Restore this snapshot as a new instance. The current instance is not modified."
        label="Restored Instance Name"
        submitLabel="Restore snapshot"
        emptyMessage="Enter an instance name"
        onSubmit={async name => {
          await restoreMutation.mutateAsync({
            serviceID: service_id,
            snapshotID: snapshot.id,
            name,
          });
        }}
      />,
      { title: "Restore Snapshot", width: 420 },
    );
  };

  const deleteSnapshot = async (snapshot: SnapshotInfo) => {
    const ok = await confirm(
      "Delete snapshot",
      `Are you sure you want to delete ${snapshot.name}?`,
    );
    if (!ok) return;
    deleteMutation.mutate({ serviceID: service_id, snapshotID: snapshot.id });
  };

  if (service == null || snapshotsQuery.isLoading) {
    return <div className="p-4">Loading...</div>;
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
          title={
            canCreate
              ? "Create snapshot"
              : "Stop the service before creating a snapshot"
          }
        >
          <Camera className="w-4 h-4" />
          Create snapshot
        </button>
      </div>

      {!canCreate && (
        <div className="alert alert-info py-2 text-sm">
          Stop this instance before creating a snapshot.
        </div>
      )}

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
                        className="btn btn-xs btn-ghost gap-1"
                        disabled={restoreMutation.isPending}
                        onClick={() => restoreSnapshot(snapshot)}
                        title="Restore as new instance"
                      >
                        <RotateCcw className="w-4 h-4" />
                        Restore
                      </button>
                      <button
                        className="btn btn-xs btn-ghost gap-1 text-error"
                        disabled={deleteMutation.isPending}
                        onClick={() => deleteSnapshot(snapshot)}
                        title="Delete snapshot"
                      >
                        <Trash2 className="w-4 h-4" />
                        Delete
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
