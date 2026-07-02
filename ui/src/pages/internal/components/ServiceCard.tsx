import { useRef, type FC } from "react";
import type { ServiceDto } from "../../../services/services";
import {
  Copy,
  Eye,
  MoreVertical,
  Power,
  Save,
  ShieldAlert,
  Trash2,
  Upload,
  ExternalLink,
} from "lucide-react";
import classNames from "classnames";
import { useModal } from "../../../components/modal/hook";
import { DefaultCredentialsCard } from "./DefaultCredentialsCard";
import type { ProxyConfigsResponse } from "../../../services/config";
import { useServiceUrls } from "../../../hooks/useServiceUrls";

type Props = {
  proxyInfo: ProxyConfigsResponse;
  service: ServiceDto;
  onDetails: () => void;
  onDelete: () => void;
  onStart: () => void;
  onStop: () => void;
  onRestart: () => void;
  onUpgrade: () => void;
  onBackup: () => void;
  onClone: () => void;
  refreshData: () => void;
};

export const ServiceCard: FC<Props> = ({
  proxyInfo,
  service,
  onDetails,
  onDelete,
  onRestart,
  onStart,
  onStop,
  onUpgrade,
  onBackup,
  onClone,
  refreshData,
}) => {
  const dropdownRef = useRef<HTMLDivElement>(null);
  const { openModal } = useModal();
  const serviceUrls = useServiceUrls(service, proxyInfo);
  const adminUrl = serviceUrls.length > 0 ? serviceUrls[0] : null;

  const executeAfterBlur = (fn: () => void) => {
    (document.activeElement as HTMLElement)?.blur?.();
    fn();
  };

  const showDefaultCredentials = () => {
    openModal(
      <DefaultCredentialsCard
        service_id={service.id}
        username={service.boot_user_email}
        password={service.boot_user_password}
        onResetCredentials={refreshData}
      />,
    );
  };

  return (
    <div
      key={service.id}
      className="card bg-base-100 shadow border border-base-300"
    >
      <div className="card-body">
        <div className="flex justify-between items-start">
          <h2 className="card-title text-base-content">{service.name}</h2>
          <div className="flex gap-4 items-center">
            <ShieldAlert
              className="w-4 h-4 active:translate-[0.5px] relative -right-3 -top-2 text-gray-300"
              onClick={showDefaultCredentials}
            />

            <div
              ref={dropdownRef}
              className="dropdown dropdown-end select-none relative -right-3 -top-2"
            >
              <label
                tabIndex={0}
                className="btn btn-sm btn-ghost btn-circle text-base-content"
              >
                <MoreVertical className="w-4 h-4" />
              </label>
              <ul
                tabIndex={0}
                className="dropdown-content menu p-2 shadow-lg bg-base-100 text-base-content rounded-box min-w-[10rem] z-[1] space-y-1"
              >
                <li>
                  <button
                    disabled={
                      service.status === "running" ||
                      service.status === "pending"
                    }
                    className={classNames(
                      "flex items-center gap-2 w-full justify-start px-2 py-1 rounded-md text-left",
                      service.status === "running" ||
                        service.status === "pending"
                        ? "text-base-content/40 cursor-not-allowed"
                        : "text-success hover:bg-success/10 hover:text-success",
                    )}
                    onClick={() => executeAfterBlur(onStart)}
                  >
                    <Power className="w-4 h-4 text-success" />
                    Start
                  </button>
                </li>
                 <li>
                  <button
                    disabled={service.status !== "running" && service.status !== "sleeping"}
                    className={classNames(
                      "flex items-center gap-2 w-full justify-start px-2 py-1 rounded-md text-left",
                      service.status !== "running" && service.status !== "sleeping"
                        ? "text-base-content/40 cursor-not-allowed"
                        : "text-warning hover:bg-warning/10 hover:text-warning",
                    )}
                    onClick={() => executeAfterBlur(onRestart)}
                  >
                    <Power className="w-4 h-4 text-warning" />
                    Restart
                  </button>
                </li>
                <li>
                  <button
                    disabled={service.status !== "running" && service.status !== "sleeping"}
                    className={classNames(
                      "flex items-center gap-2 w-full justify-start px-2 py-1 rounded-md text-left",
                      service.status !== "running" && service.status !== "sleeping"
                        ? "text-base-content/40 cursor-not-allowed"
                        : "text-error hover:bg-error/10 hover:text-error",
                    )}
                    onClick={() => executeAfterBlur(onStop)}
                  >
                    <Power className="w-4 h-4 text-error" />
                    Stop
                  </button>
                </li>
                {service.status === "stopped" && (
                  <>
                    <div className="border-t border-base-300 my-1"></div>
                    <li>
                      <button
                        onClick={() => executeAfterBlur(onUpgrade)}
                        className="flex items-center gap-2 w-full justify-start text-info hover:bg-base-200"
                      >
                        <Upload className="w-4 h-4" />
                        Upgrade
                      </button>
                    </li>
                    <li>
                      <button
                        onClick={() => executeAfterBlur(onClone)}
                        className="flex items-center gap-2 w-full justify-start text-accent hover:bg-base-200"
                      >
                        <Copy className="w-4 h-4" />
                        Clone
                      </button>
                    </li>
                    <li>
                      <button
                        onClick={() => executeAfterBlur(onBackup)}
                        className="flex items-center gap-2 w-full justify-start text-secondary hover:bg-base-200"
                      >
                        <Save className="w-4 h-4" />
                        Backup
                      </button>
                    </li>
                  </>
                )}
                <div className="border-t border-base-300 my-1"></div>
                <li>
                  <button
                    onClick={() => executeAfterBlur(onDelete)}
                    className="flex items-center gap-2 w-full justify-start text-error hover:bg-base-200"
                  >
                    <Trash2 className="w-4 h-4" />
                    Delete
                  </button>
                </li>
              </ul>
            </div>
          </div>
        </div>

        <div className="text-sm space-y-1 text-base-content/80">
          <div className="flex justify-between">
            <span className="font-medium">Version:</span>
            <span>{`${service.repository} v${service.release_version}`}</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="font-medium">Status:</span>
            <div className="flex items-center gap-1.5">
              <span
                className={classNames("badge badge-sm", {
                  "badge-success": service.status === "running",
                  "badge-info": service.status === "sleeping",
                  "badge-warning":
                    service.status === "pending" || service.status === "idle",
                  "badge-error": service.status === "failure",
                  "badge-neutral": ![
                    "running",
                    "pending",
                    "idle",
                    "failure",
                    "sleeping",
                  ].includes(service.status),
                })}
              >
                {service.status.charAt(0).toUpperCase() + service.status.slice(1)}
              </span>
            </div>
          </div>

          <div className="flex justify-between">
            <span className="font-medium">Started:</span>
            <span>{new Date(service.created).toLocaleString()}</span>
          </div>
          <div className="flex justify-between text-xs">
            <span className="font-medium">Restart Policy:</span>
            <span className="capitalize">{service.restart_policy}</span>
          </div>
        </div>
        <div className="card-actions justify-end mt-4 flex gap-2 w-full">
          <button
            className="btn btn-sm btn-neutral flex-1 flex items-center justify-center gap-2"
            onClick={onDetails}
          >
            <Eye className="w-4 h-4" />
            Details
          </button>
          {adminUrl && (
            <a
              id={`btn-open-admin-${service.id}`}
              href={adminUrl}
              target="_blank"
              rel="noreferrer"
              className="btn btn-sm btn-primary flex-1 flex items-center justify-center gap-2 text-primary-content"
              title="Abrir PocketBase Admin en nueva pestaña"
            >
              <ExternalLink className="w-4 h-4" />
              Abrir Admin
            </a>
          )}
        </div>
      </div>
    </div>
  );
};
