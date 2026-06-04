import { useQueryClient } from "@tanstack/react-query";
import type { FC } from "react";
import classNames from "classnames";
import { ServiceForm } from "../forms/ServiceForm";
import type { ServiceDto } from "../../../services/services";

type Props = {
  service_id: string;
  service?: ServiceDto;
};

export const GeneralSection: FC<Props> = ({ service_id, service }) => {
  const queryClient = useQueryClient();

  if (service == null) {
    return <div className="p-4">Loading...</div>;
  }

  const status =
    service.status.charAt(0).toUpperCase() + service.status.slice(1);

  return (
    <div className="relative pt-4">
      <ServiceForm
        record={service}
        onSaveRecord={() => {
          queryClient.invalidateQueries({ queryKey: ["services", service_id] });
          queryClient.invalidateQueries({ queryKey: ["services"] });
        }}
      />
      <p
        className={classNames("badge badge-sm absolute -top-2 right-2", {
          "badge-success": service.status === "running",
          "badge-info": service.status === "sleeping",
          "badge-warning":
            service.status === "pending" || service.status === "idle",
          "badge-error": service.status === "failure",
          "badge-neutral": !["running", "pending", "idle", "failure", "sleeping"].includes(
            service.status,
          ),
        })}
      >
        {status}
      </p>
    </div>
  );
};
