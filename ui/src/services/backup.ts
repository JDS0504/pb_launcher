import { joinUrls } from "../utils/url";
import { HttpError } from "./client/errors";
import { pb } from "./client/pb";

const parseFilename = (contentDisposition: string | null) => {
  const match = contentDisposition?.match(/filename="?([^";]+)"?/);
  return match?.[1] ?? "pblauncher-backup.zip";
};

export const backupService = {
  downloadBackup: async (serviceID: string) => {
    const url = joinUrls(pb.baseURL, `/x-api/services/${serviceID}/backup`);
    const response = await fetch(url, {
      headers: { Authorization: pb.authStore.token },
    });
    if (!response.ok) {
      const json = await response.json().catch(() => null);
      throw new HttpError(
        response.status,
        json?.message || "Failed to create backup",
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
};
