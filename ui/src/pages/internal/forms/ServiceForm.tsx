import { object } from "yup";
import { stringRequired } from "../../../utils/validation";
import { useCustomForm } from "../../../hooks/useCustomForm";
import { InputField } from "../../../components/fields/InputField";
import { Button } from "../../../components/buttons/Button";
import {
  SelectField,
  type SelectFieldOption,
} from "../../../components/fields/SelectField";
import { serviceService, type ServiceDto } from "../../../services/services";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useEffect, useMemo, type FC } from "react";
import { useModal } from "../../../components/modal/hook";
import toast from "react-hot-toast";
import { getErrorMessage } from "../../../utils/errors";
import classNames from "classnames";
import { releaseService } from "../../../services/release";

const schema = object({
  name: stringRequired(), // Name of the new PocketBase instance
  repository: stringRequired(), // Repository/source for the instance
  instanceSource: stringRequired(), // Source for the instance (template, version, etc.)
  restartPolicy: stringRequired(), // Restart policy: "no" or "on-failure"
});

type Props = {
  record?: ServiceDto;
  onSaveRecord?: () => void;
  width?: number;
};

export const ServiceForm: FC<Props> = ({ onSaveRecord, record, width }) => {
  const { closeModal } = useModal();
  const form = useCustomForm(schema, {
    defaultValues: {
      name: record?.name,
      repository: record?.repository_id,
      instanceSource: record?.release_id,
      restartPolicy: record?.restart_policy ?? "on-failure",
    },
  });
  const selectedRepository = form.watch("repository");

  const releasesQuery = useQuery({
    queryKey: ["releases"],
    queryFn: releaseService.fetchAll,
  });

  const repositoryOptions = useMemo<SelectFieldOption[]>(() => {
    const repositories = new Map<string, string>();
    for (const release of releasesQuery.data ?? []) {
      repositories.set(release.repositoryId, release.repositoryName);
    }
    return [...repositories.entries()].map(([value, label]) => ({
      label,
      value,
    }));
  }, [releasesQuery.data]);

  const releaseOptions = useMemo<SelectFieldOption[]>(() => {
    return (
      releasesQuery.data
        ?.filter(r => r.repositoryId === selectedRepository)
        .map(r => ({
        label: `v${r.version}`,
        value: r.id,
        })) ?? []
    );
  }, [releasesQuery.data, selectedRepository]);

  useEffect(() => {
    if (record != null || selectedRepository || repositoryOptions.length === 0) {
      return;
    }
    form.setValue("repository", repositoryOptions[0].value, {
      shouldValidate: true,
    });
  }, [form, record, repositoryOptions, selectedRepository]);

  useEffect(() => {
    if (record != null) return;
    const currentRelease = form.getValues("instanceSource");
    if (releaseOptions.some(option => option.value === currentRelease)) return;
    if (currentRelease === "") return;
    form.setValue("instanceSource", "", { shouldValidate: true });
  }, [form, record, releaseOptions]);

  const createInstanceMutation = useMutation({
    mutationFn: serviceService.createServiceInstance,
    onSuccess: () => {
      toast.success("Service created successfully");
      closeModal();
      onSaveRecord?.();
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const updateInstanceMutation = useMutation({
    mutationFn: serviceService.updateServiceInstance,
    onSuccess: () => {
      toast.success("Service updated successfully");
      closeModal();
      onSaveRecord?.();
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const handleFormSubmit = form.handleSubmit(
    ({ instanceSource, name, restartPolicy }) => {
      if (record == null)
        createInstanceMutation.mutate({
          name,
          release: instanceSource,
          restart_policy: restartPolicy,
        });
      else
        updateInstanceMutation.mutate({
          id: record.id,
          name,
          release: instanceSource,
          restart_policy: restartPolicy,
        });
    },
  );
  return (
    <div style={{ width: width }}>
      <form onSubmit={handleFormSubmit} className="space-y-5">
        <InputField
          label="Instance Name"
          registration={form.register("name")}
          autoComplete="off"
          error={form.formState.errors.name}
        />

        <SelectField
          label="Repository"
          options={repositoryOptions}
          isLoading={releasesQuery.isLoading}
          onReload={releasesQuery.refetch}
          registration={form.register("repository")}
          autoComplete="off"
          error={form.formState.errors.repository}
          disabled={record != null}
        />

        <SelectField
          label="Version"
          options={releaseOptions}
          isLoading={releasesQuery.isLoading}
          registration={form.register("instanceSource")}
          autoComplete="off"
          error={form.formState.errors.instanceSource}
          disabled={record != null}
          placeholderOptionLabel={
            selectedRepository ? "Select a version" : "Select a repository first"
          }
        />

        <SelectField
          label="Restart Policy"
          options={[
            { label: "No", value: "no" },
            { label: "On Failure", value: "on-failure" },
          ]}
          registration={form.register("restartPolicy")}
          autoComplete="off"
          error={form.formState.errors.restartPolicy}
        />
        <div
          className={classNames("mt-8", {
            "flex justify-end": width == null || width > 400,
          })}
        >
          <div
            className={classNames("form-control", {
              "w-[200px]": width == null || width > 400,
            })}
          >
            <Button type="submit" label="Guardar" loading={false} />
          </div>
        </div>
      </form>
    </div>
  );
};
