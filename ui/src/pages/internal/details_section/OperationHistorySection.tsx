import { useQuery } from "@tanstack/react-query";
import { useState, type FC } from "react";
import classNames from "classnames";
import { Search } from "lucide-react";
import { ErrorFallback } from "../../../components/helpers/ErrorFallback";
import { serviceService } from "../../../services/services";

type Props = {
  service_id: string;
};

const PAGE_SIZE = 25;

export const OperationHistorySection: FC<Props> = ({ service_id }) => {
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<"all" | "success" | "error">("all");
  const [page, setPage] = useState(0);

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

  const filtered = operations.filter(op => {
    const matchSearch =
      search === "" ||
      op.operation.toLowerCase().includes(search.toLowerCase()) ||
      (op.message ?? "").toLowerCase().includes(search.toLowerCase());
    const matchStatus =
      statusFilter === "all" || op.status === statusFilter;
    return matchSearch && matchStatus;
  });

  const totalPages = Math.ceil(filtered.length / PAGE_SIZE);
  const paginated = filtered.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);

  const handleSearch = (value: string) => {
    setSearch(value);
    setPage(0);
  };

  const handleStatusFilter = (value: "all" | "success" | "error") => {
    setStatusFilter(value);
    setPage(0);
  };

  return (
    <div className="space-y-4 min-w-0">
      {/* Barra de búsqueda y filtros */}
      <div className="flex flex-col sm:flex-row gap-2">
        <div className="relative flex-1">
          <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-base-content/40" />
          <input
            id="history-search"
            type="search"
            className="input input-bordered input-sm w-full pl-9"
            placeholder="Buscar por operación o mensaje..."
            value={search}
            onChange={e => handleSearch(e.target.value)}
          />
        </div>
        <div className="flex gap-1 shrink-0">
          {(["all", "success", "error"] as const).map(s => (
            <button
              key={s}
              className={classNames("btn btn-xs capitalize", {
                "btn-neutral": statusFilter === s,
                "btn-ghost": statusFilter !== s,
              })}
              onClick={() => handleStatusFilter(s)}
            >
              {s === "all" ? "Todos" : s}
            </button>
          ))}
        </div>
      </div>

      {/* Contador */}
      {(search || statusFilter !== "all") && (
        <p className="text-xs text-base-content/50">
          {filtered.length} resultado{filtered.length !== 1 ? "s" : ""} de {operations.length}
        </p>
      )}

      {filtered.length === 0 ? (
        <div className="text-sm text-base-content/70 py-4 text-center">
          {operations.length === 0
            ? "No operations yet."
            : "No se encontraron operaciones con ese criterio."}
        </div>
      ) : (
        <>
          <div className="overflow-x-auto rounded-lg">
            <table className="table table-sm w-full">
              <thead>
                <tr>
                  <th className="hidden sm:table-cell whitespace-nowrap">Time</th>
                  <th>Operation</th>
                  <th>Status</th>
                  <th className="hidden md:table-cell">Message</th>
                </tr>
              </thead>
              <tbody>
                {paginated.map(operation => (
                  <tr key={operation.id}>
                    <td className="hidden sm:table-cell whitespace-nowrap text-xs text-base-content/60">
                      {new Date(operation.created).toLocaleString()}
                    </td>
                    <td className="min-w-0">
                      <div className="capitalize font-medium">{operation.operation}</div>
                      <div className="text-xs text-base-content/50 sm:hidden">
                        {new Date(operation.created).toLocaleString()}
                      </div>
                      {/* Message visible en mobile bajo operación */}
                      <div className="text-xs text-base-content/60 md:hidden max-w-[200px] truncate mt-0.5">
                        {operation.message || ""}
                      </div>
                    </td>
                    <td className="whitespace-nowrap">
                      <span
                        className={classNames("badge badge-sm", {
                          "badge-success": operation.status === "success",
                          "badge-error": operation.status === "error",
                        })}
                      >
                        {operation.status}
                      </span>
                    </td>
                    <td className="hidden md:table-cell">
                      <div
                        className="max-w-xs truncate text-sm"
                        title={operation.message || undefined}
                      >
                        {operation.message || "—"}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Paginación */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between gap-2 pt-1">
              <span className="text-xs text-base-content/50">
                Página {page + 1} de {totalPages}
              </span>
              <div className="flex gap-1">
                <button
                  className="btn btn-xs btn-ghost"
                  disabled={page === 0}
                  onClick={() => setPage(p => p - 1)}
                >
                  ← Anterior
                </button>
                <button
                  className="btn btn-xs btn-ghost"
                  disabled={page >= totalPages - 1}
                  onClick={() => setPage(p => p + 1)}
                >
                  Siguiente →
                </button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
};
