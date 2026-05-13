import { useQuery } from "@tanstack/react-query";
import classNames from "classnames";
import { Link } from "react-router-dom";
import { ErrorFallback } from "../../components/helpers/ErrorFallback";
import { serviceService } from "../../services/services";

export const OperationsPage = () => {
  const operationsQuery = useQuery({
    queryKey: ["operation-logs", "global"],
    queryFn: serviceService.fetchAllOperationLogs,
    refetchInterval: 3000,
  });

  if (operationsQuery.isLoading) return <div className="p-4">Loading...</div>;
  if (operationsQuery.isError) {
    return (
      <ErrorFallback
        error={operationsQuery.error}
        onRetry={() => setTimeout(operationsQuery.refetch)}
      />
    );
  }

  const operations = operationsQuery.data ?? [];

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold">Operation History</h2>
        <p className="text-sm text-base-content/70">
          Review launcher actions across all services.
        </p>
      </div>

      <div className="card bg-base-100 border border-base-300 shadow-sm">
        <div className="card-body">
          {operations.length === 0 ? (
            <div className="text-sm text-base-content/70">No operations yet.</div>
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
                  {operations.map(operation => (
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
