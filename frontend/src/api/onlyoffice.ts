import { getJSON } from './client';

export type OnlyOfficeConfig = {
  documentServerUrl: string;
  document: {
    fileType: string;
    key: string;
    title: string;
    url: string;
  };
  documentType: string;
  editorConfig: {
    mode: 'view' | 'edit';
    callbackUrl: string;
    user: {
      id: string;
      name: string;
    };
  };
  type: string;
  token?: string;
};

export function getOnlyOfficeConfig(documentId: string): Promise<OnlyOfficeConfig> {
  return getJSON<OnlyOfficeConfig>(`/api/documents/${documentId}/onlyoffice/config`);
}
