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

  uploadFiles: async (
    serviceID: string,
    destPath: string,
    files: File[],
    onProgress?: (percent: number) => void,
  ): Promise<void> => {
    const url = joinUrls(pb.baseURL, `/x-api/services/${serviceID}/files/upload`);
    const formData = new FormData();
    formData.append("path", destPath);
    for (const file of files) {
      formData.append("files", file);
    }

    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      xhr.open("POST", url);
      xhr.setRequestHeader("Authorization", pb.authStore.token);

      xhr.upload.onprogress = (event) => {
        if (event.lengthComputable && onProgress) {
          onProgress(Math.round((event.loaded / event.total) * 100));
        }
      };

      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve();
          return;
        }
        let json: { message?: string } | null = null;
        try { json = JSON.parse(xhr.responseText); } catch { /* no-op */ }
        reject(new HttpError(xhr.status, json?.message ?? "Failed to upload files", json));
      };

      xhr.onerror = () => reject(new Error("Error de red durante el upload"));
      xhr.send(formData);
    });
  },

  downloadFile: async (serviceID: string, path: string): Promise<Blob> => {
    const url = joinUrls(
      pb.baseURL,
      `/x-api/services/${serviceID}/files/download?path=${encodeURIComponent(path)}`,
    );
    const response = await fetch(url, {
      headers: { Authorization: pb.authStore.token },
    });
    if (!response.ok) {
      throw new Error("Failed to download file");
    }
    return response.blob();
  },

  downloadFileInBrowser: async (serviceID: string, path: string): Promise<void> => {
    const blob = await filesService.downloadFile(serviceID, path);
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = path.split("/").pop() || "download";
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    window.URL.revokeObjectURL(url);
  },

  createFolder: async (data: { serviceID: string; path: string }) => {
    const url = joinUrls(pb.baseURL, `/x-api/services/${data.serviceID}/files/folder`);
    const response = await fetch(url, {
      method: "POST",
      headers: {
        Authorization: pb.authStore.token,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ path: data.path }),
    });
    if (!response.ok) {
      const json = await response.json().catch(() => null);
      throw new HttpError(
        response.status,
        json?.message || "Failed to create folder",
        json,
      );
    }
  },

  renameFile: async (data: { serviceID: string; oldPath: string; newPath: string }) => {
    const url = joinUrls(pb.baseURL, `/x-api/services/${data.serviceID}/files/rename`);
    const response = await fetch(url, {
      method: "PATCH",
      headers: {
        Authorization: pb.authStore.token,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ old_path: data.oldPath, new_path: data.newPath }),
    });
    if (!response.ok) {
      const json = await response.json().catch(() => null);
      throw new HttpError(
        response.status,
        json?.message || "Failed to rename file",
        json,
      );
    }
  },

  extractZip: async (data: { serviceID: string; path: string }) => {
    const url = joinUrls(pb.baseURL, `/x-api/services/${data.serviceID}/files/unzip`);
    const response = await fetch(url, {
      method: "POST",
      headers: {
        Authorization: pb.authStore.token,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ path: data.path }),
    });
    if (!response.ok) {
      const json = await response.json().catch(() => null);
      throw new HttpError(
        response.status,
        json?.message || "Failed to extract zip file",
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
