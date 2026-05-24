import { object } from "yup";
import {
  booleanRequired,
  stringRequired,
  urlRequired,
} from "../../../utils/validation";
import { useCustomForm } from "../../../hooks/useCustomForm";
import { InputField } from "../../../components/fields/InputField";
import { Button } from "../../../components/buttons/Button";
import { useMutation } from "@tanstack/react-query";
import { type FC } from "react";
import { useModal } from "../../../components/modal/hook";
import toast from "react-hot-toast";
import { getErrorMessage } from "../../../utils/errors";
import classNames from "classnames";
import { proxyEntryService, type ProxyEntryDto } from "../../../services/proxy";

const schema = object({
  name: stringRequired(),
  target_url: urlRequired(),
  enabled: booleanRequired(), //yes, no
});

type Props = {
  record?: ProxyEntryDto;
  onSaveRecord?: () => void;
  width?: number;
};

export const ProxyEntryForm: FC<Props> = ({ onSaveRecord, record, width }) => {
  const { closeModal } = useModal();
  const form = useCustomForm(schema, {
    defaultValues: {
      name: record?.name,
      target_url: record?.target_url,
      enabled: record?.enabled === "yes",
    },
  });

  const createMutation = useMutation({
    mutationFn: proxyEntryService.create,
    onSuccess: () => {
      toast.success("Entry created successfully");
      closeModal();
      onSaveRecord?.();
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const updateMutation = useMutation({
    mutationFn: proxyEntryService.update,
    onSuccess: () => {
      toast.success("Entry updated successfully");
      closeModal();
      onSaveRecord?.();
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const handleFormSubmit = form.handleSubmit(
    ({ name, target_url, enabled }) => {
      if (record == null)
        createMutation.mutate({
          name,
          target_url,
        });
      else
        updateMutation.mutate({
          id: record.id,
          name,
          target_url,
          enabled: enabled ? "yes" : "no",
        });
    },
  );

  return (
    <div style={{ width, maxWidth: "100%" }}>
      <form onSubmit={handleFormSubmit} className="space-y-5">
        <div>
          {record != null && (
            <div className="flex justify-end">
              <InputField
                label="Enabled"
                type="checkbox"
                registration={form.register("enabled")}
                error={form.formState.errors.enabled}
              />
            </div>
          )}
          <InputField
            label="Entry Name"
            registration={form.register("name")}
            autoComplete="off"
            error={form.formState.errors.name}
          />
        </div>

        <InputField
          label="Target Url"
          registration={form.register("target_url")}
          autoComplete="off"
          error={form.formState.errors.target_url}
          placeholder="http://127.0.0.1:8080"
        />

        <div
          className={classNames("mt-8", {
            "flex justify-end": width == null || width > 400,
          })}
        >
          <div
            className={classNames("form-control", {
              "w-[200px]": width == null || width > 400,
            })}
          >
            <Button type="submit" label="Guardar" loading={false} />
          </div>
        </div>
      </form>
    </div>
  );
};
