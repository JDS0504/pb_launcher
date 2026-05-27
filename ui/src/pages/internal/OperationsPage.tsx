import { useQuery } from "@tanstack/react-query";
import classNames from "classnames";
import { Link } from "react-router-dom";
import { useState, useMemo } from "react";
import { ErrorFallback } from "../../components/helpers/ErrorFallback";
import { serviceService } from "../../services/services";

export const OperationsPage = () => {
  const [selectedServiceId, setSelectedServiceId] = useState<string>("all");

  const operationsQuery = useQuery({
    queryKey: ["operation-logs", "global"],
    queryFn: serviceService.fetchAllOperationLogs,
    refetchInterval: 3000,
  });

  const servicesQuery = useQuery({
    queryKey: ["services"],
    queryFn: serviceService.fetchAllServices,
  });

  const filteredOperations = useMemo(() => {
    const ops = operationsQuery.data ?? [];
    if (selectedServiceId === "all") return ops;
    return ops.filter(op => op.service === selectedServiceId);
  }, [operationsQuery.data, selectedServiceId]);

  if (operationsQuery.isLoading || servicesQuery.isLoading) {
    return <div className="p-4">Loading...</div>;
  }

  if (operationsQuery.isError) {
    return (
      <ErrorFallback
        error={operationsQuery.error}
        onRetry={() => setTimeout(operationsQuery.refetch)}
      />
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-3.5">
        <div>
          <h2 className="text-xl font-semibold">Operation History</h2>
          <p className="text-sm text-base-content/70">
            Review launcher actions across all services.
          </p>
        </div>
        <div className="w-full md:w-auto shrink-0">
          <select
            className="select select-sm select-bordered w-full md:w-48"
            value={selectedServiceId}
            onChange={e => setSelectedServiceId(e.target.value)}
          >
            <option value="all">All Services</option>
            {servicesQuery.data?.map(service => (
              <option key={service.id} value={service.id}>
                {service.name}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div className="card bg-base-100 border border-base-300 shadow-sm">
        <div className="card-body">
          {filteredOperations.length === 0 ? (
            <div className="text-sm text-base-content/70">No operations found.</div>
          ) : (
            <div className="overflow-x-auto">
              <table className="table table-sm">
                <thead>
                  <tr>
                    <th>Time</th>
                    <th>Service</th>
                    <th>Operation</th>
                    <th>Status</th>
                    <th>Message</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredOperations.map(operation => (
                    <tr key={operation.id}>
                      <td className="whitespace-nowrap">
                        {new Date(operation.created).toLocaleString()}
                      </td>
                      <td className="whitespace-nowrap">
                        {operation.expand?.service ? (
                          <Link
                            to={`/services/${operation.expand.service.id}?section=history`}
                            className="link link-primary"
                          >
                            {operation.expand.service.name}
                          </Link>
                        ) : (
                          "-"
                        )}
                      </td>
                      <td className="capitalize">{operation.operation}</td>
                      <td>
                        <span
                           className={classNames("badge badge-sm", {
                            "badge-success": operation.status === "success",
                            "badge-error": operation.status === "error",
                          })}
                        >
                          {operation.status}
                        </span>
                      </td>
                      <td className="max-w-md whitespace-pre-wrap">
                        {operation.message || "-"}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
