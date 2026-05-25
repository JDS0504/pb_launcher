import { useState, type FC } from "react";
import { useMutation } from "@tanstack/react-query";
import toast from "react-hot-toast";
import { Upload, Trash2 } from "lucide-react";
import { filesService } from "../../../services/files";
import { useModal } from "../../../components/modal/hook";
import { getErrorMessage } from "../../../utils/errors";

type UploadFilesModalProps = {
  serviceID: string;
  isStopped: boolean;
  onUploaded: () => void;
};

const formatSize = (size: number) => {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / 1024 / 1024).toFixed(1)} MB`;
};

export const UploadFilesModal: FC<UploadFilesModalProps> = ({
  serviceID,
  isStopped,
  onUploaded,
}) => {
  const { closeModal } = useModal();
  const [folder, setFolder] = useState("pb_public");
  const [subfolder, setSubfolder] = useState("");
  const [files, setFiles] = useState<File[]>([]);
  const [isDragActive, setIsDragActive] = useState(false);

  const uploadMutation = useMutation({
    mutationFn: (variables: { destPath: string; files: File[] }) =>
      filesService.uploadFiles(serviceID, variables.destPath, variables.files),
    onSuccess: () => {
      toast.success("Archivos subidos con éxito");
      onUploaded();
      closeModal();
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  const handleDrag = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === "dragenter" || e.type === "dragover") {
      setIsDragActive(true);
    } else if (e.type === "dragleave") {
      setIsDragActive(false);
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragActive(false);
    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      const droppedFiles = Array.from(e.dataTransfer.files);
      setFiles((prev) => [...prev, ...droppedFiles]);
    }
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      const selectedFiles = Array.from(e.target.files);
      setFiles((prev) => [...prev, ...selectedFiles]);
    }
  };

  const handleRemoveFile = (index: number) => {
    setFiles((prev) => prev.filter((_, i) => i !== index));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (files.length === 0) {
      toast.error("Debe seleccionar al menos un archivo");
      return;
    }
    const cleanSub = subfolder.trim().replace(/^\/|\/$/g, ""); // remove leading/trailing slashes
    const destPath = cleanSub ? `${folder}/${cleanSub}` : folder;

    uploadMutation.mutate({ destPath, files });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4 text-sm">
      {!isStopped && (
        <div className="alert alert-warning text-xs">
          Debes detener la instancia antes de poder subir archivos.
        </div>
      )}

      <div className="grid grid-cols-2 gap-2">
        <div className="form-control w-full">
          <label className="label">
            <span className="label-text mb-1">Directorio de Destino</span>
          </label>
          <select
            className="select select-bordered select-sm w-full font-mono text-xs"
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
            <span className="label-text mb-1">Subcarpeta (Opcional)</span>
          </label>
          <input
            type="text"
            className="input input-bordered input-sm w-full font-mono text-xs"
            placeholder="ej: images, css/subfolder"
            value={subfolder}
            onChange={(e) => setSubfolder(e.target.value)}
            disabled={!isStopped}
          />
        </div>
      </div>

      {/* Zona de Drop */}
      <div
        onDragEnter={handleDrag}
        onDragOver={handleDrag}
        onDragLeave={handleDrag}
        onDrop={handleDrop}
        className={`border-2 border-dashed rounded-lg p-6 text-center transition-colors cursor-pointer flex flex-col items-center justify-center min-h-[120px] ${
          isDragActive
            ? "border-primary bg-primary/5"
            : "border-base-content/20 hover:border-primary/50"
        } ${!isStopped ? "opacity-50 pointer-events-none" : ""}`}
        onClick={() => {
          if (isStopped) {
            document.getElementById("multiple-file-input")?.click();
          }
        }}
      >
        <input
          id="multiple-file-input"
          type="file"
          multiple
          className="hidden"
          onChange={handleFileChange}
          disabled={!isStopped}
        />
        <Upload className="w-8 h-8 text-base-content/40 mb-2" />
        <p className="font-semibold text-xs">Arrastra tus archivos aquí o haz clic para seleccionarlos</p>
        <p className="text-[10px] text-base-content/50 mt-1">Sube uno o varios archivos simultáneamente</p>
      </div>

      {/* Lista de archivos a subir */}
      {files.length > 0 && (
        <div className="space-y-1.5 max-h-[150px] overflow-y-auto border border-base-content/10 rounded p-2 bg-base-100 font-mono text-[11px]">
          <div className="font-bold border-b border-base-content/10 pb-1 mb-1 text-[10px] text-base-content/60 flex justify-between">
            <span>Archivos seleccionados ({files.length})</span>
            <button
              type="button"
              className="text-error hover:underline text-[9px]"
              onClick={() => setFiles([])}
            >
              Limpiar todos
            </button>
          </div>
          {files.map((file, idx) => (
            <div key={idx} className="flex justify-between items-center py-0.5 border-b border-base-content/5 last:border-0">
              <span className="truncate max-w-[250px]" title={file.name}>
                {file.name}
              </span>
              <div className="flex items-center gap-2">
                <span className="text-[9px] opacity-60">{formatSize(file.size)}</span>
                <button
                  type="button"
                  className="text-error hover:text-error-hover"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleRemoveFile(idx);
                  }}
                >
                  <Trash2 className="w-3 h-3" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="flex justify-end gap-2 pt-2">
        <button type="button" className="btn btn-sm btn-ghost" onClick={closeModal}>
          Cancelar
        </button>
        <button
          type="submit"
          className="btn btn-sm btn-primary"
          disabled={!isStopped || uploadMutation.isPending || files.length === 0}
        >
          {uploadMutation.isPending ? "Subiendo..." : `Subir ${files.length} archivos`}
        </button>
      </div>
    </form>
  );
};
