import { useState, type FormEvent } from "react";
import toast from "react-hot-toast";
import { Button } from "../../../components/buttons/Button";
import { useModal } from "../../../components/modal/hook";
import type { SnapshotInfo } from "../../../services/backup";

type Props = {
  snapshots: SnapshotInfo[];
  initialSnapshot?: SnapshotInfo;
  defaultInstanceName?: string;
  onSubmit: (snapshot: SnapshotInfo, instanceName: string) => Promise<void> | void;
};

const formatSize = (size: number) => {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / 1024 / 1024).toFixed(1)} MB`;
};

export const RestoreSnapshotForm = ({
  snapshots,
  initialSnapshot,
  defaultInstanceName = "",
  onSubmit,
}: Props) => {
  const { closeModal } = useModal();
  const [selectedId, setSelectedId] = useState<string>(
    initialSnapshot?.id ?? snapshots[0]?.id ?? "",
  );
  const [instanceName, setInstanceName] = useState(defaultInstanceName);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const selectedSnapshot = snapshots.find(s => s.id === selectedId);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const trimmedName = instanceName.trim();
    if (trimmedName === "") {
      toast.error("Ingresa un nombre para la nueva instancia");
      return;
    }
    if (!selectedSnapshot) {
      toast.error("Selecciona un snapshot");
      return;
    }
    setIsSubmitting(true);
    try {
      await onSubmit(selectedSnapshot, trimmedName);
      closeModal();
    } catch {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      <p className="text-sm text-base-content/80">
        Selecciona el snapshot a restaurar y asigna un nombre a la nueva instancia.
        La instancia actual no se modifica.
      </p>

      {/* Select de snapshot */}
      <div className="form-control w-full">
        <label className="label">
          <span className="label-text mb-1">Snapshot a restaurar</span>
        </label>
        <select
          className="select select-bordered w-full focus:outline-none focus:ring-1 focus:ring-primary"
          value={selectedId}
          onChange={e => setSelectedId(e.target.value)}
        >
          {snapshots.map(s => (
            <option key={s.id} value={s.id}>
              {s.name} — {s.version} — {new Date(s.created_at).toLocaleString()} ({formatSize(s.size)})
            </option>
          ))}
        </select>
      </div>

      {/* Nombre de la nueva instancia */}
      <div className="form-control w-full">
        <label className="label">
          <span className="label-text mb-1">Nombre de la nueva instancia</span>
        </label>
        <input
          className="input input-md input-bordered w-full focus:outline-none focus:ring-1 focus:ring-primary"
          value={instanceName}
          onChange={e => setInstanceName(e.target.value)}
          autoComplete="off"
          autoFocus
          placeholder="Ej: mi-instancia-restaurada"
        />
      </div>

      <Button
        type="submit"
        label="Restaurar snapshot"
        loading={isSubmitting}
        disabled={instanceName.trim() === "" || !selectedSnapshot}
      />
    </form>
  );
};
