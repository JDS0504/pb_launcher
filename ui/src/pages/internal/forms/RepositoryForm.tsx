import { useState, type FormEvent } from "react";
import { useMutation } from "@tanstack/react-query";
import toast from "react-hot-toast";
import { Button } from "../../../components/buttons/Button";
import { useModal } from "../../../components/modal/hook";
import {
  DEFAULT_REPOSITORY_ID,
  repositoriesService,
  type RepositoryDto,
  type RepositoryPayload,
} from "../../../services/repositories";
import { getErrorMessage } from "../../../utils/errors";

type Props = {
  repository?: RepositoryDto;
  onSave?: () => void;
};

const defaultPayload: RepositoryPayload = {
  name: "",
  repository: "",
  token: "",
  retention: 3,
  release_file_pattern: String.raw`pocketbase_.+_linux_amd64\.zip`,
  exec_file_pattern: String.raw`^pocketbase`,
  disabled: false,
};

export const RepositoryForm = ({ repository, onSave }: Props) => {
  const { closeModal } = useModal();
  const isDefault = repository?.id === DEFAULT_REPOSITORY_ID;
  const [form, setForm] = useState<RepositoryPayload>({
    ...defaultPayload,
    ...repository,
    retention: Math.max(repository?.retention ?? defaultPayload.retention, 1),
  });

  const saveMutation = useMutation({
    mutationFn: async () => {
      const payload = {
        ...form,
        retention: Math.max(Number(form.retention) || 1, 1),
      };
      if (repository == null) return repositoriesService.create(payload);
      if (isDefault) {
        return repositoriesService.update(repository.id, {
          retention: payload.retention,
        });
      }
      return repositoriesService.update(repository.id, payload);
    },
    onSuccess: () => {
      toast.success("Repository saved successfully");
      closeModal();
      onSave?.();
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const setValue = <K extends keyof RepositoryPayload>(
    key: K,
    value: RepositoryPayload[K],
  ) => setForm(prev => ({ ...prev, [key]: value }));

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    saveMutation.mutate();
  };

  return (
    <form onSubmit={submit} className="space-y-4">
      {isDefault && (
        <div className="alert alert-info text-sm">
          The default PocketBase repository is protected. Only retention can be edited.
        </div>
      )}
      <TextInput
        label="Name"
        value={form.name}
        disabled={isDefault}
        onChange={value => setValue("name", value)}
      />
      <TextInput
        label="Repository"
        value={form.repository}
        disabled={isDefault}
        placeholder="owner/repository"
        onChange={value => setValue("repository", value)}
      />
      <TextInput
        label="GitHub Token"
        value={form.token}
        disabled={isDefault}
        type="password"
        onChange={value => setValue("token", value)}
      />
      <TextInput
        label="Retention"
        value={String(form.retention)}
        type="number"
        min={1}
        onChange={value => setValue("retention", Math.max(Number(value) || 1, 1))}
      />
      <TextInput
        label="Release File Pattern"
        value={form.release_file_pattern}
        disabled={isDefault}
        placeholder={`pocketbase_.+_linux_amd64\\.zip`}
        onChange={value => setValue("release_file_pattern", value)}
      />
      <TextInput
        label="Exec File Pattern"
        value={form.exec_file_pattern}
        disabled={isDefault}
        placeholder="^pocketbase"
        onChange={value => setValue("exec_file_pattern", value)}
      />
      {!isDefault && (
        <label className="label cursor-pointer justify-start gap-3">
          <input
            type="checkbox"
            className="checkbox"
            checked={form.disabled}
            onChange={event => setValue("disabled", event.target.checked)}
          />
          <span className="label-text">Disabled</span>
        </label>
      )}
      <Button type="submit" label="Save" loading={saveMutation.isPending} />
    </form>
  );
};

type TextInputProps = {
  label: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  type?: string;
  min?: number;
  placeholder?: string;
};

const TextInput = ({
  label,
  value,
  onChange,
  disabled,
  type = "text",
  min,
  placeholder,
}: TextInputProps) => {
  return (
    <div className="form-control w-full">
      <label className="label">
        <span className="label-text mb-1">{label}</span>
      </label>
      <input
        className="input input-bordered w-full"
        value={value}
        disabled={disabled}
        type={type}
        min={min}
        placeholder={placeholder}
        onChange={event => onChange(event.target.value)}
      />
    </div>
  );
};
