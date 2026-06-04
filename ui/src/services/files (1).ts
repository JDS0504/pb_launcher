import { joinUrls } from "../utils/url";
import { HttpError } from "./client/errors";
import { pb } from "./client/pb";

export type PBFileEntry = {
  path: string;
  size: number;
  updated_at: string;
  is_dir?: boolean;
};

export type PBFileContent = {
  path: string;
  content: string;
};

export const filesService = {
  fetchFiles: async (serviceID: string): Promise<PBFileEntry[]> => {
    const url = joinUrls(pb.baseURL, `/x-api/services/${serviceID}/files`);
    const response = await fetch(url, {
      headers: { Authorization: pb.authStore.token },
    });
    const json = await response.json().catch(() => null);
    if (!response.ok) {
      throw new HttpError(
        response.status,
        json?.message || "Failed to list files",
        json,
      );
    }
    return Array.isArray(json) ? (json as PBFileEntry[]) : [];
  },

  readFile: async (serviceID: string, path: string): Promise<PBFileContent> => {
    const url = joinUrls(
      pb.baseURL,
      `/x-api/services/${serviceID}/files/content?path=${encodeURIComponent(path)}`,
    );
    const response = await fetch(url, {
      headers: { Authorization: pb.authStore.token },
    });
    const json = await response.json().catch(() => null);
    if (!response.ok) {
      throw new HttpError(
        response.status,
        json?.message || "Failed to read file",
        json,
      );
    }
    return json as PBFileContent;
  },

  saveFile: async (data: {
    serviceID: string;
    path: string;
    content: string;
  }) => {
    const url = joinUrls(pb.baseURL, `/x-api/services/${data.serviceID}/files/content`);
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
        json?.message || "Failed to save file",
        json,
      );
    }
  },

  deleteFile: async (data: { serviceID: string; path: string }) => {
    const url = joinUrls(
      pb.baseURL,
      `/x-api/services/${data.serviceID}/files?path=${encodeURIComponent(data.path)}`,
    );
    const response = await fetch(url, {
      method: "DELETE",
      headers: { Authorization: pb.authStore.token },
    });
    if (!response.ok) {
      const json = await response.json().catch(() => null);
      throw new HttpError(
        response.status,
        json?.message || "Failed to delete file",
        json,
      );
    }
  },
};
