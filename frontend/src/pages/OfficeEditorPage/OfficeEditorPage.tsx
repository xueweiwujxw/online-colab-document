import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  CasualSheets,
  type CasualSheetsAPI,
} from '@casualoffice/sheets/sheets';
import { EmbedHostTransport, type SaveRequestData } from '@casualoffice/sheets/embed';
import { HocuspocusProvider, HocuspocusProviderWebsocket } from '@hocuspocus/provider';
import '@casualoffice/sheets/styles';
import '@univerjs/sheets/facade';
import '@univerjs/sheets-ui/facade';
import * as Y from 'yjs';
import {
  workbookDataToXlsx,
  xlsxToWorkbookData,
  type ImportedWorkbook,
} from '@casualoffice/sheets/xlsx';

import {
  fetchOfficeContent,
  getOfficeCollabSession,
  getOfficeSession,
  saveOfficeContent,
  type OfficeCollabSession,
  type OfficeSession,
} from '../../api/office';
import { ApiError, errorMessage } from '../../api/client';
import { useAuth } from '../../auth/AuthContext';
import { SHEETS_LOCALE, SHEETS_LOCALES } from './sheetsLocale';

type EditorState =
  | { status: 'loading'; session: null; buffer: null; error: null }
  | { status: 'success'; session: OfficeSession; buffer: ArrayBuffer; error: null }
  | { status: 'error'; session: null; buffer: null; error: string };

type SaveState = 'idle' | 'saving' | 'saved' | 'error';
type CollabConnectionStatus = 'connecting' | 'live' | 'offline';
type RealtimeSnapshot = {
  clientId: string;
  clock: number;
  workbook: ImportedWorkbook;
};
type CollabState =
  | { status: 'idle'; session: null }
  | { status: 'loading'; session: null }
  | { status: 'success'; session: OfficeCollabSession; connectionStatus?: CollabConnectionStatus }
  | { status: 'error'; session: null };

export function OfficeEditorPage({ documentId }: { documentId: string }) {
  const auth = useAuth();
  const activeTransport = useRef<EmbedHostTransport | null>(null);
  const [state, setState] = useState<EditorState>({
    status: 'loading',
    session: null,
    buffer: null,
    error: null,
  });
  const [saveState, setSaveState] = useState<SaveState>('idle');
  const [saveError, setSaveError] = useState<string | null>(null);
  const [collabState, setCollabState] = useState<CollabState>({ status: 'idle', session: null });
  const onTransport = useCallback((transport: EmbedHostTransport | null) => {
    activeTransport.current = transport;
  }, []);
  const onCollabStatus = useCallback((connectionStatus: CollabConnectionStatus) => {
    setCollabState((current) =>
      current.status === 'success' ? { ...current, connectionStatus } : current,
    );
  }, []);
  const onSaveStart = useCallback(() => {
    setSaveState('saving');
    setSaveError(null);
  }, []);
  const onSaveSuccess = useCallback(() => setSaveState('saved'), []);
  const onSaveError = useCallback((message: string) => {
    setSaveState('error');
    setSaveError(message);
  }, []);

  useEffect(() => {
    if (auth.status !== 'authenticated') {
      return;
    }
    let mounted = true;
    setState({ status: 'loading', session: null, buffer: null, error: null });
    setSaveState('idle');
    setSaveError(null);
    setCollabState({ status: 'idle', session: null });
    getOfficeSession(documentId)
      .then(async (session) => {
        const buffer = await fetchOfficeContent(session.downloadUrl);
        if (!mounted) {
          return;
        }
        setState({ status: 'success', session, buffer, error: null });
        if (session.fileExt === 'xlsx') {
          setCollabState({ status: 'loading', session: null });
          try {
            const collabSession = await getOfficeCollabSession(documentId);
            if (mounted) {
              setCollabState({ status: 'success', session: collabSession });
            }
          } catch {
            if (mounted) {
              setCollabState({ status: 'error', session: null });
            }
          }
        } else if (mounted) {
          setCollabState({ status: 'idle', session: null });
        }
      })
      .catch((error: unknown) => {
        if (mounted) {
          setState({
            status: 'error',
            session: null,
            buffer: null,
            error:
              error instanceof ApiError && error.status === 400
                ? '当前 Casual Office POC 仅支持 docx / xlsx 在线编辑；doc / xls 需要转换或后续接入兼容提供商。'
                : errorMessage(error, '加载编辑器失败'),
          });
        }
      });
    return () => {
      mounted = false;
    };
  }, [auth.status, documentId]);

  if (auth.status === 'loading') {
    return (
      <main className="app-shell">
        <section className="empty-state">加载中</section>
      </main>
    );
  }

  if (auth.status === 'anonymous') {
    window.location.replace('/login');
    return null;
  }

  return (
    <main className="editor-shell">
      <header className="editor-topbar">
        <div className="office-title">
          <a className="back-link" href={`/documents/${documentId}`}>
            返回文档
          </a>
          <div>
            <strong>{state.status === 'success' ? providerName(state.session.fileExt) : 'Casual Office'}</strong>
            <span>
              {state.status === 'success'
                ? `${state.session.fileExt.toUpperCase()} 在线编辑器`
                : '正在加载新编辑器'}
            </span>
          </div>
        </div>
        <div className="office-toolbar">
          <span className="office-provider-badge">新编辑器</span>
          {state.status === 'success' ? (
            <span className="connection-pill">{state.session.mode === 'edit' ? '可编辑' : '只读'}</span>
          ) : null}
          {state.status === 'success' ? <span className="office-collab-note">{collabLabel(collabState)}</span> : null}
          {state.status === 'success' && state.session.fileExt === 'xlsx' ? (
            <span className="office-collab-note">双击单元格输入，拖拽选择后再格式化</span>
          ) : null}
          {saveError ? <span className="form-error office-save-error">{saveError}</span> : null}
          {state.status === 'success' && state.session.mode === 'edit' && state.session.fileExt !== 'xlsx' ? (
            <button
              className="secondary-button"
              disabled={saveState === 'saving'}
              onClick={() => {
                setSaveState('saving');
                setSaveError(null);
                activeTransport.current?.sendCommandSave();
              }}
              type="button"
            >
              {saveState === 'saving' ? '保存中' : saveState === 'saved' ? '已保存' : '保存'}
            </button>
          ) : null}
        </div>
      </header>
      <span aria-live="polite" className="screen-reader-text" role="status">
        {saveState === 'saving' ? '正在保存文档' : saveState === 'saved' ? '文档已保存' : saveState === 'error' ? '文档保存失败' : ''}
      </span>
      {state.status === 'loading' ? (
        <section className="empty-state editor-state">正在加载 Casual Office 编辑器</section>
      ) : null}
      {state.status === 'error' ? <section className="empty-state editor-state">{state.error}</section> : null}
      {state.status === 'success' ? (
        state.session.fileExt === 'xlsx' && collabState.status === 'loading' ? (
          <section className="empty-state editor-state">正在连接表格协同服务</section>
        ) : state.session.fileExt === 'xlsx' ? (
          <DirectSheetsHost
            buffer={state.buffer}
            collabSession={collabState.status === 'success' ? collabState.session : null}
            onCollabStatus={onCollabStatus}
            onSaveError={onSaveError}
            onSaveStart={onSaveStart}
            onSaveSuccess={onSaveSuccess}
            saveState={saveState}
            session={state.session}
          />
        ) : (
          <CasualIframeHost
            buffer={state.buffer}
            onSaveError={onSaveError}
            onSaveStart={onSaveStart}
            onSaveSuccess={onSaveSuccess}
            onTransport={onTransport}
            session={state.session}
          />
        )
      ) : null}
    </main>
  );
}

function DirectSheetsHost({
  session,
  buffer,
  collabSession,
  onCollabStatus,
  onSaveStart,
  onSaveSuccess,
  onSaveError,
  saveState,
}: {
  session: OfficeSession;
  buffer: ArrayBuffer;
  collabSession: OfficeCollabSession | null;
  onCollabStatus: (status: CollabConnectionStatus) => void;
  onSaveStart: () => void;
  onSaveSuccess: () => void;
  onSaveError: (message: string) => void;
  saveState: SaveState;
}) {
  const apiRef = useRef<CasualSheetsAPI | null>(null);
  const realtimeMapRef = useRef<Y.Map<RealtimeSnapshot> | null>(null);
  const suppressRealtimePublishUntil = useRef(0);
  const realtimeClientId = useRef(createClientId());
  const realtimeClock = useRef(0);
  const lastRemoteVersion = useRef<string | null>(null);
  const [sheetApi, setSheetApi] = useState<CasualSheetsAPI | null>(null);
  const [sheetState, setSheetState] = useState<
    | { status: 'loading'; workbook: null; error: null }
    | { status: 'success'; workbook: ImportedWorkbook; error: null }
    | { status: 'error'; workbook: null; error: string }
  >({ status: 'loading', workbook: null, error: null });

  useEffect(() => {
    let mounted = true;
    setSheetState({ status: 'loading', workbook: null, error: null });
    xlsxToWorkbookData(buffer.slice(0))
      .then((workbook) => {
        if (mounted) {
          setSheetState({ status: 'success', workbook, error: null });
        }
      })
      .catch((error: unknown) => {
        if (mounted) {
          setSheetState({
            status: 'error',
            workbook: null,
            error: errorMessage(error, '解析 xlsx 失败'),
          });
        }
      });
    return () => {
      mounted = false;
    };
  }, [buffer]);

  const saveSnapshot = useCallback(
    async (snapshot: ImportedWorkbook | null) => {
      if (session.mode !== 'edit') {
        onSaveError('只读模式不能保存');
        return;
      }
      if (!snapshot) {
        onSaveError('表格尚未加载完成');
        return;
      }
      try {
        onSaveStart();
        const blob = await workbookDataToXlsx(snapshot);
        const bytes = await blob.arrayBuffer();
        await saveOfficeContent(session.saveUrl, bytes);
        onSaveSuccess();
      } catch (error: unknown) {
        onSaveError(errorMessage(error, '保存 xlsx 失败'));
      }
    },
    [onSaveError, onSaveStart, onSaveSuccess, session.mode, session.saveUrl],
  );

  useEffect(() => {
    if (!sheetApi || !collabSession?.enabled || !collabSession.serverUrl) {
      realtimeMapRef.current = null;
      return;
    }

    onCollabStatus('connecting');
    const doc = new Y.Doc();
    const map = doc.getMap<RealtimeSnapshot>('office-snapshot');
    realtimeMapRef.current = map;
    const websocket = new HocuspocusProviderWebsocket({
      messageReconnectTimeout: 10_000,
      url: officeCollabURL(collabSession.serverUrl, collabSession.room, collabSession.role),
    });
    const provider = new HocuspocusProvider({
      document: doc,
      name: collabSession.room,
      token: 'anon',
      websocketProvider: websocket,
    });
    provider.attach();

    const applyRemoteSnapshot = (snapshot: RealtimeSnapshot | undefined) => {
      if (!snapshot || snapshot.clientId === realtimeClientId.current) {
        return;
      }
      const version = `${snapshot.clientId}:${snapshot.clock}`;
      if (lastRemoteVersion.current === version) {
        return;
      }
      lastRemoteVersion.current = version;
      suppressRealtimePublishUntil.current = Date.now() + 1000;
      sheetApi.setContent(snapshot.workbook);
      sheetApi.setDocumentMode(session.mode === 'edit' ? 'editing' : 'viewing');
    };

    const observer = () => applyRemoteSnapshot(map.get('latest'));
    map.observe(observer);
    provider.on('status', ({ status }: { status: string }) => {
      onCollabStatus(status === 'connected' ? 'live' : status === 'connecting' ? 'connecting' : 'offline');
    });
    provider.on('synced', ({ state }: { state: boolean }) => {
      if (!state) {
        return;
      }
      onCollabStatus('live');
      const latest = map.get('latest');
      if (latest) {
        applyRemoteSnapshot(latest);
        return;
      }
      if (collabSession.role === 'write') {
        const initial = sheetApi.getContent();
        if (initial) {
          realtimeClock.current += 1;
          map.set('latest', {
            clientId: realtimeClientId.current,
            clock: realtimeClock.current,
            workbook: initial,
          });
        }
      }
    });

    return () => {
      map.unobserve(observer);
      realtimeMapRef.current = null;
      provider.destroy();
      websocket.destroy();
      doc.destroy();
    };
  }, [collabSession, onCollabStatus, session.mode, sheetApi]);

  const publishRealtimeSnapshot = useCallback(
    (snapshot: ImportedWorkbook) => {
      const map = realtimeMapRef.current;
      if (!map || collabSession?.role !== 'write' || Date.now() < suppressRealtimePublishUntil.current) {
        return;
      }
      realtimeClock.current += 1;
      map.set('latest', {
        clientId: realtimeClientId.current,
        clock: realtimeClock.current,
        workbook: snapshot,
      });
    },
    [collabSession?.role],
  );

  if (sheetState.status === 'loading') {
    return <section className="empty-state editor-state">正在解析 xlsx 表格</section>;
  }
  if (sheetState.status === 'error') {
    return <section className="empty-state editor-state">{sheetState.error}</section>;
  }

  return (
    <section className="office-frame office-frame-direct">
      <SpreadsheetToolbar
        api={sheetApi}
        disabled={session.mode !== 'edit'}
        onSave={() => {
          void saveSnapshot(apiRef.current?.getContent() ?? null);
        }}
        saveState={saveState}
      />
      <div className="spreadsheet-canvas">
        <CasualSheets
          key={`${session.documentId}:${session.mode}`}
          appearance="light"
          chrome="none"
          documentMode={session.mode === 'edit' ? 'editing' : 'viewing'}
          initialData={sheetState.workbook}
          lazyPlugins={false}
          locale={SHEETS_LOCALE}
          locales={SHEETS_LOCALES}
          onError={(error) => onSaveError(error.message)}
          onChange={publishRealtimeSnapshot}
          onReady={(api) => {
            apiRef.current = api;
            api.focus();
            setSheetApi(api);
          }}
          onSave={(snapshot) => {
            void saveSnapshot(snapshot);
          }}
          readOnly={session.mode !== 'edit'}
          ui={{ header: false, toolbar: false, footer: false, contextMenu: true }}
        />
      </div>
    </section>
  );
}

const NUMBER_FORMATS = [
  { label: '常规', pattern: '' },
  { label: '数字', pattern: '#,##0.00' },
  { label: '整数', pattern: '#,##0' },
  { label: '货币', pattern: '¥#,##0.00' },
  { label: '百分比', pattern: '0.00%' },
  { label: '日期', pattern: 'yyyy-mm-dd' },
  { label: '文本', pattern: '@' },
];

function SpreadsheetToolbar({
  api,
  disabled,
  onSave,
  saveState,
}: {
  api: CasualSheetsAPI | null;
  disabled: boolean;
  onSave: () => void;
  saveState: SaveState;
}) {
  const controlsDisabled = disabled || !api;
  const runCommand = useCallback(
    (command: string, params?: object) => {
      if (controlsDisabled || !api) {
        return;
      }
      void api.executeCommand(command, params).then(() => api.focus());
    },
    [api, controlsDisabled],
  );
  const applyNumberFormat = useCallback(
    (pattern: string) => {
      if (controlsDisabled || !api) {
        return;
      }
      void api.executeCommand('sheet.command.numfmt.set.numfmt', { value: pattern }).then(() => api.focus());
    },
    [api, controlsDisabled],
  );

  return (
    <div aria-label="表格工具栏" className="spreadsheet-toolbar" role="toolbar">
      <div aria-label="历史与保存" className="spreadsheet-toolbar-group" role="group">
        <button
          aria-label="撤销"
          className="sheet-toolbar-button"
          disabled={controlsDisabled}
          onClick={() => runCommand('univer.command.undo')}
          title="撤销"
          type="button"
        >
          撤销
        </button>
        <button
          aria-label="重做"
          className="sheet-toolbar-button"
          disabled={controlsDisabled}
          onClick={() => runCommand('univer.command.redo')}
          title="重做"
          type="button"
        >
          重做
        </button>
        <button
          aria-label="保存表格"
          className="primary-button sheet-save-button"
          disabled={controlsDisabled || saveState === 'saving'}
          onClick={onSave}
          type="button"
        >
          {saveState === 'saving' ? '保存中' : saveState === 'saved' ? '已保存' : '保存'}
        </button>
      </div>
      <div aria-label="文字格式" className="spreadsheet-toolbar-group" role="group">
        <button aria-label="加粗" className="sheet-toolbar-button sheet-toolbar-strong" disabled={controlsDisabled} onClick={() => runCommand('sheet.command.set-range-bold')} title="加粗" type="button">
          加粗
        </button>
        <button aria-label="斜体" className="sheet-toolbar-button sheet-toolbar-italic" disabled={controlsDisabled} onClick={() => runCommand('sheet.command.set-range-italic')} title="斜体" type="button">
          斜体
        </button>
        <button aria-label="下划线" className="sheet-toolbar-button sheet-toolbar-underline" disabled={controlsDisabled} onClick={() => runCommand('sheet.command.set-range-underline')} title="下划线" type="button">
          下划线
        </button>
        <label className="sheet-toolbar-label">
          字号
          <select defaultValue="12" disabled={controlsDisabled} onChange={(event) => runCommand('sheet.command.set-range-fontsize', { value: Number(event.target.value) })}>
            {[10, 11, 12, 14, 16, 18, 24, 32].map((size) => (
              <option key={size} value={size}>{size}</option>
            ))}
          </select>
        </label>
        <label className="sheet-toolbar-label">
          文字色
          <input aria-label="文字颜色" defaultValue="#1f2937" disabled={controlsDisabled} onChange={(event) => runCommand('sheet.command.set-range-text-color', { value: event.target.value })} type="color" />
        </label>
        <label className="sheet-toolbar-label">
          填充色
          <input aria-label="填充颜色" defaultValue="#ffffff" disabled={controlsDisabled} onChange={(event) => runCommand('sheet.command.set-background-color', { value: event.target.value })} type="color" />
        </label>
      </div>
      <div aria-label="对齐与数字" className="spreadsheet-toolbar-group" role="group">
        <button aria-label="左对齐" className="sheet-toolbar-button" disabled={controlsDisabled} onClick={() => runCommand('sheet.command.set-horizontal-text-align', { value: 1 })} title="左对齐" type="button">左对齐</button>
        <button aria-label="居中" className="sheet-toolbar-button" disabled={controlsDisabled} onClick={() => runCommand('sheet.command.set-horizontal-text-align', { value: 2 })} title="居中" type="button">居中</button>
        <button aria-label="右对齐" className="sheet-toolbar-button" disabled={controlsDisabled} onClick={() => runCommand('sheet.command.set-horizontal-text-align', { value: 3 })} title="右对齐" type="button">右对齐</button>
        <label className="sheet-toolbar-label">
          数字格式
          <select defaultValue="" disabled={controlsDisabled} onChange={(event) => applyNumberFormat(event.target.value)}>
            {NUMBER_FORMATS.map(({ label, pattern }) => (
              <option key={label} value={pattern}>{label}</option>
            ))}
          </select>
        </label>
      </div>
    </div>
  );
}

function createClientId(): string {
  return window.crypto?.randomUUID?.() ?? `${Date.now()}:${Math.random().toString(36).slice(2)}`;
}

function officeCollabURL(serverURL: string, room: string, role: string): string {
  const normalizedServerURL = normalizeOfficeCollabURL(serverURL);
  const separator = normalizedServerURL.includes('?') ? '&' : '?';
  return `${normalizedServerURL}${separator}room=${encodeURIComponent(room)}&role=${role === 'view' ? 'view' : 'write'}`;
}

function normalizeOfficeCollabURL(serverURL: string): string {
  if (/^wss?:\/\/(localhost|127\.0\.0\.1):1234\/?$/i.test(serverURL)) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${protocol}//${window.location.host}/office-collab`;
  }
  return serverURL;
}

function CasualIframeHost({
  session,
  buffer,
  onTransport,
  onSaveStart,
  onSaveSuccess,
  onSaveError,
}: {
  session: OfficeSession;
  buffer: ArrayBuffer;
  onTransport: (transport: EmbedHostTransport | null) => void;
  onSaveStart: () => void;
  onSaveSuccess: () => void;
  onSaveError: (message: string) => void;
}) {
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const latestBuffer = useMemo(() => buffer.slice(0), [buffer]);
  const embedOrigin = window.location.origin;
  const app = session.fileExt === 'xlsx' ? 'sheet' : 'docs';
  const viewMode = session.mode === 'edit' ? 'editor' : 'preview';

  useEffect(() => {
    const iframe = iframeRef.current;
    if (!iframe?.contentWindow) {
      return;
    }
    if (session.fileExt !== 'docx' && session.fileExt !== 'xlsx') {
      return;
    }
    const transport = new EmbedHostTransport({
      app,
      iframeWindow: iframe.contentWindow,
      embedOrigin,
    });
    onTransport(transport);
    transport.on({
      onEditorReady: () => {
        transport.sendHostHello({ capabilities: ['load', 'save'] });
        transport.sendSetLocale({ locale: 'zh-CN' });
        transport.sendSetTheme({ theme: 'light' });
        transport.sendSetViewMode({ viewMode });
        transport.sendSetReadOnly({ readOnly: session.mode !== 'edit' });
      },
      onLoadRequest: () => ({
        ok: true,
        bytes: latestBuffer.slice(0),
        fileName: session.title,
        readOnly: session.mode !== 'edit',
      }),
      onSaveRequest: async (request: SaveRequestData) => {
        if (session.mode !== 'edit') {
          return { ok: false, code: 'readonly', message: '只读模式不能保存' };
        }
        onSaveStart();
        const result = await saveOfficeContent(session.saveUrl, request.bytes);
        onSaveSuccess();
        return { ok: true, etag: result.etag };
      },
      onError: (error: { message: string }) => {
        onSaveError(error.message);
      },
    });
    const helloTimer = window.setTimeout(() => {
      transport.sendHostHello({ capabilities: ['load', 'save'] });
    }, 100);
    return () => {
      window.clearTimeout(helloTimer);
      transport.destroy();
      onTransport(null);
    };
  }, [app, embedOrigin, latestBuffer, onSaveError, onSaveStart, onSaveSuccess, onTransport, session, viewMode]);

  if (session.fileExt !== 'docx' && session.fileExt !== 'xlsx') {
    return <section className="empty-state editor-state">当前 POC 仅支持 docx / xlsx。</section>;
  }

  return (
    <section className="office-frame">
      <iframe
        className="office-sheet-iframe"
        ref={iframeRef}
        src={`${embedBasePath(session.fileExt)}/embed.html?app=${app}&docId=${encodeURIComponent(
          session.documentId,
        )}&viewMode=${viewMode}`}
        title={session.title}
      />
    </section>
  );
}

function embedBasePath(fileExt: string): string {
  return fileExt === 'xlsx' ? '/casual-sheets' : '/casual-docs';
}

function providerName(fileExt: string): string {
  return fileExt === 'xlsx' ? '表格编辑器' : '文档编辑器';
}

function collabLabel(state: CollabState): string {
  if (state.status === 'loading') {
    return '协同检测中';
  }
  if (state.status === 'success' && state.session.enabled) {
    if (state.connectionStatus === 'connecting') {
      return '协同连接中';
    }
    if (state.connectionStatus === 'live') {
      return state.session.role === 'write' ? '协同已连接' : '协同只读在线';
    }
    if (state.connectionStatus === 'offline') {
      return '协同离线';
    }
    return state.session.role === 'write' ? '实时协同已连接' : '协同只读在线';
  }
  return '单人编辑';
}
