import { joinUrls } from "../utils/url";
import { HttpError } from "./client/errors";
import { pb } from "./client/pb";

const parseFilename = (contentDisposition: string | null) => {
  const match = contentDisposition?.match(/filename="?([^";]+)"?/);
  return match?.[1] ?? "pblauncher-backup.zip";
};

export const backupService = {
  restoreBackup: async (data: { file: File; name: string }) => {
    const form = new FormData();
    form.append("backup", data.file);
    form.append("name", data.name);
    const url = joinUrls(pb.baseURL, "/x-api/services/restore");
    const response = await fetch(url, {
      method: "POST",
      headers: { Authorization: pb.authStore.token },
      body: form,
    });
    const json = await response.json().catch(() => null);
    if (!response.ok) {
      throw new HttpError(
        response.status,
        json?.message || "Failed to restore backup",
        json,
      );
    }
    return json as { service_id: string };
  },

  cloneService: async (data: { serviceID: string; name: string }) => {
    const form = new FormData();
    form.append("name", data.name);
    const url = joinUrls(pb.baseURL, `/x-api/services/${data.serviceID}/clone`);
    const response = await fetch(url, {
      method: "POST",
      headers: { Authorization: pb.authStore.token },
      body: form,
    });
    const json = await response.json().catch(() => null);
    if (!response.ok) {
      throw new HttpError(
        response.status,
        json?.message || "Failed to clone service",
        json,
      );
    }
    return json as { service_id: string };
  },

  listSnapshots: async (serviceID: string): Promise<SnapshotInfo[]> => {
    const url = joinUrls(pb.baseURL, `/x-api/services/${serviceID}/snapshots`);
    const response = await fetch(url, {
      headers: { Authorization: pb.authStore.token },
    });
    const json = await response.json().catch(() => null);
    if (!response.ok) {
      throw new HttpError(
        response.status,
        json?.message || "Failed to list snapshots",
        json,
      );
    }
    return Array.isArray(json) ? (json as SnapshotInfo[]) : [];
  },

  createSnapshot: async (data: { serviceID: string; name: string; comment?: string }) => {
    const form = new FormData();
    form.append("name", data.name);
    if (data.comment) form.append("comment", data.comment);
    const url = joinUrls(pb.baseURL, `/x-api/services/${data.serviceID}/snapshots`);
    const response = await fetch(url, {
      method: "POST",
      headers: { Authorization: pb.authStore.token },
      body: form,
    });
    const json = await response.json().catch(() => null);
    if (!response.ok) {
      throw new HttpError(
        response.status,
        json?.message || "Failed to create snapshot",
        json,
      );
    }
    return json as SnapshotInfo;
  },

  // Restaura in-place: reemplaza los datos del servicio actual con el snapshot.
  // Crea un auto-backup previo si el estado actual no está snapshotado.
  restoreSnapshot: async (data: { serviceID: string; snapshotID: string }) => {
    const url = joinUrls(
      pb.baseURL,
      `/x-api/services/${data.serviceID}/snapshots/${data.snapshotID}/restore`,
    );
    const response = await fetch(url, {
      method: "POST",
      headers: { Authorization: pb.authStore.token },
    });
    const json = await response.json().catch(() => null);
    if (!response.ok) {
      throw new HttpError(
        response.status,
        json?.message || "Failed to restore snapshot",
        json,
      );
    }
    return json as { pre_restore_snapshot_id: string | null; pre_restore_snapshot_name: string | null };
  },

  downloadSnapshot: async (serviceID: string, snapshotID: string) => {
    const url = joinUrls(
      pb.baseURL,
      `/x-api/services/${serviceID}/snapshots/${snapshotID}/download`,
    );
    const response = await fetch(url, {
      headers: { Authorization: pb.authStore.token },
    });
    if (!response.ok) {
      const json = await response.json().catch(() => null);
      throw new HttpError(
        response.status,
        json?.message || "Failed to download snapshot",
        json,
      );
    }
    const blob = await response.blob();
    const downloadUrl = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = downloadUrl;
    link.download = parseFilename(response.headers.get("content-disposition"));
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(downloadUrl);
  },

  deleteSnapshot: async (data: { serviceID: string; snapshotID: string }) => {
    const url = joinUrls(
      pb.baseURL,
      `/x-api/services/${data.serviceID}/snapshots/${data.snapshotID}`,
    );
    const response = await fetch(url, {
      method: "DELETE",
      headers: { Authorization: pb.authStore.token },
    });
    if (!response.ok) {
      const json = await response.json().catch(() => null);
      throw new HttpError(
        response.status,
        json?.message || "Failed to delete snapshot",
        json,
      );
    }
  },
};

export type SnapshotInfo = {
  id: string;
  name: string;
  comment: string;
  service_id: string;
  type: "manual" | "pre-restore";
  version: string;
  created_at: string;
  size: number;
  file: string;
};
