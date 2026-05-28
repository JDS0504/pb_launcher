import { object, string } from "yup";
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

const LOCAL_STORAGE_KEY = "pb-launcher-create-service-defaults";

const getLocalStorageDefaults = () => {
  try {
    const data = localStorage.getItem(LOCAL_STORAGE_KEY);
    return data ? JSON.parse(data) : {};
  } catch {
    return {};
  }
};

const schema = object({
  name: stringRequired(), // Name of the new PocketBase instance
  repository: stringRequired(), // Repository/source for the instance
  instanceSource: stringRequired(), // Source for the instance (template, version, etc.)
  restartPolicy: stringRequired(), // Restart policy: "no" or "on-failure"
  superuserPassword: string().optional(),
  cpuQuota: string().optional(),
  memoryLimit: string().optional(),
});

type Props = {
  record?: ServiceDto;
  onSaveRecord?: () => void;
  width?: number;
};

export const ServiceForm: FC<Props> = ({ onSaveRecord, record, width }) => {
  const { closeModal } = useModal();
  const savedDefaults = useMemo(() => {
    return record == null ? getLocalStorageDefaults() : {};
  }, [record]);

  const form = useCustomForm(schema, {
    defaultValues: {
      name: record?.name ?? "",
      repository: record?.repository_id ?? savedDefaults.repository ?? "",
      instanceSource: record?.release_id ?? savedDefaults.instanceSource ?? "",
      restartPolicy: record?.restart_policy ?? savedDefaults.restartPolicy ?? "on-failure",
      superuserPassword: savedDefaults.superuserPassword ?? "",
      cpuQuota: record?.cpu_quota ?? savedDefaults.cpuQuota ?? "default",
      memoryLimit: record?.memory_limit ?? savedDefaults.memoryLimit ?? "default",
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
    mutationFn: async (data: {
      name: string;
      release: string;
      restart_policy: string;
      superuserPassword?: string;
      cpu_quota?: string;
      memory_limit?: string;
    }) => {
      const newService = await serviceService.createServiceInstance({
        name: data.name,
        release: data.release,
        restart_policy: data.restart_policy,
        cpu_quota: data.cpu_quota,
        memory_limit: data.memory_limit,
      });

      if (data.superuserPassword && newService?.id) {
        try {
          await serviceService.upsertSuperuser({
            service_id: newService.id,
            password: data.superuserPassword,
          });
        } catch (e) {
          console.error("Failed to upsert superuser on creation", e);
          throw new Error("El servicio se creó, pero no se pudo configurar la contraseña del administrador: " + getErrorMessage(e));
        }
      }
      return newService;
    },
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
    async ({ instanceSource, name, restartPolicy, superuserPassword, cpuQuota, memoryLimit }) => {
      if (record == null) {
        if (!superuserPassword) {
          form.setError("superuserPassword", {
            type: "manual",
            message: "La contraseña de administrador es requerida",
          });
          return;
        }

        try {
          localStorage.setItem(
            LOCAL_STORAGE_KEY,
            JSON.stringify({
              repository: form.getValues("repository"),
              instanceSource,
              restartPolicy,
              superuserPassword,
              cpuQuota,
              memoryLimit,
            }),
          );
        } catch (e) {
          console.error("Failed to save to localStorage", e);
        }

        createInstanceMutation.mutate({
          name,
          release: instanceSource,
          restart_policy: restartPolicy,
          superuserPassword,
          cpu_quota: cpuQuota,
          memory_limit: memoryLimit,
        });
      } else {
        updateInstanceMutation.mutate({
          id: record.id,
          name,
          release: instanceSource,
          restart_policy: restartPolicy,
          cpu_quota: cpuQuota,
          memory_limit: memoryLimit,
        });
      }
    },
  );
  return (
    <div style={{ width, maxWidth: "100%" }}>
      <form onSubmit={handleFormSubmit} className="space-y-5">
        <InputField
          label="Instance Name"
          registration={form.register("name")}
          autoComplete="off"
          error={form.formState.errors.name}
        />

        {record == null && (
          <InputField
            label="Superuser Password"
            type="password"
            registration={form.register("superuserPassword")}
            autoComplete="new-password"
            error={form.formState.errors.superuserPassword}
          />
        )}

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

        <SelectField
          label="Límite de CPU"
          options={[
            { label: "Predeterminado del Servidor", value: "default" },
            { label: "10% de CPU del Servidor", value: "10%" },
            { label: "20% de CPU del Servidor", value: "20%" },
            { label: "30% de CPU del Servidor", value: "30%" },
            { label: "50% de CPU del Servidor", value: "50%" },
            { label: "80% de CPU del Servidor", value: "80%" },
            { label: "100% de CPU del Servidor (1 vCPU)", value: "100%" },
            { label: "Sin límite (Uso libre)", value: "none" },
          ]}
          registration={form.register("cpuQuota")}
          autoComplete="off"
          error={form.formState.errors.cpuQuota}
        />

        <SelectField
          label="Límite de RAM"
          options={[
            { label: "Predeterminado del Servidor", value: "default" },
            { label: "128 MB", value: "128M" },
            { label: "256 MB", value: "256M" },
            { label: "384 MB", value: "384M" },
            { label: "512 MB", value: "512M" },
            { label: "768 MB", value: "768M" },
            { label: "1024 MB (1 GB)", value: "1024M" },
            { label: "2048 MB (2 GB)", value: "2048M" },
            { label: "Sin límite (Uso libre)", value: "none" },
          ]}
          registration={form.register("memoryLimit")}
          autoComplete="off"
          error={form.formState.errors.memoryLimit}
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
