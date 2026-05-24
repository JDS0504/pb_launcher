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
                    "modal-box w-[calc(100vw-1rem)] max-w-[calc(100vw-1rem)] max-h-[calc(100dvh-1rem)] sm:w-full sm:max-w-[calc(100vw-2rem)] sm:max-h-[calc(100dvh-2rem)]",
                    "flex flex-col overflow-hidden",
                    "bg-base-100 text-base-content shadow-xl",
                    "p-3 sm:p-5 md:p-6",
                  )}
                  style={modalStyle}
                  onClick={e => e.stopPropagation()}
                >
                  {!disableCloseButton && (
                    <div className="absolute right-2 top-2 z-10 sm:right-4 sm:top-4">
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
                    <div className="mb-3 pr-10 sm:mb-4">
                      <h3 className="text-lg font-semibold sm:text-xl">{title}</h3>
                    </div>
                  )}
                  <div className="min-w-0 flex-1 overflow-y-auto">{modalBody}</div>
                </div>
              </dialog>
            );
          },
        )}
      </ModalContext.Provider>
    </div>
  );
};
