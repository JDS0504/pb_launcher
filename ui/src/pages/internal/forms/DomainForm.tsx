import { object } from "yup";
import { booleanRequired, stringRequired } from "../../../utils/validation";
import { useCustomForm } from "../../../hooks/useCustomForm";
import type { FC } from "react";
import { InputField } from "../../../components/fields/InputField";
import { Button } from "../../../components/buttons/Button";
import classNames from "classnames";
import { useMutation } from "@tanstack/react-query";
import {
  domainsService,
  type DomainDto,
} from "../../../services/services_domain";
import toast from "react-hot-toast";
import { useModal } from "../../../components/modal/hook";
import { getErrorMessage } from "../../../utils/errors";

const domainRegex = /^(?!:\/\/)([a-zA-Z0-9-_]+\.)+[a-zA-Z]{2,}$/;
const schema = object({
  domain: stringRequired().matches(domainRegex, {
    message: "Invalid domain format",
  }),
  use_https: booleanRequired(),
});

type Props = {
  service_id: string;
  use_https: "yes" | "no";
  record?: DomainDto;
  onSaveRecord?: () => void;
  width?: number;
};

export const DomainForm: FC<Props> = ({
  service_id,
  record,
  width,
  use_https,
  onSaveRecord,
}) => {
  const { closeModal } = useModal();
  const form = useCustomForm(schema, {
    defaultValues: {
      domain: record?.domain,
      use_https: (record?.use_https ?? use_https) === "yes",
    },
  });

  const createMutation = useMutation({
    mutationFn: domainsService.createDomain,
    onSuccess: () => {
      toast.success("Domain added to the service successfully");
      closeModal();
      onSaveRecord?.();
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const updateMutation = useMutation({
    mutationFn: domainsService.updateDomain,
    onSuccess: () => {
      toast.success("Domain link updated successfully");
      closeModal();
      onSaveRecord?.();
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const handleFormSubmit = form.handleSubmit(async ({ domain, use_https }) => {
    if (record == null)
      createMutation.mutate({
        domain,
        service: service_id,
        use_https,
      });
    else updateMutation.mutate({ id: record.id, use_https });
  });

  const isWide = !width || width > 400;
  return (
    <div style={{ width: width ?? 300, maxWidth: "100%" }}>
      <form onSubmit={handleFormSubmit} className="space-y-5">
        <InputField
          label="Domain"
          registration={form.register("domain")}
          autoComplete="off"
          error={form.formState.errors.domain}
        />

        <InputField
          label="Use HTTPS"
          registration={form.register("use_https")}
          autoComplete="off"
          type="checkbox"
          error={form.formState.errors.use_https}
        />

        <div className={classNames("mt-6", { "flex justify-end": isWide })}>
          <div className={classNames("form-control", { "w-[200px]": isWide })}>
            <Button type="submit" label="Save" loading={false} />
          </div>
        </div>
      </form>
    </div>
  );
};
