import { joinUrls } from "../utils/url";
import { HttpError } from "./client/errors";
import { pb } from "./client/pb";

export type CPUInfo = {
  usage_percent: number;
  cores: number;
};

export type RAMInfo = {
  total_bytes: number;
  used_bytes: number;
  free_bytes: number;
  usage_percent: number;
};

export type DiskInfo = {
  total_bytes: number;
  used_bytes: number;
  free_bytes: number;
  usage_percent: number;
  path: string;
};

export type HostInfo = {
  os: string;
  platform: string;
  uptime_seconds: number;
  active_instances: number;
};

export type SystemStatus = {
  cpu: CPUInfo;
  ram: RAMInfo;
  disk: DiskInfo;
  host: HostInfo;
};

export const statusService = {
  fetchSystemStatus: async (): Promise<SystemStatus> => {
    const url = joinUrls(pb.baseURL, "/x-api/system/status");
    const response = await fetch(url, {
      headers: { Authorization: pb.authStore.token },
    });
    const json = await response.json().catch(() => null);
    if (!response.ok) {
      throw new HttpError(
        response.status,
        json?.message || "Failed to fetch system status",
        json,
      );
    }
    return json as SystemStatus;
  },
};
