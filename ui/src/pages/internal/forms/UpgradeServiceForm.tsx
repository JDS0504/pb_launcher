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
  onUpgrade?: () => void;
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

export const UpgradeServiceForm: FC<Props> = ({ service, onUpgrade }) => {
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
          compareVersions(release.version, service.release_version) > 0,
      )
      .sort((a, b) => compareVersions(b.version, a.version))
      .map(release => ({
        label: `${release.repositoryName} v${release.version}`,
        value: release.id,
      }));
  }, [releasesQuery.data, service.repository_id, service.release_version]);

  const upgradeMutation = useMutation({
    mutationFn: serviceService.executeServiceCommand,
    onSuccess: () => {
      toast.success("Upgrade scheduled successfully");
      closeModal();
      onUpgrade?.();
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!targetRelease) {
      toast.error("Select a target version");
      return;
    }
    upgradeMutation.mutate({
      service_id: service.id,
      action: "upgrade",
      target_release: targetRelease,
    });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      <div className="text-sm text-base-content/80">
        Current version: {service.repository} v{service.release_version}
      </div>
      <SelectField
        label="Target Version"
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
            ? "No newer downloaded versions"
            : "Select target version"
        }
      />
      <Button
        type="submit"
        label="Upgrade"
        loading={upgradeMutation.isPending}
        disabled={releaseOptions.length === 0}
      />
    </form>
  );
};
