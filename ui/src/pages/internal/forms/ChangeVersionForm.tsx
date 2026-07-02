import { useMemo, useState, type FC, type FormEvent } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import toast from "react-hot-toast";
import { Button } from "../../../components/buttons/Button";
import { SelectField } from "../../../components/fields/SelectField";
import { useModal } from "../../../components/modal/hook";
import { getErrorMessage } from "../../../utils/errors";
import { releaseService, type ReleaseOption } from "../../../services/release";
import { serviceService, type ServiceDto } from "../../../services/services";

type Props = {
  service: ServiceDto;
  onSuccess?: () => void;
};

const compareVersions = (a: string, b: string) => {
  const left = a.split(".").map(Number);
  const right = b.split(".").map(Number);
  const maxLength = Math.max(left.length, right.length);
  for (let i = 0; i < maxLength; i++) {
    const diff = (left[i] ?? 0) - (right[i] ?? 0);
    if (diff !== 0) return diff;
  }
  return 0;
};

export const ChangeVersionForm: FC<Props> = ({ service, onSuccess }) => {
  const { closeModal } = useModal();
  const [targetRelease, setTargetRelease] = useState("");

  const releasesQuery = useQuery({
    queryKey: ["releases"],
    queryFn: releaseService.fetchAll,
  });

  const releaseOptions = useMemo(() => {
    return (releasesQuery.data ?? [])
      .filter(
        (release: ReleaseOption) =>
          release.repositoryId === service.repository_id &&
          release.id !== service.release_id,
      )
      .sort((a, b) => compareVersions(b.version, a.version))
      .map(release => {
        const diff = compareVersions(release.version, service.release_version);
        const tag = diff > 0 ? "↑ upgrade" : "↓ downgrade";
        return {
          label: `${release.repositoryName} v${release.version}  (${tag})`,
          value: release.id,
        };
      });
  }, [releasesQuery.data, service.repository_id, service.release_id, service.release_version]);

  const changeVersionMutation = useMutation({
    mutationFn: serviceService.executeServiceCommand,
    onSuccess: () => {
      toast.success("Cambio de versión programado correctamente");
      closeModal();
      onSuccess?.();
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!targetRelease) {
      toast.error("Selecciona una versión destino");
      return;
    }
    changeVersionMutation.mutate({
      service_id: service.id,
      action: "upgrade",
      target_release: targetRelease,
    });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      <div className="text-sm text-base-content/80">
        Versión actual:{" "}
        <span className="font-mono font-semibold">
          {service.repository} v{service.release_version}
        </span>
      </div>
      <SelectField
        label="Versión destino"
        options={releaseOptions}
        value={targetRelease}
        onChange={event => setTargetRelease(event.target.value)}
        registration={{
          name: "target_release",
          onBlur: async () => undefined,
          onChange: async () => undefined,
          ref: () => undefined,
        }}
        isLoading={releasesQuery.isLoading}
        onReload={releasesQuery.refetch}
        disabled={releaseOptions.length === 0}
        placeholderOptionLabel={
          releaseOptions.length === 0
            ? "No hay versiones descargadas disponibles"
            : "Selecciona una versión"
        }
      />
      <Button
        type="submit"
        label="Cambiar versión"
        loading={changeVersionMutation.isPending}
        disabled={releaseOptions.length === 0 || !targetRelease}
      />
    </form>
  );
};
