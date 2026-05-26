import { pb } from "./client/pb";
import { joinUrls } from "../utils/url";

export const cliService = {
  executeCliCommand: async (serviceId: string, args: string[]) => {
    const url = joinUrls(pb.baseURL, `/x-api/service/cli/${serviceId}`);
    const response = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: pb.authStore.token,
      },
      body: JSON.stringify({ args }),
    });

    const json = await response.json();
    if (!response.ok) {
      throw new Error(json.error || "Failed to execute CLI command");
    }
    return json as { success: boolean; output: string };
  },
};
