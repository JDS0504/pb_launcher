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
    return pb.collection(REPOSITORIES_COLLECTION).getFullList<RepositoryDto>({
      sort: "name",
    });
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
};
