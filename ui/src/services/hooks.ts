import { joinUrls } from "../utils/url";
import { HttpError } from "./client/errors";
import { pb } from "./client/pb";

export type PBHookFile = {
  path: string;
  size: number;
  updated_at: string;
};

export type PBHookContent = {
  path: string;
  content: string;
};

const parseFilename = (contentDisposition: string | null) => {
  const match = contentDisposition?.match(/filename="?([^";]+)"?/);
  return match?.[1] ?? "pb-hooks.zip";
};

export const hooksService = {
  fetchHooks: async (serviceID: string): Promise<PBHookFile[]> => {
    const url = joinUrls(pb.baseURL, `/x-api/services/${serviceID}/hooks`);
    const response = await fetch(url, {
      headers: { Authorization: pb.authStore.token },
    });
    const json = await response.json().catch(() => null);
    if (!response.ok) {
      throw new HttpError(
        response.status,
        json?.message || "Failed to list PB hooks",
        json,
      );
    }
    return Array.isArray(json) ? (json as PBHookFile[]) : [];
  },

  exportHooks: async (serviceID: string) => {
    const url = joinUrls(pb.baseURL, `/x-api/services/${serviceID}/hooks/export`);
    const response = await fetch(url, {
      headers: { Authorization: pb.authStore.token },
    });
    if (!response.ok) {
      const json = await response.json().catch(() => null);
      throw new HttpError(
        response.status,
        json?.message || "Failed to export PB hooks",
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

  importHooks: async (data: { serviceID: string; file: File }) => {
    const form = new FormData();
    form.append("hooks", data.file);
    const url = joinUrls(pb.baseURL, `/x-api/services/${data.serviceID}/hooks/import`);
    const response = await fetch(url, {
      method: "POST",
      headers: { Authorization: pb.authStore.token },
      body: form,
    });
    const json = await response.json().catch(() => null);
    if (!response.ok) {
      throw new HttpError(
        response.status,
        json?.message || "Failed to import PB hooks",
        json,
      );
    }
    return json as { files: string[]; count: number };
  },

  readHook: async (serviceID: string, path: string): Promise<PBHookContent> => {
    const url = joinUrls(
      pb.baseURL,
      `/x-api/services/${serviceID}/hooks/file?path=${encodeURIComponent(path)}`,
    );
    const response = await fetch(url, {
      headers: { Authorization: pb.authStore.token },
    });
    const json = await response.json().catch(() => null);
    if (!response.ok) {
      throw new HttpError(
        response.status,
        json?.message || "Failed to read PB hook",
        json,
      );
    }
    return json as PBHookContent;
  },

  saveHook: async (data: {
    serviceID: string;
    path: string;
    content: string;
  }) => {
    const url = joinUrls(pb.baseURL, `/x-api/services/${data.serviceID}/hooks/file`);
    const response = await fetch(url, {
      method: "PUT",
      headers: {
        Authorization: pb.authStore.token,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ path: data.path, content: data.content }),
    });
    if (!response.ok) {
      const json = await response.json().catch(() => null);
      throw new HttpError(
        response.status,
        json?.message || "Failed to save PB hook",
        json,
      );
    }
  },

  deleteHook: async (data: { serviceID: string; path: string }) => {
    const url = joinUrls(
      pb.baseURL,
      `/x-api/services/${data.serviceID}/hooks/file?path=${encodeURIComponent(data.path)}`,
    );
    const response = await fetch(url, {
      method: "DELETE",
      headers: { Authorization: pb.authStore.token },
    });
    if (!response.ok) {
      const json = await response.json().catch(() => null);
      throw new HttpError(
        response.status,
        json?.message || "Failed to delete PB hook",
        json,
      );
    }
  },
};
