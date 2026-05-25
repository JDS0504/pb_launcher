import { useState, type FC } from "react";
import { useMutation } from "@tanstack/react-query";
import toast from "react-hot-toast";
import { filesService } from "../../../services/files";
import { useModal } from "../../../components/modal/hook";
import { getErrorMessage } from "../../../utils/errors";

type NewFileModalProps = {
  serviceID: string;
  isStopped: boolean;
  onCreated: (path: string) => void;
};

export const NewFileModal: FC<NewFileModalProps> = ({ serviceID, isStopped, onCreated }) => {
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
    onError: (error) => toast.error(getErrorMessage(error)),
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
          <span className="label-text mb-1">Nombre / Ruta relativa del archivo</span>
        </label>
        <input
          type="text"
          className="input input-bordered input-sm w-full font-mono text-xs"
          placeholder="ej: index.html, subcarpeta/styles.css"
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
