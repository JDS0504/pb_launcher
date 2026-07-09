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
    | "restoring"
    | "sleeping";

  _pb_install: string;
  boot_user_email: string;
  boot_user_password: string;
  last_started: string;

  restart_policy: string;
  error_message: string;
  ip: string;
  port: number;

  repository: string;
  repository_id: string;
  release_id: string;
  release_version: string;
  cpu_quota?: string;
  memory_limit?: string;

  current_snapshot_id?: string;
  current_snapshot_applied_at?: string;

  created: string;

  // expand
  expand: {
    release: {
      id: string;
      version: string;
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
    cpu_quota?: string;
    memory_limit?: string;
  }) => {
    const services = pb.collection(SERVICES_COLLECTION);
    return await services.create({
      name: data.name,
      release: data.release,
      restart_policy: data.restart_policy,
      cpu_quota: data.cpu_quota,
      memory_limit: data.memory_limit,
    });
  },
  updateServiceInstance: async (data: {
    id: string;
    name: string;
    release: string;
    restart_policy: string;
    cpu_quota?: string;
    memory_limit?: string;
  }) => {
    const services = pb.collection(SERVICES_COLLECTION);
    await services.update(data.id, {
      name: data.name,
      release: data.release,
      restart_policy: data.restart_policy,
      cpu_quota: data.cpu_quota,
      memory_limit: data.memory_limit,
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
    "ip",
    "port",
    "created",
    "release",
    "cpu_quota",
    "memory_limit",
    "current_snapshot_id",
    "current_snapshot_applied_at",
    "expand.release.id",
    "expand.release.version",
  ].join(","),

  fetchServiceByName: async (name: string): Promise<ServiceDto> => {
    const service = await pb
        .collection(SERVICES_COLLECTION)
        .getFirstListItem<
          Omit<_Service, "repository" | "release_id" | "release_version">
        >(`name="${name}"`, {
          expand: "release",
          fields: serviceService.serviceFields,
        });

    const [commands, domains] = await Promise.all([

      pb.collection(COMANDS_COLLECTION).getFullList<{ service: string }>({
        fields: "service",
        filter: `status="pending"&&service="${service.id}"`,
      }),

      domainsService.fetchAllByServiceID(service.id),
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
      ip: service.ip ?? "",
      port: service.port ?? 0,
      created: service.created,
      repository: "PocketBase",
      repository_id: "pocketbase",
      release_id: service.expand.release.id,
      release_version: service.expand.release.version,
      domains: domains,
      cpu_quota: service.cpu_quota,
      memory_limit: service.memory_limit,
      current_snapshot_id: service.current_snapshot_id,
      current_snapshot_applied_at: service.current_snapshot_applied_at,
    };
  },

  fetchAllServices: async (): Promise<ServiceDto[]> => {
    const [services, commands, domains] = await Promise.all([
      pb
        .collection(SERVICES_COLLECTION)
        .getFullList<
          Omit<_Service, "repository" | "release_id" | "release_version">
        >({
          expand: "release",
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
        ip: s.ip ?? "",
        port: s.port ?? 0,
        created: s.created,
        repository: "PocketBase",
        repository_id: "pocketbase",
        release_id: s.expand.release.id,
        release_version: s.expand.release.version,
        domains: domains.filter(
          d => d.service === s.id,
        ),
        cpu_quota: s.cpu_quota,
        memory_limit: s.memory_limit,
        current_snapshot_id: s.current_snapshot_id,
        current_snapshot_applied_at: s.current_snapshot_applied_at,
      }),
    );
  },

  deleteServiceInstance: async (id: string) => {
    await pb.collection(SERVICES_COLLECTION).delete(id);
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
  upsertSuperuser: async (data: { service_id: string; password?: string }) => {
    const url = joinUrls(pb.baseURL, `/x-api/upsert_superuser/${data.service_id}`);
    const response = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: pb.authStore.token,
      },
      body: JSON.stringify({ password: data.password }),
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
  fetchOperationLogs: async (service_id: string, sinceDate?: string): Promise<OperationLog[]> => {
    const sevenDaysAgo = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString().replace("T", " ").slice(0, 19);
    let filter = `service="${service_id}"&&created>="${sevenDaysAgo}"`;
    if (sinceDate) {
      filter += `&&created>="${sinceDate}"`;
    }
    return pb.collection(OPERATION_LOGS_COLLECTION).getFullList<OperationLog>({
      filter,
      fields: "id,service,operation,status,message,metadata,created",
      sort: "-created",
    });
  },
  fetchAllOperationLogs: async (filter?: string): Promise<OperationLog[]> => {
    const sevenDaysAgo = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString().replace("T", " ").slice(0, 19);
    const baseFilter = `created>="${sevenDaysAgo}"`;
    const combined = filter ? `(${filter})&&${baseFilter}` : baseFilter;
    return pb.collection(OPERATION_LOGS_COLLECTION).getFullList<OperationLog>({
      filter: combined,
      expand: "service",
      fields:
        "id,service,operation,status,message,metadata,created,expand.service.id,expand.service.name",
      sort: "-created",
    });
  },
  fetchServiceUptimeView: async (): Promise<ServiceUptimeViewDto[]> => {
    return pb.collection("service_uptime_view").getFullList<ServiceUptimeViewDto>();
  },
  fetchServiceUptimeByID: async (serviceID: string): Promise<ServiceUptimeViewDto> => {
    return pb.collection("service_uptime_view").getOne<ServiceUptimeViewDto>(serviceID);
  },
};

export interface ServiceUptimeViewDto {
  id: string;
  service_name: string;
  service_status: string;
  uptime_24h: number;
  active_hours_24h: number;
  inactive_hours_24h: number;
  uptime_7d: number;
  active_hours_7d: number;
  inactive_hours_7d: number;
}

