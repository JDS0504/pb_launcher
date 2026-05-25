import {
  lazy,
  Suspense,
  useEffect,
  useState,
  type FC,
} from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  FileText,
  FolderOpen,
  Play,
  Square,
  Trash2,
  Save,
  Plus,
  RefreshCw,
  AlertTriangle,
} from "lucide-react";
import toast from "react-hot-toast";
import { ErrorFallback } from "../../../components/helpers/ErrorFallback";
import { filesService, type PBFileEntry } from "../../../services/files";
import { serviceService, type ServiceDto } from "../../../services/services";
import { getErrorMessage } from "../../../utils/errors";
import { useModal } from "../../../components/modal/hook";
import { useConfirmModal } from "../../../hooks/useConfirmModal";

type Props = {
  service_id: string;
  service?: ServiceDto;
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

const indentFor = (path: string) => {
  const parts = path.split("/");
  return Math.max(parts.length - 1, 0) * 16;
};

const getFileName = (path: string) => {
  const parts = path.split("/");
  return parts[parts.length - 1];
};

export const FileManagerSection: FC<Props> = ({ service_id, service }) => {
  const { openModal } = useModal();
  const confirm = useConfirmModal();
  const queryClient = useQueryClient();

  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [editorContent, setEditorContent] = useState("");
  const [originalContent, setOriginalContent] = useState("");
  const [isBinaryFile, setIsBinaryFile] = useState(false);

  const filesQuery = useQuery<PBFileEntry[]>({
    queryKey: ["pb-files", service_id],
    queryFn: () => filesService.fetchFiles(service_id),
  });

  const files = filesQuery.data ?? [];
  const selectedFile = files.find(f => f.path === selectedPath);
  const isDirSelected = selectedFile?.is_dir === true;

  const fileContentQuery = useQuery({
    queryKey: ["pb-file-content", service_id, selectedPath],
    queryFn: () => filesService.readFile(service_id, selectedPath || ""),
    enabled: selectedPath != null && !isBinaryFile && !isDirSelected,
  });

  useEffect(() => {
    if (selectedPath) {
      const isBinary =
        !isDirSelected && (
        selectedPath.endsWith(".db") ||
        selectedPath.endsWith(".png") ||
        selectedPath.endsWith(".jpg") ||
        selectedPath.endsWith(".jpeg") ||
        selectedPath.endsWith(".gif") ||
        selectedPath.endsWith(".ico") ||
        selectedPath.endsWith(".zip")
        );
      setIsBinaryFile(isBinary);
      if (isBinary || isDirSelected) {
        setEditorContent("");
        setOriginalContent("");
      }
    } else {
      setIsBinaryFile(false);
      setEditorContent("");
      setOriginalContent("");
    }
  }, [selectedPath, isDirSelected]);

  useEffect(() => {
    if (fileContentQuery.data && !isBinaryFile && !isDirSelected) {
      setEditorContent(fileContentQuery.data.content);
      setOriginalContent(fileContentQuery.data.content);
    }
  }, [fileContentQuery.data, isBinaryFile, isDirSelected]);

  const commandMutation = useMutation({
    mutationFn: serviceService.executeServiceCommand,
    onSuccess: (_, variables) => {
      toast.success(`Comando '${variables.action}' enviado con éxito`);
      queryClient.invalidateQueries({ queryKey: ["services", service_id] });
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const saveMutation = useMutation({
    mutationFn: filesService.saveFile,
    onSuccess: () => {
      toast.success("Archivo guardado con éxito");
      setOriginalContent(editorContent);
      filesQuery.refetch();
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const deleteMutation = useMutation({
    mutationFn: filesService.deleteFile,
    onSuccess: () => {
      toast.success("Archivo eliminado con éxito");
      setSelectedPath(null);
      filesQuery.refetch();
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  if (service == null) {
    return <div className="p-4">Loading...</div>;
  }
  if (filesQuery.isError) {
    return <ErrorFallback error={filesQuery.error} onRetry={filesQuery.refetch} />;
  }
  const isStopped = service?.status === "stopped";
  const hasChanges = editorContent !== originalContent;

  const handleStartService = () => {
    commandMutation.mutate({ service_id, action: "start" });
  };

  const handleStopService = () => {
    commandMutation.mutate({ service_id, action: "stop" });
  };

  const handleSave = () => {
    if (!selectedPath) return;
    saveMutation.mutate({
      serviceID: service_id,
      path: selectedPath,
      content: editorContent,
    });
  };

  const handleDelete = async () => {
    if (!selectedPath) return;
    const ok = await confirm(
      "Eliminar Archivo",
      `¿Estás seguro de que deseas eliminar permanentemente el archivo ${selectedPath}?`
    );
    if (ok) {
      deleteMutation.mutate({ serviceID: service_id, path: selectedPath });
    }
  };

  const openNewFileModal = () => {
    openModal(
      <NewFileModal
        serviceID={service_id}
        isStopped={isStopped}
        onCreated={(newPath) => {
          filesQuery.refetch().then(() => {
            setSelectedPath(newPath);
          });
        }}
      />,
      { title: "Nuevo Archivo", width: 450 }
    );
  };

  // Determinar advertencias
  const showDbWarning = selectedPath?.endsWith(".db") || selectedPath?.includes("pb_data");

  return (
    <div className="space-y-4">
      {/* Barra de control rápido de Estado */}
      <div className="flex flex-wrap items-center justify-between gap-4 p-4 rounded-xl bg-base-300 border border-base-200">
        <div className="flex items-center gap-3">
          <span className="font-semibold text-sm">Estado de la Instancia:</span>
          <span
            className={`badge font-semibold text-xs py-2.5 px-3 uppercase ${
              service?.status === "running"
                ? "badge-success text-success-content"
                : service?.status === "sleeping"
                ? "badge-info text-info-content"
                : service?.status === "pending"
                ? "badge-warning text-warning-content animate-pulse"
                : "badge-error text-error-content"
            }`}
          >
            {service?.status}
          </span>
        </div>

        <div className="flex gap-2">
          {!isStopped ? (
            <button
              type="button"
              className="btn btn-sm btn-error gap-1.5"
              onClick={handleStopService}
              disabled={commandMutation.isPending || service?.status === "pending"}
            >
              <Square className="w-3.5 h-3.5 fill-current" />
              Detener Servicio
            </button>
          ) : (
            <button
              type="button"
              className="btn btn-sm btn-success gap-1.5"
              onClick={handleStartService}
              disabled={commandMutation.isPending || service?.status === "pending"}
            >
              <Play className="w-3.5 h-3.5 fill-current" />
              Iniciar Servicio
            </button>
          )}

          <button
            type="button"
            className="btn btn-sm btn-ghost gap-1.5"
            onClick={() => filesQuery.refetch()}
            disabled={filesQuery.isFetching}
          >
            <RefreshCw className={`w-3.5 h-3.5 ${filesQuery.isFetching ? "animate-spin" : ""}`} />
            Actualizar Lista
          </button>

          <button
            type="button"
            className="btn btn-sm btn-primary gap-1.5"
            disabled={!isStopped}
            onClick={openNewFileModal}
          >
            <Plus className="w-4 h-4" />
            Nuevo Archivo
          </button>
        </div>
      </div>

      {/* Contenedor Principal: Árbol lateral + Editor */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-4 h-[65vh]">
        {/* Árbol de Archivos (Col 4) */}
        <div className="lg:col-span-4 flex flex-col bg-base-200 border border-base-300 rounded-xl overflow-hidden h-full">
          <div className="p-3 bg-base-300 font-semibold text-xs uppercase tracking-wider border-b border-base-300 flex justify-between items-center">
            <span>Explorador de Archivos</span>
            <span className="badge badge-sm badge-neutral">{files.length} archivos</span>
          </div>

          <div className="flex-1 overflow-y-auto p-2 space-y-1 font-mono text-xs">
            {files.length === 0 ? (
              <div className="p-4 text-center text-base-content/50">No hay archivos en la instancia.</div>
            ) : (
              files.map((f) => {
                const isSelected = selectedPath === f.path;
                return (
                  <button
                    key={f.path}
                    type="button"
                    onClick={() => setSelectedPath(f.path)}
                    className={`w-full text-left py-1.5 px-2 rounded-lg flex items-center justify-between gap-2 transition-colors ${
                      isSelected
                        ? "bg-primary text-primary-content"
                        : "hover:bg-base-300 text-base-content/90"
                    }`}
                  >
                    <span
                      className="flex items-center gap-1.5 overflow-hidden text-ellipsis whitespace-nowrap"
                      style={{ paddingLeft: `${indentFor(f.path)}px` }}
                    >
                      {f.is_dir ? (
                        <FolderOpen className="w-3.5 h-3.5 shrink-0 text-amber-500" />
                      ) : (
                        <FileText className="w-3.5 h-3.5 shrink-0" />
                      )}
                      <span>{getFileName(f.path)}</span>
                    </span>
                    <span className={`text-[10px] opacity-70 shrink-0 ${isSelected ? "text-primary-content" : ""}`}>
                      {!f.is_dir && formatSize(f.size)}
                    </span>
                  </button>
                );
              })
            )}
          </div>
        </div>

        {/* Editor de Código (Col 8) */}
        <div className="lg:col-span-8 flex flex-col bg-base-200 border border-base-300 rounded-xl overflow-hidden h-full">
          {selectedPath == null ? (
            <div className="flex-1 flex flex-col items-center justify-center text-center p-6 text-base-content/60 space-y-2">
              <FolderOpen className="w-12 h-12 stroke-1 text-base-content/40" />
              <p className="text-sm">Selecciona un archivo del explorador lateral para comenzar a visualizarlo o editarlo.</p>
              {!isStopped && (
                <div className="alert alert-warning text-xs max-w-md mt-2">
                  Nota: El servicio está encendido. Si necesitas modificar, crear o borrar archivos, deberás apagarlo temporalmente.
                </div>
              )}
            </div>
          ) : (
            <div className="flex-grow flex flex-col min-h-0">
              {/* Encabezado del Editor */}
              <div className="p-3 bg-base-300 border-b border-base-300 flex flex-wrap justify-between items-center gap-2">
                <span className="font-mono text-xs font-semibold text-primary overflow-hidden text-ellipsis whitespace-nowrap max-w-xs md:max-w-md">
                  {selectedPath}
                </span>

                <div className="flex gap-2">
                  <button
                    type="button"
                    className="btn btn-xs btn-error gap-1"
                    disabled={!isStopped || deleteMutation.isPending}
                    onClick={handleDelete}
                  >
                    <Trash2 className="w-3 h-3" />
                    Borrar
                  </button>

                  {!isDirSelected && (
                    <button
                      type="button"
                      className="btn btn-xs btn-primary gap-1"
                      disabled={!isStopped || !hasChanges || saveMutation.isPending || isBinaryFile}
                      onClick={handleSave}
                    >
                      <Save className="w-3 h-3" />
                      Guardar
                    </button>
                  )}
                </div>
              </div>

              {/* Advertencias sobre pb_data o archivos .db */}
              {showDbWarning && (
                <div className="alert alert-error rounded-none text-xs flex gap-2 py-2">
                  <AlertTriangle className="w-4 h-4 shrink-0 text-error-content" />
                  <div>
                    <span className="font-semibold">ADVERTENCIA CRÍTICA:</span> Este archivo/carpeta pertenece a{" "}
                    <span className="font-mono">pb_data</span> o contiene datos binarios. Modificarlo o escribir sobre él puede{" "}
                    <span className="font-semibold">CORROMPER</span> o romper tu base de datos de PocketBase de forma permanente. Procede con extremo cuidado.
                  </div>
                </div>
              )}

              {/* Advertencia de Estado Activo */}
              {!isStopped && (
                <div className="alert alert-warning rounded-none text-xs flex justify-between py-2 gap-3">
                  <div className="flex gap-2 items-center">
                    <AlertTriangle className="w-4 h-4 shrink-0 text-warning-content" />
                    <span>El servicio debe estar detenido para poder guardar o aplicar modificaciones en los archivos.</span>
                  </div>
                  <button
                    type="button"
                    onClick={handleStopService}
                    className="btn btn-xs btn-warning"
                    disabled={commandMutation.isPending}
                  >
                    Detener Ahora
                  </button>
                </div>
              )}

              {/* Área del Editor de Código */}
              <div className="flex-1 overflow-hidden min-h-0 bg-base-100">
                {isDirSelected ? (
                  <div className="w-full h-full flex flex-col items-center justify-center text-center p-6 text-base-content/60 space-y-2">
                    <FolderOpen className="w-12 h-12 text-amber-500 stroke-1" />
                    <p className="font-semibold text-sm">Directorio seleccionado ({getFileName(selectedPath)})</p>
                    <p className="text-xs max-w-md">Para agregar nuevos archivos dentro de este directorio o sus subcarpetas, haz clic en el botón de la barra superior <span className="font-semibold text-primary">"Nuevo Archivo"</span>.</p>
                  </div>
                ) : isBinaryFile ? (
                  <div className="w-full h-full flex flex-col items-center justify-center text-center p-6 text-base-content/60 space-y-2">
                    <AlertTriangle className="w-10 h-10 text-error" />
                    <p className="font-semibold text-sm">Este es un archivo binario o base de datos ({getFileName(selectedPath)})</p>
                    <p className="text-xs max-w-md">No es posible renderizar archivos binarios como texto de forma segura. Editarlo directamente en texto dañará el archivo.</p>
                  </div>
                ) : fileContentQuery.isLoading ? (
                  <div className="p-4 text-xs">Cargando contenido del archivo...</div>
                ) : fileContentQuery.isError ? (
                  <div className="p-4 text-xs text-error">
                    Error al abrir archivo: {getErrorMessage(fileContentQuery.error)}
                  </div>
                ) : (
                  <Suspense fallback={<div className="p-4 text-xs">Cargando editor...</div>}>
                    <PBHookCodeEditor
                      value={editorContent}
                      editable={isStopped}
                      onChange={(val) => setEditorContent(val)}
                    />
                  </Suspense>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

// Modal para crear archivos
type NewFileModalProps = {
  serviceID: string;
  isStopped: boolean;
  onCreated: (path: string) => void;
};

const NewFileModal: FC<NewFileModalProps> = ({ serviceID, isStopped, onCreated }) => {
  const { closeModal } = useModal();
  const [folder, setFolder] = useState("pb_public");
  const [filePath, setFilePath] = useState("");

  const saveMutation = useMutation({
    mutationFn: filesService.saveFile,
    onSuccess: (_, variables) => {
      toast.success("Archivo creado con éxito");
      onCreated(variables.path);
      closeModal();
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const cleanPath = filePath.trim();
    if (!cleanPath) {
      toast.error("La ruta es obligatoria");
      return;
    }
    const finalPath = `${folder}/${cleanPath.startsWith("/") ? cleanPath.substring(1) : cleanPath}`;
    saveMutation.mutate({
      serviceID,
      path: finalPath,
      content: "",
    });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4 text-sm">
      {!isStopped && (
        <div className="alert alert-warning text-xs">
          Debes detener la instancia antes de poder crear un nuevo archivo.
        </div>
      )}

      <div className="form-control w-full">
        <label className="label">
          <span className="label-text mb-1">Directorio de Origen</span>
        </label>
        <select
          className="select select-bordered select-sm w-full font-mono"
          value={folder}
          onChange={(e) => setFolder(e.target.value)}
          disabled={!isStopped}
        >
          <option value="pb_public">pb_public (Estáticos web)</option>
          <option value="pb_hooks">pb_hooks (JS Hooks)</option>
          <option value="pb_migrations">pb_migrations (Migraciones DB)</option>
          <option value="pb_data">pb_data (Datos internos)</option>
        </select>
      </div>

      <div className="form-control w-full">
        <label className="label">
          <span className="label-text mb-1">Nombre / Ruta relativa del archivo</span>
        </label>
        <input
          type="text"
          className="input input-bordered input-sm w-full font-mono text-xs"
          placeholder="ej: index.html, subcarpeta/styles.css, main.pb.js"
          value={filePath}
          onChange={(e) => setFilePath(e.target.value)}
          disabled={!isStopped}
          required
        />
      </div>

      <div className="flex justify-end gap-2 pt-2">
        <button type="button" className="btn btn-sm btn-ghost" onClick={closeModal}>
          Cancelar
        </button>
        <button
          type="submit"
          className="btn btn-sm btn-primary"
          disabled={!isStopped || saveMutation.isPending || filePath.trim() === ""}
        >
          Crear Archivo
        </button>
      </div>
    </form>
  );
};
