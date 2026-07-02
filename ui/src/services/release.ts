import { pb } from "./client/pb";

export const RELEASES_COLLECTION = "releases";
export const COMANDS_COLLECTION = "comands";

export interface ReleaseOption {
  id: string;
  repositoryId: string;
  repositoryName: string;
  version: string;
}

interface ReleaseDto {
  id: string;
  version: string;
}

export const releaseService = {
  fetchAll: async (): Promise<ReleaseOption[]> => {
    const releases = pb.collection(RELEASES_COLLECTION);
    const records = await releases.getFullList<ReleaseDto>({
      fields: "id,version",
      sort: "-version",
    });
    return records.map(r => ({
      id: r.id,
      repositoryId: "pocketbase",
      repositoryName: "PocketBase",
      version: r.version,
    }));
  },
};
