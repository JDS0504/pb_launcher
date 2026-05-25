import { useState, type FormEvent } from "react";
import toast from "react-hot-toast";
import { Button } from "../../../components/buttons/Button";
import { useModal } from "../../../components/modal/hook";

type Props = {
  description: string;
  label: string;
  submitLabel: string;
  onSubmit: (password: string) => Promise<void> | void;
};

export const ConfirmAdminPasswordForm = ({
  description,
  label,
  submitLabel,
  onSubmit,
}: Props) => {
  const { closeModal } = useModal();
  const [password, setPassword] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const trimmed = password.trim();
    if (trimmed === "") {
      toast.error("Por favor ingresa la contraseña");
      return;
    }

    setIsSubmitting(true);
    try {
      await onSubmit(trimmed);
      closeModal();
    } catch {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-5 w-[320px] max-w-full">
      <div className="text-sm text-base-content/85 leading-relaxed">{description}</div>
      <div className="form-control w-full">
        <label className="label">
          <span className="label-text mb-1 font-semibold">{label}</span>
        </label>
        <input
          type="password"
          className="input input-md input-bordered w-full focus:outline-none focus:ring-1 focus:ring-primary"
          value={password}
          onChange={event => setPassword(event.target.value)}
          autoComplete="off"
          autoFocus
        />
      </div>
      <div className="flex justify-end pt-2">
        <Button
          type="submit"
          label={submitLabel}
          loading={isSubmitting}
          disabled={password.trim() === ""}
        />
      </div>
    </form>
  );
};
