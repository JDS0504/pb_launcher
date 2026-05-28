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
  Server,
  Trash2,
  Save,
  Plus,
  AlertTriangle,
  Upload,
  Download,
  Search,
  RefreshCw,
  Play,
  Square,
} from "lucide-react";
import toast from "react-hot-toast";
import { ErrorFallback } from "../../components/helpers/ErrorFallback";
import { filesService, type PBFileEntry } from "../../services/files";
import { serviceService, type ServiceDto } from "../../services/services";
import { getErrorMessage } from "../../utils/errors";
import { useModal } from "../../components/modal/hook";
import { useConfirmModal } from "../../hooks/useConfirmModal";
import { NewFileModal } from "./components/NewFileModal";
import { UploadFilesModal } from "./components/UploadFilesModal";
import { NewFolderModal } from "./components/NewFolderModal";
import { RenameModal } from "./components/RenameModal";

const PBHookCodeEditor = lazy(() =>
  import("./details_section/PBHookCodeEditor").then((module) => ({
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

// Componente para renderizar de forma perezosa el árbol de archivos de una instancia
type ServiceFileTreeProps = {
  service: ServiceDto;
  isExpanded: boolean;
  onToggleExpand: () => void;
  selectedServiceId?: string;
  selectedPath: string | null;
  onSelectFile: (service: ServiceDto, path: string) => void;
  searchQuery: string;
  selectedFilePaths: Set<string>;
  setSelectedFilePaths: React.Dispatch<React.SetStateAction<Set<string>>>;
  isStopped: boolean;
};

const ServiceFileTree: FC<ServiceFileTreeProps> = ({
  service,
  isExpanded,
  onToggleExpand,
  selectedServiceId,
  selectedPath,
  onSelectFile,
  searchQuery,
  selectedFilePaths,
  setSelectedFilePaths,
  isStopped,
}) => {
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set());

  const showFiles = isExpanded || searchQuery.trim() !== "";

  const filesQuery = useQuery<PBFileEntry[]>({
    queryKey: ["pb-files", service.id],
    queryFn: () => filesService.fetchFiles(service.id),
    enabled: showFiles,
  });

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

  const files = filesQuery.data ?? [];
  const visibleFiles = searchQuery.trim() !== ""
    ? files.filter(f => getFileName(f.path).toLowerCase().includes(searchQuery.toLowerCase()))
    : files.filter((f) => isPathVisible(f.path, expandedPaths));

  return (
    <div className="space-y-0.5">
      {/* Nodo Raíz: El Servicio / Instancia */}
      <button
        type="button"
        onClick={onToggleExpand}
        className={`w-full text-left py-1.5 px-2 rounded-lg flex items-center justify-between gap-2 transition-colors ${
          selectedServiceId === service.id && selectedPath === null
            ? "bg-primary/10 text-primary"
            : "hover:bg-base-300 text-base-content"
        }`}
      >
        <span className="flex items-center gap-1.5 overflow-hidden text-ellipsis whitespace-nowrap">
          {isExpanded ? (
            <ChevronDown className="w-3.5 h-3.5 shrink-0 text-base-content/60" />
          ) : (
            <ChevronRight className="w-3.5 h-3.5 shrink-0 text-base-content/60" />
          )}
          <Server className={`w-3.5 h-3.5 shrink-0 ${service.status === "running" ? "text-success animate-pulse" : "text-base-content/50"}`} />
          <span className="font-bold truncate">{service.name}</span>
          <span
            className={`w-2 h-2 rounded-full shrink-0 ${
              service.status === "running"
                ? "bg-success"
                : service.status === "sleeping"
                ? "bg-info"
                : service.status === "pending"
                ? "bg-warning"
                : "bg-base-content/30"
            }`}
            title={`Estado: ${service.status}`}
          />
        </span>
      </button>

      {/* Árbol de archivos interno */}
      {showFiles && (
        <div className="pl-2 ml-3.5 space-y-0.5">
          {filesQuery.isLoading ? (
            <div className="p-2 text-[10px] text-base-content/50 italic animate-pulse">Cargando archivos...</div>
          ) : filesQuery.isError ? (
            <div className="p-2 text-[10px] text-error">Error al cargar archivos</div>
          ) : files.length === 0 ? (
            <div className="p-2 text-[10px] text-base-content/50 italic">No hay archivos</div>
          ) : (
            visibleFiles.map((f) => {
              const isSelected = selectedServiceId === service.id && selectedPath === f.path;
              const fullPathKey = `${service.id}::${f.path}`;
              return (
                <div
                  key={f.path}
                  className={`w-full py-0.5 px-1.5 rounded flex items-center justify-between gap-2 transition-colors text-[11px] ${
                    isSelected
                      ? "bg-base-300 font-semibold"
                      : "hover:bg-base-300"
                  }`}
                >
                  <div className="flex items-center gap-1.5 min-w-0 flex-1">
                    <input
                      type="checkbox"
                      className="checkbox checkbox-xs checkbox-neutral shrink-0"
                      checked={selectedFilePaths.has(fullPathKey)}
                      disabled={!isStopped}
                      onChange={(e) => {
                        const checked = e.target.checked;
                        setSelectedFilePaths(prev => {
                          const next = new Set(prev);
                          if (checked) {
                            next.add(fullPathKey);
                          } else {
                            next.delete(fullPathKey);
                          }
                          return next;
                        });
                      }}
                    />
                    <button
                      type="button"
                      onClick={() => {
                        onSelectFile(service, f.path);
                        if (f.is_dir) {
                          togglePath(f.path);
                        }
                      }}
                      className="text-left flex-1 min-w-0 flex items-center gap-1 font-mono text-[11px] text-base-content/90"
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
                          <span className="w-3 shrink-0" />
                          <FileText className="w-3 h-3 shrink-0" />
                        </>
                      )}
                      <span className="truncate">{getFileName(f.path)}</span>
                    </button>
                  </div>
                  <span className={`text-[9px] opacity-70 shrink-0 select-none ${isSelected ? "text-primary font-semibold" : ""}`}>
                    {!f.is_dir && formatSize(f.size)}
                  </span>
                </div>
              );
            })
          )}
        </div>
      )}
    </div>
  );
};

export const FilesPage = () => {
  const { openModal } = useModal();
  const confirm = useConfirmModal();
  const queryClient = useQueryClient();

  const [selectedFile, setSelectedFile] = useState<{
    service: ServiceDto;
    path: string;
  } | null>(null);
  const [editorContent, setEditorContent] = useState("");
  const [originalContent, setOriginalContent] = useState("");
  const [isBinaryFile, setIsBinaryFile] = useState(false);
  const [expandedServices, setExpandedServices] = useState<Set<string>>(new Set());
  const [searchQuery, setSearchQuery] = useState("");
  const [imagePreviewUrl, setImagePreviewUrl] = useState("");
  const [selectedFilePaths, setSelectedFilePaths] = useState<Set<string>>(new Set());

  const isImageFile = (path: string) => {
    const ext = path.toLowerCase().split('.').pop();
    return ["png", "jpg", "jpeg", "gif", "ico", "svg", "webp"].includes(ext || "");
  };

  useEffect(() => {
    let url = "";
    if (selectedFile && isImageFile(selectedFile.path)) {
      filesService.downloadFile(selectedFile.service.id, selectedFile.path)
        .then(blob => {
          url = URL.createObjectURL(blob);
          setImagePreviewUrl(url);
        })
        .catch(() => {
          setImagePreviewUrl("");
        });
    } else {
      setImagePreviewUrl("");
    }
    return () => {
      if (url) {
        URL.revokeObjectURL(url);
      }
    };
  }, [selectedFile]);

  const handleDownload = async () => {
    if (!selectedFile) return;
    try {
      const blob = await filesService.downloadFile(selectedFile.service.id, selectedFile.path);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = getFileName(selectedFile.path);
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      window.URL.revokeObjectURL(url);
    } catch {
      toast.error("Error al descargar el archivo");
    }
  };

  const openRenameModal = () => {
    if (!selectedFile) return;
    const isSvcStopped = selectedFile.service.status === "stopped";
    openModal(
      <RenameModal
        serviceID={selectedFile.service.id}
        isStopped={isSvcStopped}
        currentPath={selectedFile.path}
        onRenamed={(newPath) => {
          queryClient.invalidateQueries({ queryKey: ["pb-files", selectedFile.service.id] });
          setSelectedFile({ ...selectedFile, path: newPath });
        }}
      />,
      { title: "Renombrar / Mover", width: 450 }
    );
  };

  const openNewFolderModal = (service: ServiceDto) => {
    const isSvcStopped = service.status === "stopped";
    openModal(
      <NewFolderModal
        serviceID={service.id}
        isStopped={isSvcStopped}
        onCreated={() => {
          queryClient.invalidateQueries({ queryKey: ["pb-files", service.id] });
        }}
      />,
      { title: `Nueva Carpeta en ${service.name}`, width: 450 }
    );
  };

  const unzipMutation = useMutation({
    mutationFn: filesService.extractZip,
    onSuccess: () => {
      toast.success("Archivo ZIP extraído con éxito");
      if (selectedFile) {
        queryClient.invalidateQueries({ queryKey: ["pb-files", selectedFile.service.id] });
      }
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  const handleExtractZip = () => {
    if (!selectedFile) return;
    unzipMutation.mutate({
      serviceID: selectedFile.service.id,
      path: selectedFile.path,
    });
  };

  const bulkDeleteMutation = useMutation({
    mutationFn: async (paths: string[]) => {
      for (const p of paths) {
        const [serviceId, filePath] = p.split("::");
        await filesService.deleteFile({ serviceID: serviceId, path: filePath });
      }
    },
    onSuccess: () => {
      toast.success("Archivos eliminados con éxito");
      const serviceIds = new Set(Array.from(selectedFilePaths).map(p => p.split("::")[0]));
      serviceIds.forEach(id => {
        queryClient.invalidateQueries({ queryKey: ["pb-files", id] });
      });
      setSelectedFilePaths(new Set());
      setSelectedFile(null);
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  const handleBulkDelete = async () => {
    if (selectedFilePaths.size === 0) return;
    const ok = await confirm(
      "Eliminar Archivos Seleccionados",
      `¿Estás seguro de que deseas eliminar permanentemente los ${selectedFilePaths.size} archivos/carpetas seleccionados?`
    );
    if (ok) {
      bulkDeleteMutation.mutate(Array.from(selectedFilePaths));
    }
  };

  // Consulta global de servicios (instancias)
  const servicesQuery = useQuery<ServiceDto[]>({
    queryKey: ["services"],
    queryFn: serviceService.fetchAllServices,
    refetchInterval: 3000,
  });

  const services = servicesQuery.data ?? [];

  // Mantener actualizado el objeto del servicio seleccionado para capturar cambios de estado en tiempo real
  const activeService = selectedFile
    ? services.find((s) => s.id === selectedFile.service.id) || selectedFile.service
    : null;

  const isDirSelected =
    selectedFile != null &&
    queryClient
      .getQueryData<PBFileEntry[]>(["pb-files", selectedFile.service.id])
      ?.find((f) => f.path === selectedFile.path)?.is_dir === true;

  const fileContentQuery = useQuery({
    queryKey: ["pb-file-content", selectedFile?.service.id, selectedFile?.path],
    queryFn: () =>
      filesService.readFile(selectedFile?.service.id || "", selectedFile?.path || ""),
    enabled: selectedFile != null && !isBinaryFile && !isDirSelected,
  });

  useEffect(() => {
    if (selectedFile) {
      const isBinary =
        !isDirSelected &&
        (selectedFile.path.endsWith(".db") ||
          selectedFile.path.endsWith(".png") ||
          selectedFile.path.endsWith(".jpg") ||
          selectedFile.path.endsWith(".jpeg") ||
          selectedFile.path.endsWith(".gif") ||
          selectedFile.path.endsWith(".ico") ||
          selectedFile.path.endsWith(".zip"));
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
  }, [selectedFile, isDirSelected]);

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
      queryClient.invalidateQueries({ queryKey: ["services"] });
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  const saveMutation = useMutation({
    mutationFn: filesService.saveFile,
    onSuccess: () => {
      toast.success("Archivo guardado con éxito");
      setOriginalContent(editorContent);
      if (selectedFile) {
        queryClient.invalidateQueries({ queryKey: ["pb-files", selectedFile.service.id] });
      }
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  const deleteMutation = useMutation({
    mutationFn: filesService.deleteFile,
    onSuccess: () => {
      toast.success("Archivo eliminado con éxito");
      if (selectedFile) {
        queryClient.invalidateQueries({ queryKey: ["pb-files", selectedFile.service.id] });
      }
      setSelectedFile(null);
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  const toggleServiceExpand = (serviceId: string) => {
    setExpandedServices((prev) => {
      const next = new Set(prev);
      if (next.has(serviceId)) {
        next.delete(serviceId);
      } else {
        next.add(serviceId);
      }
      return next;
    });
  };

  if (servicesQuery.isError) {
    return <ErrorFallback error={servicesQuery.error} onRetry={servicesQuery.refetch} />;
  }

  const isStopped = activeService?.status === "stopped";
  const hasChanges = editorContent !== originalContent;

  const handleStartService = (serviceId: string) => {
    commandMutation.mutate({ service_id: serviceId, action: "start" });
  };

  const handleStopService = (serviceId: string) => {
    commandMutation.mutate({ service_id: serviceId, action: "stop" });
  };

  const handleSave = () => {
    if (!selectedFile) return;
    saveMutation.mutate({
      serviceID: selectedFile.service.id,
      path: selectedFile.path,
      content: editorContent,
    });
  };

  const handleDelete = async () => {
    if (!selectedFile) return;
    const ok = await confirm(
      "Eliminar Archivo",
      `¿Estás seguro de que deseas eliminar permanentemente el archivo ${selectedFile.path}?`
    );
    if (ok) {
      deleteMutation.mutate({ serviceID: selectedFile.service.id, path: selectedFile.path });
    }
  };

  const openNewFileModal = (service: ServiceDto) => {
    const isSvcStopped = service.status === "stopped";
    openModal(
      <NewFileModal
          serviceID={service.id}
          isStopped={isSvcStopped}
          onCreated={(newPath) => {
            queryClient.invalidateQueries({ queryKey: ["pb-files", service.id] });
            setSelectedFile({ service, path: newPath });
          }}
      />,
      { title: `Nuevo Archivo en ${service.name}`, width: 450 }
    );
  };


  const openUploadFilesModal = (service: ServiceDto) => {
    const isSvcStopped = service.status === "stopped";
    openModal(
      <UploadFilesModal
        serviceID={service.id}
        isStopped={isSvcStopped}
        onUploaded={() => {
          queryClient.invalidateQueries({ queryKey: ["pb-files", service.id] });
        }}
      />,
      { title: `Subir Archivos a ${service.name}`, width: 480 }
    );
  };

  const showDbWarning =
    selectedFile?.path?.endsWith(".db") || selectedFile?.path?.includes("pb_data");

  return (
    <div className="space-y-4">
      {/* Contenedor Principal */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-4 h-[calc(100vh-8rem)] min-h-[500px]">
        {/* Explorador de Archivos Global (Col 4) */}
        <div className="lg:col-span-4 flex flex-col bg-base-200 border border-base-300 rounded-xl overflow-hidden h-full min-h-0">
          <div className="p-3 bg-base-300 font-semibold text-xs uppercase tracking-wider border-b border-base-300 flex justify-between items-center shrink-0">
            <span>Archivos de Instancias</span>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => {
                  queryClient.invalidateQueries({ queryKey: ["services"] });
                  if (selectedFile) {
                    queryClient.invalidateQueries({ queryKey: ["pb-files", selectedFile.service.id] });
                  }
                }}
                className="btn btn-xs btn-ghost btn-circle"
                title="Actualizar todo"
              >
                <RefreshCw className="w-3.5 h-3.5" />
              </button>
              <span className="badge badge-sm badge-neutral">{services.length} instancias</span>
            </div>
          </div>

          {/* Buscador de archivos en tiempo real */}
          <div className="p-2 border-b border-base-300 bg-base-100 shrink-0">
            <div className="relative">
              <Search className="w-3.5 h-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-base-content/40" />
              <input
                type="text"
                className="input input-xs input-bordered w-full pl-8 font-mono text-[11px]"
                placeholder="Buscar archivos..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
            </div>
          </div>

          {/* Panel de acciones por lotes */}
          {selectedFilePaths.size > 0 && (
            <div className="p-2 border-b border-base-300 bg-base-300/50 flex justify-between items-center shrink-0">
              <span className="text-[10px] font-semibold text-base-content/75">
                {selectedFilePaths.size} seleccionados (acciones en ventana derecha)
              </span>
            </div>
          )}

          <div className="flex-1 overflow-y-auto p-2 space-y-1 font-mono text-xs min-h-0">
            {servicesQuery.isLoading ? (
              <div className="p-4 text-center text-base-content/50 animate-pulse">Cargando instancias...</div>
            ) : services.length === 0 ? (
              <div className="p-4 text-center text-base-content/50">No hay instancias configuradas.</div>
            ) : (
              services.map((service) => {
                const isServiceStopped = service.status === "stopped";
                return (
                  <div key={service.id} className="space-y-0.5 border border-base-300/40 rounded-lg p-1 bg-base-100/50">
                    <ServiceFileTree
                      service={service}
                      isExpanded={expandedServices.has(service.id)}
                      onToggleExpand={() => toggleServiceExpand(service.id)}
                      selectedServiceId={selectedFile?.service.id}
                      selectedPath={selectedFile?.path || null}
                      onSelectFile={(svc, path) => setSelectedFile({ service: svc, path })}
                      searchQuery={searchQuery}
                      selectedFilePaths={selectedFilePaths}
                      setSelectedFilePaths={setSelectedFilePaths}
                      isStopped={isServiceStopped}
                    />
                    {expandedServices.has(service.id) && (
                      <div className="pl-4 pr-1 py-1 flex flex-col gap-1 border-t border-base-300/30 mt-1">
                        {/* Botones de operaciones de archivos */}
                        <button
                          type="button"
                          onClick={() => openNewFileModal(service)}
                          className="btn btn-xs btn-ghost gap-1 w-full justify-start text-[10px] h-6 min-h-6 opacity-75 hover:opacity-100"
                          disabled={!isServiceStopped}
                        >
                          <Plus className="w-3 h-3 text-primary" />
                          Nuevo Archivo
                        </button>
                        <button
                          type="button"
                          onClick={() => openNewFolderModal(service)}
                          className="btn btn-xs btn-ghost gap-1 w-full justify-start text-[10px] h-6 min-h-6 opacity-75 hover:opacity-100"
                          disabled={!isServiceStopped}
                        >
                          <Plus className="w-3 h-3 text-secondary" />
                          Nueva Carpeta
                        </button>
                        <button
                          type="button"
                          onClick={() => openUploadFilesModal(service)}
                          className="btn btn-xs btn-ghost gap-1.5 w-full justify-start text-[10px] h-6 min-h-6 opacity-75 hover:opacity-100"
                          disabled={!isServiceStopped}
                        >
                          <Upload className="w-3 h-3 text-info" />
                          Subir Archivos
                        </button>
                      </div>
                    )}
                  </div>
                );
              })
            )}
          </div>
        </div>

        {/* Editor de Código (Col 8) */}
        <div className="lg:col-span-8 flex flex-col bg-base-200 border border-base-300 rounded-xl overflow-hidden h-full min-h-0">
          {selectedFile == null ? (
            selectedFilePaths.size > 0 ? (
              <div className="flex-1 flex flex-col items-center justify-center text-center p-6 text-base-content/60 space-y-4">
                <Trash2 className="w-12 h-12 stroke-1 text-error/80" />
                <div className="space-y-1">
                  <p className="font-bold text-sm text-base-content">Selección múltiple activa ({selectedFilePaths.size} elementos)</p>
                  <p className="text-xs max-w-xs text-base-content/60">Haz clic en el botón inferior para eliminar de forma permanente la selección.</p>
                </div>
                <button
                  type="button"
                  onClick={handleBulkDelete}
                  className="btn btn-sm btn-error gap-1.5 font-semibold"
                  disabled={bulkDeleteMutation.isPending}
                >
                  <Trash2 className="w-4 h-4" />
                  Eliminar Selección ({selectedFilePaths.size})
                </button>
              </div>
            ) : (
              <div className="flex-1 flex flex-col items-center justify-center text-center p-6 text-base-content/60 space-y-2">
                <FolderOpen className="w-12 h-12 stroke-1 text-base-content/40" />
                <p className="text-sm">Selecciona un archivo del explorador lateral para comenzar a visualizarlo o editarlo.</p>
              </div>
            )
          ) : (
            <div className="flex-grow flex flex-col min-h-0">
              {/* Encabezado del Editor */}
              <div className="p-3 bg-base-300 border-b border-base-300 flex flex-wrap justify-between items-center gap-2">
                <div className="flex flex-col min-w-0">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-base-content/50">
                    Instancia: {selectedFile.service.name}
                  </span>
                  <span className="font-mono text-xs font-semibold text-primary truncate max-w-xs md:max-w-md" title={selectedFile.path}>
                    {selectedFile.path}
                  </span>
                </div>

                 <div className="flex flex-wrap gap-1.5">
                  {/* Botones de Control de Servicio e Instancia */}
                  {activeService && (
                    <>
                      {activeService.status === "stopped" ? (
                        <button
                          type="button"
                          onClick={() => handleStartService(activeService.id)}
                          className="btn btn-xs btn-success gap-1"
                          disabled={commandMutation.isPending || (activeService.status as string) === "pending"}
                        >
                          <Play className="w-2.5 h-2.5 fill-current" />
                          Iniciar Servicio
                        </button>
                      ) : (
                        <button
                          type="button"
                          onClick={() => handleStopService(activeService.id)}
                          className="btn btn-xs btn-error gap-1"
                          disabled={commandMutation.isPending || (activeService.status as string) === "pending"}
                        >
                          <Square className="w-2.5 h-2.5 fill-current" />
                          Detener Servicio
                        </button>
                      )}
                      <button
                        type="button"
                        onClick={() => {
                          queryClient.invalidateQueries({ queryKey: ["pb-files", activeService.id] });
                          queryClient.invalidateQueries({ queryKey: ["services"] });
                        }}
                        className="btn btn-xs btn-neutral gap-1"
                        title="Recargar archivos de esta instancia"
                      >
                        <RefreshCw className="w-2.5 h-2.5" />
                        Recargar
                      </button>
                    </>
                  )}

                  {!isDirSelected && (
                    <button
                      type="button"
                      className="btn btn-xs btn-neutral gap-1"
                      onClick={handleDownload}
                    >
                      <Download className="w-3 h-3" />
                      Descargar
                    </button>
                  )}

                  {selectedFile.path.toLowerCase().endsWith(".zip") && (
                    <button
                      type="button"
                      className="btn btn-xs btn-warning gap-1"
                      disabled={!isStopped || unzipMutation.isPending}
                      onClick={handleExtractZip}
                    >
                      {unzipMutation.isPending ? "Extrayendo..." : "Descomprimir"}
                    </button>
                  )}

                  <button
                    type="button"
                    className="btn btn-xs btn-neutral gap-1"
                    disabled={!isStopped}
                    onClick={openRenameModal}
                  >
                    Renombrar
                  </button>

                  {selectedFilePaths.size > 0 ? (
                    <button
                      type="button"
                      className="btn btn-xs btn-error gap-1 animate-pulse"
                      disabled={bulkDeleteMutation.isPending}
                      onClick={handleBulkDelete}
                    >
                      <Trash2 className="w-3 h-3" />
                      Borrar Selección ({selectedFilePaths.size})
                    </button>
                  ) : (
                    <button
                      type="button"
                      className="btn btn-xs btn-error gap-1"
                      disabled={!isStopped || deleteMutation.isPending}
                      onClick={handleDelete}
                    >
                      <Trash2 className="w-3 h-3" />
                      Borrar
                    </button>
                  )}

                  {!isDirSelected && !isBinaryFile && !isImageFile(selectedFile.path) && (
                    <button
                      type="button"
                      className="btn btn-xs btn-primary gap-1"
                      disabled={!isStopped || !hasChanges || saveMutation.isPending}
                      onClick={handleSave}
                    >
                      <Save className="w-3 h-3" />
                      Guardar
                    </button>
                  )}
                </div>
              </div>

              {/* Advertencia sobre pb_data o archivos .db */}
              {showDbWarning && (
                <div className="alert alert-error rounded-none text-xs flex gap-2 py-2">
                  <AlertTriangle className="w-4 h-4 shrink-0 text-error-content" />
                  <div>
                    <span className="font-semibold">ADVERTENCIA CRÍTICA:</span> Este archivo pertenece a{" "}
                    <span className="font-mono">pb_data</span> o es una base de datos. Escribir aquí puede{" "}
                    <span className="font-semibold">CORROMPER</span> tus datos permanentemente.
                  </div>
                </div>
              )}

              {/* Advertencia de Estado Activo y Botón de Apagado Rápido */}
              {!isStopped && activeService && (
                <div className="alert alert-warning rounded-none text-xs flex justify-between py-2 gap-3 items-center">
                  <div className="flex gap-2 items-center min-w-0">
                    <AlertTriangle className="w-4 h-4 shrink-0 text-warning-content" />
                    <span className="truncate">
                      La instancia <strong>{activeService.name}</strong> ({activeService.status}) debe estar detenida para guardar cambios.
                    </span>
                  </div>
                  <button
                    type="button"
                    onClick={() => handleStopService(activeService.id)}
                    className="btn btn-xs btn-warning shrink-0"
                    disabled={commandMutation.isPending || activeService.status === "pending"}
                  >
                    {commandMutation.isPending ? "Deteniendo..." : "Detener Ahora"}
                  </button>
                </div>
              )}

              {/* Área del Editor de Código */}
              <div className="flex-1 overflow-hidden min-h-0 bg-base-100">
                {isDirSelected ? (
                  <div className="w-full h-full flex flex-col items-center justify-center text-center p-6 text-base-content/60 space-y-2">
                    <FolderOpen className="w-12 h-12 text-amber-500 stroke-1" />
                    <p className="font-semibold text-sm">Directorio seleccionado ({getFileName(selectedFile.path)})</p>
                    <p className="text-xs max-w-md">Para agregar nuevos archivos dentro de esta carpeta, usa el botón "Nuevo Archivo" o "Nueva Carpeta" del explorador lateral.</p>
                  </div>
                ) : imagePreviewUrl ? (
                  <div className="w-full h-full flex items-center justify-center p-6 bg-base-300 overflow-auto">
                    <img
                      src={imagePreviewUrl}
                      alt={getFileName(selectedFile.path)}
                      className="max-w-full max-h-full object-contain rounded-lg shadow-xl border border-base-300"
                    />
                  </div>
                ) : isBinaryFile ? (
                  <div className="w-full h-full flex flex-col items-center justify-center text-center p-6 text-base-content/60 space-y-2">
                    <AlertTriangle className="w-10 h-10 text-error" />
                    <p className="font-semibold text-sm">Este es un archivo binario o base de datos ({getFileName(selectedFile.path)})</p>
                    <p className="text-xs max-w-md">No es posible renderizar archivos binarios como texto.</p>
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


