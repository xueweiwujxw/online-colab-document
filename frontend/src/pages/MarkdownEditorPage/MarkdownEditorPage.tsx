import { OfficeEditorPage } from '../OfficeEditorPage/OfficeEditorPage';

// Markdown is opened by Casual Docs too. Its official WASM converter preserves
// the .md file on save while exposing the same rich-text tables and Yjs
// collaboration primitives as the DOCX editor.
export function MarkdownEditorPage({ documentId }: { documentId: string }) {
  return <OfficeEditorPage documentId={documentId} />;
}
