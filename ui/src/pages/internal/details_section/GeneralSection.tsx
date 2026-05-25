import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { FC } from "react";
import { ServiceForm } from "../forms/ServiceForm";
import { Navigate } from "react-router-dom";
import classNames from "classnames";
import { serviceService } from "../../../services/services";
import { ErrorFallback } from "../../../components/helpers/ErrorFallback";

type Props = {
  service_id: string;
};

export const GeneralSection: FC<Props> = ({ service_id }) => {
  const queryClient = useQueryClient();
  const serviceQuery = useQuery({
    queryKey: ["services", service_id],
    queryFn: () => serviceService.fetchServiceByID(service_id),
    refetchOnMount: true,
  });

  if (serviceQuery.isFetching) {
    return <div className="p-4">Loading...</div>;
  }

  if (serviceQuery.isError)
    return (
      <ErrorFallback
        error={serviceQuery.error}
        onRetry={() => setTimeout(serviceQuery.refetch)}
      />
    );

  if (serviceQuery.data == null) return <Navigate to={"/"} />;
  const service = serviceQuery.data;
  const status =
    service.status.charAt(0).toUpperCase() + service.status.slice(1);

  return (
    <div className="relative pt-4">
      <ServiceForm
        record={service}
        onSaveRecord={() => {
          queryClient.invalidateQueries({ queryKey: ["services", service_id] });
        }}
      />
      <p
        className={classNames("badge badge-sm absolute -top-2 right-2", {
          "badge-success": service.status === "running",
          "badge-warning":
            service.status === "pending" || service.status === "idle",
          "badge-error": service.status === "failure",
          "badge-neutral": !["running", "pending", "idle", "failure"].includes(
            service.status,
          ),
        })}
      >
        {status}
      </p>
    </div>
  );
};
