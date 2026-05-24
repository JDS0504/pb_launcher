import { ConfirmModalContext } from "./context";
import { useState, type ReactNode } from "react";

export const ConfirmModalProvider = ({ children }: { children: ReactNode }) => {
  const [isOpen, setIsOpen] = useState(false);
  const [resolve, setResolve] = useState<(value: boolean) => void>(() => {});
  const [title, setTitle] = useState<string>("Confirmation");
  const [message, setMessage] = useState<string>(
    "Are you sure you want to do this?",
  );

  const openModal = (title: string, message: string): Promise<boolean> => {
    setIsOpen(true);
    setTitle(title);
    setMessage(message);
    return new Promise(res => setResolve(() => res));
  };

  const closeModal = () => {
    setIsOpen(false);
  };

  const handleConfirm = () => {
    resolve(true);
    closeModal();
  };

  const handleCancel = () => {
    resolve(false);
    closeModal();
  };

  const Modal = () => {
    if (!isOpen) return null;

    return (
      <div className="fixed inset-0 z-[9999] flex items-center justify-center bg-black/50 p-2 sm:p-4">
        <div className="w-full max-w-sm rounded-lg bg-base-100 p-4 shadow-lg sm:p-6">
          <h3 className="text-lg font-bold mb-2">{title}</h3>
          <p className="mb-5 text-sm text-base-content/80 sm:mb-6 sm:text-base">
            {message}
          </p>
          <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end sm:gap-4">
            <button className="btn btn-ghost" onClick={handleCancel}>
              Cancel
            </button>
            <button className="btn btn-primary" onClick={handleConfirm}>
              Confirm
            </button>
          </div>
        </div>
      </div>
    );
  };

  return (
    <ConfirmModalContext.Provider value={{ openModal }}>
      <>
        <Modal />
        {children}
      </>
    </ConfirmModalContext.Provider>
  );
};
