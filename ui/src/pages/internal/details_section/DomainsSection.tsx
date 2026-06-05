import { useMemo, type FC } from "react";
import { DomainCard } from "../components/DomainCard";
import { DnsSetupGuideButton } from "../components/DnsSetupGuide";
import { useProxyConfigs } from "../../../hooks/useProxyConfigs";
import { formatUrl } from "../../../utils/url";
import { Plus } from "lucide-react";
import { useModal } from "../../../components/modal/hook";
import { DomainForm } from "../forms/DomainForm";
import {
  domainsService,
  type DomainDto,
} from "../../../services/services_domain";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ErrorFallback } from "../../../components/helpers/ErrorFallback";
import { getErrorMessage } from "../../../utils/errors";
import toast from "react-hot-toast";
import { useConfirmModal } from "../../../hooks/useConfirmModal";

type Props = {
  service_id: string;
  url_route_suffix: string;
};

export const DomainsSection: FC<Props> = ({
  service_id,
  url_route_suffix,
}) => {
  const queryClient = useQueryClient();
  const confirm = useConfirmModal();
  const { openModal } = useModal();
  const proxy = useProxyConfigs();
  const queryKey = useMemo(() => {
    return ["services", service_id, "domains"];
  }, [service_id]);

  const domainsQuery = useQuery({
    queryKey,
    queryFn: async () => {
      return domainsService.fetchAllByServiceID(service_id);
    },
  });

  const handleSuccess = () => {
    domainsQuery.refetch();
    queryClient.invalidateQueries({ queryKey: ["services", service_id] });
    queryClient.invalidateQueries({ queryKey: ["services"] });
  };

  const proxyDomain = useMemo((): DomainDto => {
    return {
      id: "__",
      service: "",
      domain: proxy.base_domain ? `${service_id}.${proxy.base_domain}` : "--",
      use_https: proxy.use_https ? "yes" : "no",
    };
  }, [proxy.base_domain, proxy.use_https, service_id]);

  const openCreateModal = () => {
    openModal(
      <DomainForm
        service_id={service_id}
        onSaveRecord={handleSuccess}
        use_https={proxy.use_https ? "yes" : "no"}
        width={360}
      />,
      {
        title: "Create Domain",
      },
    );
  };

  const openEditModal = (record: DomainDto) => {
    openModal(
      <DomainForm
        service_id={service_id}
        width={360}
        record={record}
        onSaveRecord={handleSuccess}
        use_https={proxy.use_https ? "yes" : "no"}
      />,
      {
        title: "Edit Domain",
      },
    );
  };

  const deleteMutation = useMutation({
    mutationFn: domainsService.deleteDomain,
    onSuccess: handleSuccess,
    onError: error => toast.error(getErrorMessage(error)),
  });

  const handleDelete = async (id: string) => {
    const ok = await confirm(
      "Delete domain",
      "Are you sure you want to delete this domain?",
    );
    if (ok) {
      deleteMutation.mutate(id);
    }
  };

  const requestSSLCertificate = useMutation({
    mutationFn: domainsService.createSSLRequest,
    onSuccess: handleSuccess,
    onError: error => toast.error(getErrorMessage(error)),
  });

  const handleCreateSSLRequest = async (domain: string) =>
    requestSSLCertificate.mutate(domain);

  if (domainsQuery.isFetching) {
    return <div className="p-4">Loading...</div>;
  }

  if (domainsQuery.isError)
    return (
      <ErrorFallback
        error={domainsQuery.error}
        onRetry={() => setTimeout(domainsQuery.refetch)}
      />
    );

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <DnsSetupGuideButton serverIp={proxy.base_domain ?? "YOUR_SERVER_IP"} />
        <button
          className="btn btn-sm btn-primary gap-2"
          onClick={openCreateModal}
        >
          <Plus className="w-4 h-4" />
          New instance
        </button>
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-4 min-w-0">
        <DomainCard
          readonly
          url={formatUrl(
            proxyDomain.use_https === "yes" ? "https" : "http",
            proxyDomain.domain,
            proxy.use_https ? proxy.https_port : proxy.http_port,
          )}
          port={proxy.use_https ? proxy.https_port : proxy.http_port}
          domain={proxyDomain}
          suffix={url_route_suffix}
        />
        {(domainsQuery.data ?? []).map(domain => (
          <DomainCard
            key={domain.id}
            domain={domain}
            port={proxy.use_https ? proxy.https_port : proxy.http_port}
            onEdit={() => openEditModal(domain)}
            onDelete={() => handleDelete(domain.id)}
            onValidate={() => handleCreateSSLRequest(domain.domain)}
            suffix={url_route_suffix}
          />
        ))}
      </div>

      {(domainsQuery.data ?? []).length === 0 && (
        <div className="rounded-lg border border-dashed border-base-300 p-6 text-center space-y-2">
          <p className="text-sm text-base-content/60">
            Solo está activo el dominio del sistema. Puedes añadir dominios personalizados con SSL.
          </p>
          <button
            className="btn btn-sm btn-primary gap-2"
            onClick={openCreateModal}
          >
            <Plus className="w-4 h-4" />
            Añadir dominio personalizado
          </button>
        </div>
      )}
    </div>
  );
};
