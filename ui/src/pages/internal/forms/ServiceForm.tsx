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

const getProfileFromLimits = (cpu?: string, ram?: string): string => {
  if (!cpu && !ram) return "low";
  if (cpu === "default" && ram === "default") return "default";
  if (cpu === "25%" && ram === "256M") return "low";
  if (cpu === "50%" && ram === "512M") return "standard";
  if (cpu === "100%" && ram === "1024M") return "high";
  if (cpu === "none" && ram === "none") return "none";
  return "low";
};

const getLimitsFromProfile = (profile: string): { cpu_quota: string; memory_limit: string } => {
  switch (profile) {
    case "low":
      return { cpu_quota: "25%", memory_limit: "256M" };
    case "standard":
      return { cpu_quota: "50%", memory_limit: "512M" };
    case "high":
      return { cpu_quota: "100%", memory_limit: "1024M" };
    case "none":
      return { cpu_quota: "none", memory_limit: "none" };
    case "default":
    default:
      return { cpu_quota: "default", memory_limit: "default" };
  }
};

const schema = object({
  name: stringRequired(), // Name of the new PocketBase instance
  repository: stringRequired(), // Repository/source for the instance
  instanceSource: stringRequired(), // Source for the instance (template, version, etc.)
  restartPolicy: stringRequired(), // Restart policy: "no" or "on-failure"
  superuserPassword: string().optional(),
  resourceProfile: string().optional(),
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
      repository: "pocketbase",
      instanceSource: record?.release_id ?? savedDefaults.instanceSource ?? "",
      restartPolicy: record?.restart_policy ?? savedDefaults.restartPolicy ?? "on-failure",
      superuserPassword: savedDefaults.superuserPassword ?? "",
      resourceProfile: getProfileFromLimits(record?.cpu_quota, record?.memory_limit) ?? savedDefaults.resourceProfile ?? "low",
    },
  });

  const releasesQuery = useQuery({
    queryKey: ["releases"],
    queryFn: releaseService.fetchAll,
  });

  const releaseOptions = useMemo<SelectFieldOption[]>(() => {
    return (
      releasesQuery.data
        ?.map(r => ({
        label: `v${r.version}`,
        value: r.id,
        })) ?? []
    );
  }, [releasesQuery.data]);

  useEffect(() => {
    if (record != null) return;
    const currentRelease = form.getValues("instanceSource");
    if (releaseOptions.length > 0) {
      if (!currentRelease || !releaseOptions.some(option => option.value === currentRelease)) {
        form.setValue("instanceSource", releaseOptions[0].value, { shouldValidate: true });
      }
    } else if (currentRelease !== "") {
      form.setValue("instanceSource", "", { shouldValidate: true });
    }
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
    async ({ instanceSource, name, restartPolicy, superuserPassword, resourceProfile }) => {
      const limits = getLimitsFromProfile(resourceProfile || "default");
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
              resourceProfile,
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
          cpu_quota: limits.cpu_quota,
          memory_limit: limits.memory_limit,
        });
      } else {
        updateInstanceMutation.mutate({
          id: record.id,
          name,
          release: instanceSource || record.release_id,
          restart_policy: restartPolicy,
          cpu_quota: limits.cpu_quota,
          memory_limit: limits.memory_limit,
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
          label="Version"
          options={releaseOptions}
          isLoading={releasesQuery.isLoading}
          registration={form.register("instanceSource")}
          autoComplete="off"
          error={form.formState.errors.instanceSource}
          disabled={record != null}
          placeholderOptionLabel="Select a version"
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
          label="Perfil de Recursos"
          options={[
            { label: "Bajo (256 MB RAM + 25% CPU) - Desarrollo o pruebas", value: "low" },
            { label: "Estándar (512 MB RAM + 50% CPU) - Recomendado", value: "standard" },
            { label: "Alto (1024 MB / 1 GB RAM + 100% CPU) - Producción y alta carga", value: "high" },
            { label: "Sin límites (Uso libre sin restricciones de cgroups)", value: "none" },
          ]}
          registration={form.register("resourceProfile")}
          autoComplete="off"
          error={form.formState.errors.resourceProfile}
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
