import { createContext, useContext, type ReactNode } from 'react';

/**
 * Contract for upload: applications provide the implementation; 
 * the renderer only calls these callbacks
 */
export interface UploadResponse {
  key: string;
  name?: string;
}

export type UploadHandler = (file: File) => Promise<UploadResponse>;
export type ViewFileHandler = (key: string) => Promise<string>;

export interface UploadContextValue {
  onUpload?: UploadHandler;
  viewFile?: ViewFileHandler;
}

const UploadContext = createContext<UploadContextValue | null>(null);

export function UploadProvider({
  children,
  onUpload,
  viewFile,
}: {
  children: ReactNode;
  onUpload?: UploadHandler;
  viewFile?: ViewFileHandler;
}) {
  const value: UploadContextValue = { onUpload, viewFile };
  return (
    <UploadContext.Provider value={value}>
      {children}
    </UploadContext.Provider>
  );
}

export function useUpload(): UploadContextValue | null {
  return useContext(UploadContext);
}
