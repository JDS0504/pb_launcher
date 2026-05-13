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
  expand: {
    repository: {
      id: string;
      name: string;
    };
  };
}

export const releaseService = {
  fetchAll: async (): Promise<ReleaseOption[]> => {
    const releases = pb.collection(RELEASES_COLLECTION);
    const records = await releases.getFullList<ReleaseDto>({
      expand: "repository",
      fields: "id,version,expand.repository.id,expand.repository.name",
      sort: "repository,-version",
    });
    return records.map(r => ({
      id: r.id,
      repositoryId: r.expand.repository.id,
      repositoryName: r.expand.repository.name,
      version: r.version,
    }));
  },
};
