import {
  lazy,
  Suspense,
  useEffect,
  useState,
  type ChangeEvent,
  type FC,
} from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Download, FilePlus, RefreshCw, Trash2, Upload } from "lucide-react";
import toast from "react-hot-toast";
import { ErrorFallback } from "../../../components/helpers/ErrorFallback";
import { hooksService, type PBHookFile } from "../../../services/hooks";
import { serviceService } from "../../../services/services";
import { getErrorMessage } from "../../../utils/errors";
import { useModal } from "../../../components/modal/hook";
import { useConfirmModal } from "../../../hooks/useConfirmModal";

type Props = {
  service_id: string;
};

const PBHookCodeEditor = lazy(() =>
  import("./PBHookCodeEditor").then(module => ({
    default: module.PBHookCodeEditor,
  })),
);

const formatSize = (size: number) => {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / 1024 / 1024).toFixed(1)} MB`;
};

const indentFor = (path: string) => Math.max(path.split("/").length - 1, 0) * 16;

export const PBHooksSection: FC<Props> = ({ service_id }) => {
  const [file, setFile] = useState<File | null>(null);
  const { openModal } = useModal();

  const serviceQuery = useQuery({
    queryKey: ["services", service_id],
    queryFn: () => serviceService.fetchServiceByID(service_id),
    refetchInterval: 3000,
  });

  const hooksQuery = useQuery({
    queryKey: ["pb-hooks", service_id],
    queryFn: () => hooksService.fetchHooks(service_id),
    refetchInterval: 3000,
  });

  const exportMutation = useMutation({
    mutationFn: hooksService.exportHooks,
    onError: error => toast.error(getErrorMessage(error)),
  });

  const importMutation = useMutation({
    mutationFn: hooksService.importHooks,
    onSuccess: result => {
      toast.success(`Imported ${result.count} PB hook files`);
      setFile(null);
      setTimeout(() => hooksQuery.refetch());
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const handleFileChange = (event: ChangeEvent<HTMLInputElement>) => {
    setFile(event.target.files?.[0] ?? null);
  };

  if (serviceQuery.isError) {
    return <ErrorFallback error={serviceQuery.error} onRetry={serviceQuery.refetch} />;
  }
  if (hooksQuery.isError) {
    return <ErrorFallback error={hooksQuery.error} onRetry={hooksQuery.refetch} />;
  }

  const service = serviceQuery.data;
  const isStopped = service?.status === "stopped";
  const hooks = hooksQuery.data ?? [];

  const openEditor = (hookPath?: string) => {
    openModal(
      <PBHookEditor
        serviceID={service_id}
        initialPath={hookPath}
        isStopped={isStopped}
        onChange={() => setTimeout(() => hooksQuery.refetch())}
      />,
      { title: hookPath ? "Edit PB Hook" : "New PB Hook", width: 900 },
    );
  };

  return (
    <div className="space-y-5">
      <div className="text-sm text-base-content/80">
        Import replaces the full PB hooks folder. Imported hooks apply on next start.
      </div>

      {!isStopped && (
        <div className="alert alert-warning text-sm">
          Stop the service before replacing PB hooks.
        </div>
      )}

      <div className="flex flex-col md:flex-row gap-3 md:items-center">
        <button
          type="button"
          className="btn btn-sm btn-secondary gap-2"
          onClick={() => exportMutation.mutate(service_id)}
          disabled={exportMutation.isPending}
        >
          <Download className="w-4 h-4" />
          Export hooks
        </button>

        <input
          type="file"
          accept=".zip,application/zip"
          className="file-input file-input-sm file-input-bordered w-full md:max-w-sm"
          onChange={handleFileChange}
          disabled={!isStopped || importMutation.isPending}
        />
        <button
          type="button"
          className="btn btn-sm btn-primary gap-2"
          disabled={!isStopped || file == null || importMutation.isPending}
          onClick={() => file && importMutation.mutate({ serviceID: service_id, file })}
        >
          <Upload className="w-4 h-4" />
          Import ZIP
        </button>

        <button
          type="button"
          className="btn btn-sm btn-ghost gap-2"
          onClick={() => hooksQuery.refetch()}
        >
          <RefreshCw className="w-4 h-4" />
          Refresh
        </button>
        <button
          type="button"
          className="btn btn-sm btn-accent gap-2"
          disabled={!isStopped}
          onClick={() => openEditor()}
        >
          <FilePlus className="w-4 h-4" />
          New hook
        </button>
      </div>

      {hooksQuery.isLoading ? (
        <div className="p-4">Loading...</div>
      ) : hooks.length === 0 ? (
        <div className="text-sm text-base-content/70">No PB hooks published.</div>
      ) : (
        <HooksTable hooks={hooks} onOpenHook={openEditor} />
      )}
    </div>
  );
};

const HooksTable: FC<{
  hooks: PBHookFile[];
  onOpenHook: (path: string) => void;
}> = ({ hooks, onOpenHook }) => {
  return (
    <div className="overflow-x-auto">
      <table className="table table-sm">
        <thead>
          <tr>
            <th>File</th>
            <th>Size</th>
            <th>Updated</th>
          </tr>
        </thead>
        <tbody>
          {hooks.map(hook => (
            <tr
              key={hook.path}
              className="cursor-pointer hover:bg-base-300"
              onClick={() => onOpenHook(hook.path)}
            >
              <td className="font-mono whitespace-nowrap text-primary">
                <span style={{ paddingLeft: indentFor(hook.path) }}>{hook.path}</span>
              </td>
              <td className="whitespace-nowrap">{formatSize(hook.size)}</td>
              <td className="whitespace-nowrap">
                {new Date(hook.updated_at).toLocaleString()}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

type PBHookEditorProps = {
  serviceID: string;
  initialPath?: string;
  isStopped: boolean;
  onChange: () => void;
};

const PBHookEditor: FC<PBHookEditorProps> = ({
  serviceID,
  initialPath,
  isStopped,
  onChange,
}) => {
  const { closeModal } = useModal();
  const confirm = useConfirmModal();
  const [path, setPath] = useState(initialPath ?? "main.pb.js");
  const [content, setContent] = useState("");

  const isNew = initialPath == null;

  const hookQuery = useQuery({
    queryKey: ["pb-hook", serviceID, initialPath],
    queryFn: () => hooksService.readHook(serviceID, initialPath ?? ""),
    enabled: initialPath != null,
  });

  useEffect(() => {
    if (hookQuery.data) {
      setPath(hookQuery.data.path);
      setContent(hookQuery.data.content);
    }
  }, [hookQuery.data]);

  const saveMutation = useMutation({
    mutationFn: hooksService.saveHook,
    onSuccess: () => {
      toast.success("PB hook saved successfully");
      onChange();
      closeModal();
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const deleteMutation = useMutation({
    mutationFn: hooksService.deleteHook,
    onSuccess: () => {
      toast.success("PB hook deleted successfully");
      onChange();
      closeModal();
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const save = () => {
    saveMutation.mutate({ serviceID, path, content });
  };

  const deleteHook = async () => {
    const ok = await confirm(
      "Delete PB hook",
      `Are you sure you want to delete ${path}?`,
    );
    if (ok) deleteMutation.mutate({ serviceID, path });
  };

  if (hookQuery.isLoading) return <div className="p-4">Loading...</div>;
  if (hookQuery.isError) {
    return <ErrorFallback error={hookQuery.error} onRetry={hookQuery.refetch} />;
  }

  return (
    <div className="space-y-4">
      {!isStopped && (
        <div className="alert alert-warning text-sm">
          Stop the service before editing PB hooks.
        </div>
      )}
      <div className="form-control w-full">
        <label className="label">
          <span className="label-text mb-1">File path</span>
        </label>
        <input
          className="input input-bordered w-full font-mono"
          value={path}
          disabled={!isNew || !isStopped}
          onChange={event => setPath(event.target.value)}
          placeholder="main.pb.js"
        />
      </div>
      <div className="form-control w-full">
        <label className="label">
          <span className="label-text mb-1">Code</span>
        </label>
        <div
          className="overflow-hidden rounded-lg border border-base-300"
          aria-disabled={!isStopped}
        >
          <Suspense fallback={<div className="p-4">Loading editor...</div>}>
            <PBHookCodeEditor
              value={content}
              editable={isStopped}
              onChange={value => setContent(value)}
            />
          </Suspense>
        </div>
      </div>
      <div className="flex flex-col md:flex-row justify-between gap-2">
        <button
          type="button"
          className="btn btn-error gap-2"
          disabled={isNew || !isStopped || deleteMutation.isPending}
          onClick={deleteHook}
        >
          <Trash2 className="w-4 h-4" />
          Delete
        </button>
        <div className="flex gap-2 justify-end">
          <button type="button" className="btn btn-ghost" onClick={closeModal}>
            Close
          </button>
          <button
            type="button"
            className="btn btn-primary"
            disabled={!isStopped || saveMutation.isPending || path.trim() === ""}
            onClick={save}
          >
            Save
          </button>
        </div>
      </div>
    </div>
  );
};
