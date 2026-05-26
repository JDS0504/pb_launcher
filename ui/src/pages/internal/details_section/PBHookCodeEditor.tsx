import CodeMirror from "@uiw/react-codemirror";
import { javascript } from "@codemirror/lang-javascript";
import { oneDark } from "@codemirror/theme-one-dark";

type Props = {
  value: string;
  editable: boolean;
  onChange: (value: string) => void;
};

export const PBHookCodeEditor = ({ value, editable, onChange }: Props) => {
  return (
    <div className="w-full h-full min-h-0 flex flex-col">
      <CodeMirror
        value={value}
        height="100%"
        className="w-full flex-1 min-h-0"
        theme={oneDark}
        extensions={[javascript({ jsx: false, typescript: false })]}
        editable={editable}
        basicSetup={{
          foldGutter: true,
          lineNumbers: true,
          highlightActiveLine: true,
          highlightSelectionMatches: true,
        }}
        onChange={onChange}
      />
    </div>
  );
};
