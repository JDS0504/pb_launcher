import { joinUrls } from "../utils/url";
import { HttpError } from "./client/errors";
import { pb } from "./client/pb";

export const REPOSITORIES_COLLECTION = "repositories";
export const DEFAULT_REPOSITORY_ID = "pb91u2l315h29a5";

export type RepositoryDto = {
  id: string;
  name: string;
  repository: string;
  token: string;
  retention: number;
  release_file_pattern: string;
  exec_file_pattern: string;
  disabled: boolean;
  last_sync_at?: string;
  last_sync_status?: "never" | "syncing" | "success" | "error";
  last_sync_error?: string;
  release_count?: number;
  downloaded_versions_count?: number;
};

export type RepositoryPayload = {
  name: string;
  repository: string;
  token: string;
  retention: number;
  release_file_pattern: string;
  exec_file_pattern: string;
  disabled: boolean;
};

export const repositoriesService = {
  fetchAll: async (): Promise<RepositoryDto[]> => {
    const url = joinUrls(pb.baseURL, "/x-api/repositories/status");
    const response = await fetch(url, {
      headers: { Authorization: pb.authStore.token },
    });
    const json = await response.json().catch(() => null);
    if (!response.ok) {
      throw new HttpError(
        response.status,
        json?.message || "Failed to fetch repositories",
        json,
      );
    }
    return Array.isArray(json) ? (json as RepositoryDto[]) : [];
  },

  create: async (data: RepositoryPayload) => {
    return pb.collection(REPOSITORIES_COLLECTION).create(data);
  },

  update: async (id: string, data: Partial<RepositoryPayload>) => {
    return pb.collection(REPOSITORIES_COLLECTION).update(id, data);
  },

  delete: async (id: string) => {
    return pb.collection(REPOSITORIES_COLLECTION).delete(id);
  },

  sync: async (id: string) => {
    const url = joinUrls(pb.baseURL, `/x-api/repositories/${id}/sync`);
    const response = await fetch(url, {
      method: "POST",
      headers: { Authorization: pb.authStore.token },
    });
    if (!response.ok) {
      const json = await response.json().catch(() => null);
      throw new HttpError(
        response.status,
        json?.message || "Failed to sync repository",
        json,
      );
    }
  },
};
