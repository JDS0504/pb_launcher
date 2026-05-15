import {
  useCallback,
  useState,
  type ReactNode,
  type CSSProperties,
} from "react";
import {
  ModalContext,
  type ModalComponent,
  type ModalContextType,
} from "./context";
import { X } from "lucide-react";
import classNames from "classnames";

export const ModalProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  const [stack, setStack] = useState<ModalComponent[]>([]);

  const openModal = useCallback<ModalContextType["openModal"]>(
    (content, props) => {
      setStack(prev => [...prev, { content, ...(props ?? {}) }]);
    },
    [],
  );

  const closeModal = useCallback(() => setStack(prev => prev.slice(0, -1)), []);
  const closeAllModals = useCallback(() => setStack([]), []);

  return (
    <div>
      <ModalContext.Provider value={{ openModal, closeModal, closeAllModals }}>
        {children}
        {stack.map(
          (
            {
              content,
              title,
              width,
              height,
              zIndex,
              closeOnBackdropClick,
              disableCloseButton,
            },
            index,
          ) => {
            const modalBody =
              typeof content === "function" ? content(index) : content;

            const modalStyle: CSSProperties = {
              width: width || "auto",
              height: height || "auto",
            };

            const overlayStyle: CSSProperties = {
              zIndex: zIndex ?? 50,
            };

            return (
              <dialog
                open
                key={index}
                className="modal modal-open"
                style={overlayStyle}
                onClick={closeOnBackdropClick ? closeModal : undefined}
              >
                <div
                  className={classNames(
                    "modal-box w-full max-w-[calc(100vw-2rem)] max-h-[calc(100vh-2rem)]",
                    "flex flex-col overflow-hidden",
                    "bg-base-100 text-base-content shadow-xl",
                    "p-4 sm:p-6",
                  )}
                  style={modalStyle}
                  onClick={e => e.stopPropagation()}
                >
                  {!disableCloseButton && (
                    <div className="absolute right-4 top-4 z-10">
                      <button
                        onClick={closeModal}
                        className="btn btn-sm btn-circle btn-ghost"
                        aria-label="Cerrar"
                      >
                        <X className="w-5 h-5" />
                      </button>
                    </div>
                  )}
                  {title && (
                    <div className="mb-4">
                      <h3 className="text-xl font-semibold">{title}</h3>
                    </div>
                  )}
                  <div className="flex-1 overflow-y-auto">{modalBody}</div>
                </div>
              </dialog>
            );
          },
        )}
      </ModalContext.Provider>
    </div>
  );
};
