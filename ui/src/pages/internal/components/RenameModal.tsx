import { useState, type FC } from "react";
import { useMutation } from "@tanstack/react-query";
import toast from "react-hot-toast";
import { filesService } from "../../../services/files";
import { useModal } from "../../../components/modal/hook";
import { getErrorMessage } from "../../../utils/errors";

type RenameModalProps = {
  serviceID: string;
  isStopped: boolean;
  currentPath: string;
  onRenamed: (newPath: string) => void;
};

export const RenameModal: FC<RenameModalProps> = ({
  serviceID,
  isStopped,
  currentPath,
  onRenamed,
}) => {
  const { closeModal } = useModal();
  const [newPath, setNewPath] = useState(currentPath);

  const renameMutation = useMutation({
    mutationFn: filesService.renameFile,
    onSuccess: () => {
      toast.success("Archivo/Carpeta renombrado con éxito");
      onRenamed(newPath);
      closeModal();
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const cleanPath = newPath.trim();
    if (!cleanPath) {
      toast.error("La nueva ruta es obligatoria");
      return;
    }
    if (cleanPath === currentPath) {
      toast.error("La ruta no ha cambiado");
      return;
    }
    renameMutation.mutate({
      serviceID,
      oldPath: currentPath,
      newPath: cleanPath,
    });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4 text-sm">
      {!isStopped && (
        <div className="alert alert-warning text-xs">
          Debes detener la instancia antes de poder renombrar o mover archivos/carpetas.
        </div>
      )}

      <div className="form-control w-full">
        <label className="label">
          <span className="label-text mb-1">Ruta actual</span>
        </label>
        <input
          type="text"
          className="input input-bordered input-sm w-full font-mono text-xs opacity-60 bg-base-200"
          value={currentPath}
          disabled
        />
      </div>

      <div className="form-control w-full">
        <label className="label">
          <span className="label-text mb-1">Nueva ruta relativa</span>
        </label>
        <input
          type="text"
          className="input input-bordered input-sm w-full font-mono text-xs"
          value={newPath}
          onChange={(e) => setNewPath(e.target.value)}
          disabled={!isStopped}
          required
        />
        <label className="label">
          <span className="label-text-alt text-base-content/50 mt-1">
            Puedes cambiar el nombre o mover el archivo cambiando la carpeta contenedora.
          </span>
        </label>
      </div>

      <div className="flex justify-end gap-2 pt-2">
        <button type="button" className="btn btn-sm btn-ghost" onClick={closeModal}>
          Cancelar
        </button>
        <button
          type="submit"
          className="btn btn-sm btn-primary"
          disabled={!isStopped || renameMutation.isPending || newPath.trim() === currentPath}
        >
          Renombrar
        </button>
      </div>
    </form>
  );
};
