import { useState, type FC } from "react";
import { useMutation } from "@tanstack/react-query";
import toast from "react-hot-toast";
import { filesService } from "../../../services/files";
import { useModal } from "../../../components/modal/hook";
import { getErrorMessage } from "../../../utils/errors";

type NewFolderModalProps = {
  serviceID: string;
  isStopped: boolean;
  onCreated: (path: string) => void;
};

export const NewFolderModal: FC<NewFolderModalProps> = ({ serviceID, isStopped, onCreated }) => {
  const { closeModal } = useModal();
  const [folder, setFolder] = useState("pb_public");
  const [folderPath, setFolderPath] = useState("");

  const createFolderMutation = useMutation({
    mutationFn: filesService.createFolder,
    onSuccess: (_, variables) => {
      toast.success("Carpeta creada con éxito");
      onCreated(variables.path);
      closeModal();
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const cleanPath = folderPath.trim();
    if (!cleanPath) {
      toast.error("La ruta es obligatoria");
      return;
    }
    const finalPath = `${folder}/${cleanPath.startsWith("/") ? cleanPath.substring(1) : cleanPath}`;
    createFolderMutation.mutate({
      serviceID,
      path: finalPath,
    });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4 text-sm">
      {!isStopped && (
        <div className="alert alert-warning text-xs">
          Debes detener la instancia antes de poder crear una nueva carpeta.
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
          <span className="label-text mb-1">Nombre / Ruta de la carpeta</span>
        </label>
        <input
          type="text"
          className="input input-bordered input-sm w-full font-mono text-xs"
          placeholder="ej: assets, images/banners"
          value={folderPath}
          onChange={(e) => setFolderPath(e.target.value)}
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
          disabled={!isStopped || createFolderMutation.isPending || folderPath.trim() === ""}
        >
          Crear Carpeta
        </button>
      </div>
    </form>
  );
};
