import { useState, type ChangeEvent, type FormEvent } from "react";
import { useMutation } from "@tanstack/react-query";
import toast from "react-hot-toast";
import { Button } from "../../../components/buttons/Button";
import { useModal } from "../../../components/modal/hook";
import { backupService } from "../../../services/backup";
import { getErrorMessage } from "../../../utils/errors";

type Props = {
  onRestore?: () => void;
};

export const RestoreBackupForm = ({ onRestore }: Props) => {
  const { closeModal } = useModal();
  const [file, setFile] = useState<File | null>(null);
  const [name, setName] = useState("");

  const restoreMutation = useMutation({
    mutationFn: backupService.restoreBackup,
    onSuccess: () => {
      toast.success("Snapshot importado exitosamente");
      closeModal();
      onRestore?.();
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const handleFileChange = (event: ChangeEvent<HTMLInputElement>) => {
    setFile(event.target.files?.[0] ?? null);
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (file == null) {
      toast.error("Selecciona un archivo de snapshot (.zip)");
      return;
    }
    const trimmedName = name.trim();
    if (trimmedName === "") {
      toast.error("Ingresa un nombre para la instancia");
      return;
    }
    restoreMutation.mutate({ file, name: trimmedName });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      <div className="text-sm text-base-content/80">
        Crea una nueva instancia PocketBase importando un archivo ZIP de snapshot previamente exportado.
      </div>
      <div className="form-control w-full">
        <label className="label">
          <span className="label-text mb-1">Nombre de la nueva instancia</span>
        </label>
        <input
          className="input input-md input-bordered w-full focus:outline-none focus:ring-1 focus:ring-primary"
          value={name}
          onChange={event => {
            const sanitized = event.target.value
              .toLowerCase()
              .replace(/[^a-z0-9-]/g, "-")
              .replace(/-+/g, "-");
            setName(sanitized);
          }}
          placeholder="ej. mi-app-migrada"
          autoComplete="off"
        />
      </div>
      <input
        type="file"
        accept=".zip,application/zip"
        className="file-input file-input-bordered w-full"
        onChange={handleFileChange}
      />
      <Button
        type="submit"
        label="Importar Snapshot"
        loading={restoreMutation.isPending}
        disabled={file == null || name.trim() === ""}
      />
    </form>
  );
};
