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
  Folder,
  ChevronDown,
  ChevronRight,
  Play,
  Square,
  Trash2,
  Save,
  Plus,
  RefreshCw,
  AlertTriangle,
  Upload,
} from "lucide-react";
import toast from "react-hot-toast";
import { ErrorFallback } from "../../../components/helpers/ErrorFallback";
import { filesService, type PBFileEntry } from "../../../services/files";
import { serviceService, type ServiceDto } from "../../../services/services";
import { getErrorMessage } from "../../../utils/errors";
import { useModal } from "../../../components/modal/hook";
import { useConfirmModal } from "../../../hooks/useConfirmModal";
import { NewFileModal } from "../components/NewFileModal";
import { UploadFilesModal } from "../components/UploadFilesModal";

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
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set());

  const togglePath = (path: string) => {
    setExpandedPaths((prev) => {
      const next = new Set(prev);
      if (next.has(path)) {
        next.delete(path);
        for (const p of next) {
          if (p.startsWith(path + "/")) {
            next.delete(p);
          }
        }
      } else {
        next.add(path);
      }
      return next;
    });
  };

  const isPathVisible = (path: string, expanded: Set<string>): boolean => {
    const parts = path.split("/");
    if (parts.length <= 1) return true;
    for (let i = 1; i < parts.length; i++) {
      const parent = parts.slice(0, i).join("/");
      if (!expanded.has(parent)) return false;
    }
    return true;
  };

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

  const openUploadFilesModal = () => {
    openModal(
      <UploadFilesModal
        serviceID={service_id}
        isStopped={isStopped}
        onUploaded={() => {
          filesQuery.refetch();
        }}
      />,
      { title: "Subir Archivos", width: 480 }
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

          <button
            type="button"
            className="btn btn-sm btn-primary gap-1.5"
            disabled={!isStopped}
            onClick={openUploadFilesModal}
          >
            <Upload className="w-4 h-4" />
            Subir Archivos
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
              files
                .filter((f) => isPathVisible(f.path, expandedPaths))
                .map((f) => {
                  const isSelected = selectedPath === f.path;
                  return (
                    <button
                      key={f.path}
                      type="button"
                      onClick={() => {
                        setSelectedPath(f.path);
                        if (f.is_dir) {
                          togglePath(f.path);
                        }
                      }}
                      className={`w-full text-left py-1.5 px-2 rounded-lg flex items-center justify-between gap-2 transition-colors ${
                        isSelected
                          ? "bg-primary text-primary-content"
                          : "hover:bg-base-300 text-base-content/90"
                      }`}
                    >
                      <span
                        className="flex items-center gap-1 overflow-hidden text-ellipsis whitespace-nowrap"
                        style={{ paddingLeft: `${indentFor(f.path)}px` }}
                      >
                        {f.is_dir ? (
                          <>
                            {expandedPaths.has(f.path) ? (
                              <ChevronDown className="w-3.5 h-3.5 shrink-0 text-base-content/60" />
                            ) : (
                              <ChevronRight className="w-3.5 h-3.5 shrink-0 text-base-content/60" />
                            )}
                            {expandedPaths.has(f.path) ? (
                              <FolderOpen className="w-3.5 h-3.5 shrink-0 text-amber-500" />
                            ) : (
                              <Folder className="w-3.5 h-3.5 shrink-0 text-amber-500" />
                            )}
                          </>
                        ) : (
                          <>
                            <span className="w-3.5 shrink-0" />
                            <FileText className="w-3.5 h-3.5 shrink-0" />
                          </>
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


