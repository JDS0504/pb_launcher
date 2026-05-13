import { useQuery } from "@tanstack/react-query";
import type { FC } from "react";
import classNames from "classnames";
import { ErrorFallback } from "../../../components/helpers/ErrorFallback";
import { serviceService } from "../../../services/services";

type Props = {
  service_id: string;
};

export const OperationHistorySection: FC<Props> = ({ service_id }) => {
  const operationsQuery = useQuery({
    queryKey: ["operation-logs", service_id],
    queryFn: () => serviceService.fetchOperationLogs(service_id),
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
  if (operations.length === 0) {
    return <div className="text-sm text-base-content/70">No operations yet.</div>;
  }

  return (
    <div className="overflow-x-auto">
      <table className="table table-sm">
        <thead>
          <tr>
            <th>Time</th>
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
  );
};
