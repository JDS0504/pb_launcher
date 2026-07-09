import { useState, type FormEvent } from "react";
import toast from "react-hot-toast";
import { Button } from "../../../components/buttons/Button";
import { useModal } from "../../../components/modal/hook";

type Props = {
  defaultName?: string;
  description: string;
  label: string;
  submitLabel: string;
  emptyMessage: string;
  onSubmit: (name: string, comment: string) => Promise<void> | void;
};

export const SnapshotNameForm = ({
  defaultName = "",
  description,
  label,
  submitLabel,
  emptyMessage,
  onSubmit,
}: Props) => {
  const { closeModal } = useModal();
  const [name, setName] = useState(defaultName);
  const [comment, setComment] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const trimmedName = name.trim();
    if (trimmedName === "") {
      toast.error(emptyMessage);
      return;
    }

    setIsSubmitting(true);
    try {
      await onSubmit(trimmedName, comment.trim());
      closeModal();
    } catch {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      <div className="text-sm text-base-content/80">{description}</div>
      <div className="form-control w-full">
        <label className="label">
          <span className="label-text mb-1">{label}</span>
        </label>
        <input
          className="input input-md input-bordered w-full focus:outline-none focus:ring-1 focus:ring-primary"
          value={name}
          onChange={event => setName(event.target.value)}
          autoComplete="off"
          autoFocus
        />
      </div>
      <div className="form-control w-full">
        <label className="label">
          <span className="label-text mb-1">Comentario <span className="text-base-content/40">(opcional)</span></span>
        </label>
        <textarea
          className="textarea textarea-bordered w-full focus:outline-none focus:ring-1 focus:ring-primary resize-none"
          rows={2}
          placeholder="¿Qué cambios incluye este snapshot?"
          value={comment}
          onChange={event => setComment(event.target.value)}
        />
      </div>
      <Button
        type="submit"
        label={submitLabel}
        loading={isSubmitting}
        disabled={name.trim() === ""}
      />
    </form>
  );
};
