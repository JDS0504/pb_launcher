import { joinUrls } from "../utils/url";
import { HttpError } from "./client/errors";
import { pb } from "./client/pb";
import { COMANDS_COLLECTION } from "./release";
import { domainsService, type DomainDto } from "./services_domain";

interface _Service {
  id: string;
  name: string;
  status:
    | "idle"
    | "pending"
    | "running"
    | "stopped"
    | "failure"
    | "restoring";

  _pb_install: string;
  boot_user_email: string;
  boot_user_password: string;
  last_started: string;

  restart_policy: string;
  error_message: string;

  created: string;

  repository: string;
  repository_id: string;
  release_id: string;
  release_version: string;

  // expand
  expand: {
    release: {
      id: string;
      version: string;
      expand: {
        repository: {
          id: string;
          name: string;
        };
      };
    };
  };
}

export type ServiceLog = {
  id: number;
  service_id: string;
  stream: "stdout" | "stderr";
  message: string;
  timestamp: string; // ISO 8601 format
};

export type OperationLog = {
  id: string;
  service: string;
  operation: string;
  status: "success" | "error";
  message: string;
  metadata?: Record<string, unknown>;
  created: string;
  expand?: {
    service?: {
      id: string;
      name: string;
    };
  };
};

export const SERVICES_COLLECTION = "services";
export const OPERATION_LOGS_COLLECTION = "operation_logs";

export type ServiceDto = Omit<_Service, "expand"> & { domains?: DomainDto[] };

export const serviceService = {
  createServiceInstance: async (data: {
    name: string;
    release: string;
    restart_policy: string;
  }) => {
    const services = pb.collection(SERVICES_COLLECTION);
    await services.create({
      name: data.name,
      release: data.release,
      restart_policy: data.restart_policy,
    });
  },
  updateServiceInstance: async (data: {
    id: string;
    name: string;
    release: string;
    restart_policy: string;
  }) => {
    const services = pb.collection(SERVICES_COLLECTION);
    await services.update(data.id, {
      name: data.name,
      release: data.release,
      restart_policy: data.restart_policy,
    });
  },

  serviceFields: [
    "id",
    "name",
    "status",
    "_pb_install",
    "boot_user_email",
    "boot_user_password",
    "last_started",
    "restart_policy",
    "error_message",
    "created",
    "release",
        "expand.release.id",
        "expand.release.version",
        "expand.release.expand.repository.id",
        "expand.release.expand.repository.name",
  ].join(","),

  fetchServiceByID: async (serviceID: string): Promise<ServiceDto> => {
    const [service, commands] = await Promise.all([
      pb
        .collection(SERVICES_COLLECTION)
        .getOne<
          Omit<_Service, "repository" | "release_id" | "release_version">
        >(serviceID, {
          filter: `deleted=""`,
          expand: "release.repository",
          fields: serviceService.serviceFields,
        }),

      pb.collection(COMANDS_COLLECTION).getFullList<{ service: string }>({
        fields: "service",
        filter: `status="pending"&&service="${serviceID}"`,
      }),
    ]);
    return {
      id: service.id,
      name: service.name,
      status: commands.length > 0 ? "pending" : service.status,
      _pb_install: service._pb_install ?? "",
      boot_user_email: service.boot_user_email,
      boot_user_password: service.boot_user_password,
      last_started: service.last_started,
      restart_policy: service.restart_policy,
      error_message: service.error_message,
      created: service.created,
        repository: service.expand.release.expand.repository.name,
        repository_id: service.expand.release.expand.repository.id,
        release_id: service.expand.release.id,
        release_version: service.expand.release.version,
    };
  },

  fetchAllServices: async (): Promise<ServiceDto[]> => {
    const [services, commands, domains] = await Promise.all([
      pb
        .collection(SERVICES_COLLECTION)
        .getFullList<
          Omit<_Service, "repository" | "release_id" | "release_version">
        >({
          filter: `deleted=""`,
          expand: "release.repository",
          fields: serviceService.serviceFields,
        }),
      pb.collection(COMANDS_COLLECTION).getFullList<{ service: string }>({
        fields: "service",
        filter: `status="pending"`,
      }),
      domainsService.fetchFullList(),
    ]);
    const pendingServices = new Set(commands.map(c => c.service));
    return services.map(
      (s): ServiceDto => ({
        id: s.id,
        name: s.name,
        status: pendingServices.has(s.id) ? "pending" : s.status,
        _pb_install: s._pb_install ?? "",
        boot_user_email: s.boot_user_email,
        boot_user_password: s.boot_user_password,
        last_started: s.last_started,
        restart_policy: s.restart_policy,
        error_message: s.error_message,
        created: s.created,
        repository: s.expand.release.expand.repository.name,
        repository_id: s.expand.release.expand.repository.id,
        release_id: s.expand.release.id,
        release_version: s.expand.release.version,
        domains: domains.filter(
          d => d.service === s.id && d.x_has_valid_ssl_cert,
        ),
      }),
    );
  },

  deleteServiceInstance: async (id: string) => {
    const services = pb.collection(SERVICES_COLLECTION);
    await services.update(id, { deleted: new Date().toJSON() });
  },

  executeServiceCommand: async (data: {
    service_id: string;
    action: "stop" | "start" | "restart" | "upgrade";
    target_release?: string;
  }) => {
    const comands = pb.collection(COMANDS_COLLECTION);
    await comands.create({
      service: data.service_id,
      action: data.action,
      target_release: data.target_release,
    });
  },
  upsertSuperuser: async (service_id: string) => {
    const url = joinUrls(pb.baseURL, `/x-api/upsert_superuser/${service_id}`);
    const response = await fetch(url, {
      headers: { Authorization: pb.authStore.token },
    });
    const json = await response.json();
    if (!response.ok) {
      throw new HttpError(
        response.status,
        json?.message || "Unexpected error",
        json,
      );
    }
    return json as { email: string; password: string };
  },
  fetchServiceLogs: async (
    signal: AbortSignal,
    service_id: string,
    limit = 10,
  ): Promise<ServiceLog[]> => {
    const url = joinUrls(
      pb.baseURL,
      `/x-api/service/logs/${service_id}/${limit}`,
    );
    const response = await fetch(url, {
      signal,
      headers: { Authorization: pb.authStore.token },
    });
    const json = await response.json();

    if (!response.ok) {
      throw new HttpError(
        response.status,
        json?.message || "Unexpected error",
        json,
      );
    }
    if (json == null || !Array.isArray(json)) return [];
    return json as ServiceLog[];
  },
  fetchOperationLogs: async (service_id: string): Promise<OperationLog[]> => {
    return pb.collection(OPERATION_LOGS_COLLECTION).getFullList<OperationLog>({
      filter: `service="${service_id}"`,
      fields: "id,service,operation,status,message,metadata,created",
      sort: "-created",
    });
  },
  fetchAllOperationLogs: async (): Promise<OperationLog[]> => {
    return pb.collection(OPERATION_LOGS_COLLECTION).getFullList<OperationLog>({
      expand: "service",
      fields:
        "id,service,operation,status,message,metadata,created,expand.service.id,expand.service.name",
      sort: "-created",
    });
  },
};
