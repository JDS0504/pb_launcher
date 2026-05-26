import { useMemo } from "react";
import { formatUrl } from "../utils/url";
import type { ServiceDto } from "../services/services";
import type { ProxyConfigsResponse } from "../services/config";

/**
 * Computa la lista de URLs de acceso de una instancia de servicio.
 * Da preferencia a los dominios custom registrados; el auto-domain
 * (id.base_domain) se añade al final si base_domain está configurado.
 *
 * Filtra dominios vacíos o inválidos para evitar URLs con "undefined".
 */
export const useServiceUrls = (
  service: ServiceDto | undefined,
  proxyInfo: ProxyConfigsResponse,
): string[] => {
  return useMemo((): string[] => {
    if (!service) return [];

    const customDomains = (service.domains ?? [])
      .map(d => d.domain)
      .filter(Boolean);

    const allDomains = [...customDomains];

    if (allDomains.length === 0 && proxyInfo.base_domain) {
      const autoDomain = `${service.id}.${proxyInfo.base_domain}`;
      allDomains.push(autoDomain);
    }

    return allDomains
      .map(domain => {
        const customDom = service.domains?.find(d => d.domain === domain);
        const useHttps = customDom
          ? customDom.use_https === "yes"
          : proxyInfo.use_https ?? false;

        const urlStr = formatUrl(
          useHttps ? "https" : "http",
          domain,
          useHttps ? proxyInfo.https_port : proxyInfo.http_port,
        );

        if (!urlStr) return null;

        if (service._pb_install)
          return `${urlStr}/_/#/pbinstal/${service._pb_install}`;
        return `${urlStr}/_/`;
      })
      .filter((url): url is string => url !== null && url !== "/_/");
  }, [proxyInfo, service]);
};
