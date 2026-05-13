import { useMutation, useQuery } from "@tanstack/react-query";
import { Edit, Plus, Trash2 } from "lucide-react";
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
  });

  const deleteMutation = useMutation({
    mutationFn: repositoriesService.delete,
    onSuccess: () => {
      toast.success("Repository deleted successfully");
      setTimeout(() => repositoriesQuery.refetch());
    },
    onError: error => toast.error(getErrorMessage(error)),
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
