import { useMutation, useQuery } from "@tanstack/react-query";
import { Edit, Plus, RefreshCw, Trash2 } from "lucide-react";
import classNames from "classnames";
import toast from "react-hot-toast";
import { useModal } from "../../components/modal/hook";
import { ErrorFallback } from "../../components/helpers/ErrorFallback";
import { useConfirmModal } from "../../hooks/useConfirmModal";
import { getErrorMessage } from "../../utils/errors";
import { RepositoryForm } from "./forms/RepositoryForm";
import {
  DEFAULT_REPOSITORY_ID,
  repositoriesService,
  type RepositoryDto,
} from "../../services/repositories";

export const SettingsPage = () => {
  const { openModal } = useModal();
  const confirm = useConfirmModal();

  const repositoriesQuery = useQuery({
    queryKey: ["repositories"],
    queryFn: repositoriesService.fetchAll,
    refetchInterval: query =>
      query.state.data?.some(repository => repository.last_sync_status === "syncing")
        ? 3000
        : false,
  });

  const deleteMutation = useMutation({
    mutationFn: repositoriesService.delete,
    onSuccess: () => {
      toast.success("Repository deleted successfully");
      setTimeout(() => repositoriesQuery.refetch());
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const syncMutation = useMutation({
    mutationFn: repositoriesService.sync,
    onSuccess: () => {
      toast.success("Repository synced successfully");
      setTimeout(() => repositoriesQuery.refetch());
    },
    onError: error => {
      toast.error(getErrorMessage(error));
      setTimeout(() => repositoriesQuery.refetch());
    },
  });

  const openRepositoryForm = (repository?: RepositoryDto) => {
    openModal(
      <RepositoryForm
        repository={repository}
        onSave={() => setTimeout(() => repositoriesQuery.refetch())}
      />,
      { title: repository ? "Edit Repository" : "New Repository", width: 520 },
    );
  };

  const deleteRepository = async (repository: RepositoryDto) => {
    const ok = await confirm(
      "Delete repository",
      `Are you sure you want to delete ${repository.name}?`,
    );
    if (ok) deleteMutation.mutate(repository.id);
  };

  if (repositoriesQuery.isLoading) return <div className="p-4">Loading...</div>;
  if (repositoriesQuery.isError) {
    return (
      <ErrorFallback
        error={repositoriesQuery.error}
        onRetry={repositoriesQuery.refetch}
      />
    );
  }

  const repositories = repositoriesQuery.data ?? [];

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold">Settings</h2>
        <p className="text-sm text-base-content/70">
          Manage global launcher configuration.
        </p>
      </div>

      <div className="card bg-base-100 border border-base-300 shadow-sm">
        <div className="card-body gap-4">
          <div className="flex flex-col md:flex-row md:items-center justify-between gap-3">
            <div>
              <h3 className="card-title text-base">Repositories</h3>
              <p className="text-sm text-base-content/70">
                Configure release sources used to create and upgrade instances.
              </p>
            </div>
            <button
              className="btn btn-sm btn-primary gap-2"
              onClick={() => openRepositoryForm()}
            >
              <Plus className="w-4 h-4" />
              New repository
            </button>
          </div>

          <div className="overflow-x-auto">
          <table className="table table-sm">
            <thead>
              <tr>
                <th>Name</th>
                <th>Repository</th>
                <th>Retention</th>
                <th>Status</th>
                <th>Sync</th>
                <th>Last Sync</th>
                <th>Releases</th>
                <th>Release Pattern</th>
                <th>Exec Pattern</th>
                <th className="text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {repositories.map(repository => {
                const isDefault = repository.id === DEFAULT_REPOSITORY_ID;
                return (
                  <tr key={repository.id}>
                    <td>
                      <div className="font-medium">{repository.name}</div>
                      {isDefault && <div className="badge badge-xs badge-info">default</div>}
                    </td>
                    <td className="font-mono whitespace-nowrap">{repository.repository}</td>
                    <td>{repository.retention}</td>
                    <td>
                      <span className={repository.disabled ? "badge badge-warning" : "badge badge-success"}>
                        {repository.disabled ? "Disabled" : "Enabled"}
                      </span>
                    </td>
                    <td>
                      <div className="flex flex-col gap-1">
                        <span
                          className={classNames("badge badge-sm", {
                            "badge-neutral": !repository.last_sync_status || repository.last_sync_status === "never",
                            "badge-info": repository.last_sync_status === "syncing",
                            "badge-success": repository.last_sync_status === "success",
                            "badge-error": repository.last_sync_status === "error",
                          })}
                        >
                          {repository.last_sync_status ?? "never"}
                        </span>
                        {repository.last_sync_error && (
                          <span className="text-xs text-error max-w-[220px] truncate" title={repository.last_sync_error}>
                            {repository.last_sync_error}
                          </span>
                        )}
                      </div>
                    </td>
                    <td className="whitespace-nowrap">
                      {repository.last_sync_at
                        ? new Date(repository.last_sync_at).toLocaleString()
                        : "-"}
                    </td>
                    <td>{repository.release_count ?? 0}</td>
                    <td className="font-mono text-xs max-w-xs truncate">
                      {repository.release_file_pattern}
                    </td>
                    <td className="font-mono text-xs max-w-xs truncate">
                      {repository.exec_file_pattern}
                    </td>
                    <td>
                      <div className="flex justify-end gap-2">
                        <button
                          className="btn btn-xs btn-ghost"
                          disabled={repository.disabled || syncMutation.isPending || repository.last_sync_status === "syncing"}
                          onClick={() => syncMutation.mutate(repository.id)}
                          title="Sync now"
                        >
                          <RefreshCw
                            className={classNames("w-4 h-4", {
                              "animate-spin": repository.last_sync_status === "syncing",
                            })}
                          />
                        </button>
                        <button
                          className="btn btn-xs btn-ghost"
                          onClick={() => openRepositoryForm(repository)}
                        >
                          <Edit className="w-4 h-4" />
                        </button>
                        <button
                          className="btn btn-xs btn-ghost text-error"
                          disabled={isDefault}
                          onClick={() => deleteRepository(repository)}
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
          </div>
        </div>
      </div>
    </div>
  );
};
