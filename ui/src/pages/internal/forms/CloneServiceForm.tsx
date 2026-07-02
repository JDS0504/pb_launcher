import { useState, type FormEvent } from "react";
import { useMutation } from "@tanstack/react-query";
import toast from "react-hot-toast";
import { Button } from "../../../components/buttons/Button";
import { useModal } from "../../../components/modal/hook";
import { backupService } from "../../../services/backup";
import type { ServiceDto } from "../../../services/services";
import { getErrorMessage } from "../../../utils/errors";

type Props = {
  service: ServiceDto;
  onClone?: () => void;
};

export const CloneServiceForm = ({ service, onClone }: Props) => {
  const { closeModal } = useModal();
  const [name, setName] = useState(`${service.name}-clone`);

  const cloneMutation = useMutation({
    mutationFn: backupService.cloneService,
    onSuccess: () => {
      toast.success("Service cloned successfully");
      closeModal();
      onClone?.();
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const trimmedName = name.trim();
    if (trimmedName === "") {
      toast.error("Enter an instance name");
      return;
    }
    cloneMutation.mutate({ serviceID: service.id, name: trimmedName });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      <div className="text-sm text-base-content/80">
        Clone creates a new instance using the same PocketBase version and data.
      </div>
      <div className="form-control w-full">
        <label className="label">
          <span className="label-text mb-1">Instance Name</span>
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
          autoComplete="off"
        />
      </div>
      <Button
        type="submit"
        label="Clone"
        loading={cloneMutation.isPending}
        disabled={name.trim() === ""}
      />
    </form>
  );
};
