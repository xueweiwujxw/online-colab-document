import { useEffect, useRef, useState } from 'react';
import {
  baseKeymap,
  chainCommands,
  exitCode,
  newlineInCode,
  setBlockType,
  toggleMark,
  wrapIn,
} from 'prosemirror-commands';
import { dropCursor } from 'prosemirror-dropcursor';
import { history, redo, undo } from 'prosemirror-history';
import { keymap } from 'prosemirror-keymap';
import {
  DOMParser as ProseMirrorDOMParser,
  Fragment,
  Node as ProseMirrorNode,
  Schema,
  type Mark,
  type NodeSpec,
} from 'prosemirror-model';
import { EditorState, type Command } from 'prosemirror-state';
import { EditorView } from 'prosemirror-view';
import { tableEditing, tableNodes } from 'prosemirror-tables';
import * as Y from 'yjs';

import {
  documentDownloadURL,
  getMarkdownSnapshot,
  markdownWebSocketURL,
  saveMarkdownDocument,
  type PresenceUser,
} from '../../api/documents';
import { errorMessage } from '../../api/client';
import { useAuth } from '../../auth/AuthContext';

type ConnectionState = 'loading' | 'connected' | 'disconnected' | 'reconnecting' | 'error';
type SaveState = 'idle' | 'saving' | 'saved' | 'error';

type ServerMessage = {
  type: 'init' | 'update' | 'presence' | 'error';
  content?: string;
  canEdit?: boolean;
  updateSeq?: number;
  users?: PresenceUser[];
  error?: string;
};

const markdownSchema = new Schema({
  nodes: {
    doc: { content: 'block+' },
    paragraph: {
      content: 'inline*',
      group: 'block',
      parseDOM: [{ tag: 'p' }],
      toDOM: () => ['p', 0],
    },
    blockquote: {
      content: 'block+',
      group: 'block',
      defining: true,
      parseDOM: [{ tag: 'blockquote' }],
      toDOM: () => ['blockquote', 0],
    },
    heading: {
      attrs: { level: { default: 1 } },
      content: 'inline*',
      group: 'block',
      defining: true,
      parseDOM: [
        { tag: 'h1', attrs: { level: 1 } },
        { tag: 'h2', attrs: { level: 2 } },
        { tag: 'h3', attrs: { level: 3 } },
      ],
      toDOM: (node) => [`h${node.attrs.level}`, 0],
    },
    code_block: {
      content: 'text*',
      marks: '',
      group: 'block',
      code: true,
      defining: true,
      parseDOM: [{ tag: 'pre', preserveWhitespace: 'full' }],
      toDOM: () => ['pre', ['code', 0]],
    },
    bullet_list: {
      content: 'list_item+',
      group: 'block',
      parseDOM: [{ tag: 'ul' }],
      toDOM: () => ['ul', 0],
    },
    ordered_list: {
      content: 'list_item+',
      group: 'block',
      parseDOM: [{ tag: 'ol' }],
      toDOM: () => ['ol', 0],
    },
    list_item: {
      content: 'paragraph block*',
      defining: true,
      parseDOM: [{ tag: 'li' }],
      toDOM: () => ['li', 0],
    },
    ...tableNodes({
      tableGroup: 'block',
      cellContent: 'block+',
      cellAttributes: {},
    }),
    text: { group: 'inline' },
    hard_break: {
      inline: true,
      group: 'inline',
      selectable: false,
      parseDOM: [{ tag: 'br' }],
      toDOM: () => ['br'],
    },
  } satisfies Record<string, NodeSpec>,
  marks: {
    strong: {
      parseDOM: [{ tag: 'strong' }, { tag: 'b' }],
      toDOM: () => ['strong', 0],
    },
    em: {
      parseDOM: [{ tag: 'em' }, { tag: 'i' }],
      toDOM: () => ['em', 0],
    },
    strike: {
      parseDOM: [{ tag: 's' }, { tag: 'del' }],
      toDOM: () => ['s', 0],
    },
    code: {
      parseDOM: [{ tag: 'code' }],
      toDOM: () => ['code', 0],
    },
    link: {
      attrs: { href: {} },
      inclusive: false,
      parseDOM: [{ tag: 'a[href]', getAttrs: (node) => ({ href: (node as HTMLElement).getAttribute('href') }) }],
      toDOM: (mark) => ['a', { href: mark.attrs.href, rel: 'noreferrer', target: '_blank' }, 0],
    },
  },
});

export function MarkdownEditorPage({ documentId }: { documentId: string }) {
  const auth = useAuth();
  const docRef = useRef(new Y.Doc());
  const textRef = useRef(docRef.current.getText('markdown'));
  const reconnectRef = useRef<number | null>(null);
  const socketRef = useRef<WebSocket | null>(null);
  const [draft, setDraft] = useState('');
  const [canEdit, setCanEdit] = useState(false);
  const [users, setUsers] = useState<PresenceUser[]>([]);
  const [connection, setConnection] = useState<ConnectionState>('loading');
  const [error, setError] = useState<string | null>(null);
  const [lastSavedSeq, setLastSavedSeq] = useState<number | null>(null);
  const [saveState, setSaveState] = useState<SaveState>('idle');

  useEffect(() => {
    if (auth.status !== 'authenticated') {
      return;
    }
    let stopped = false;

    function applyContent(content: string) {
      const text = textRef.current;
      docRef.current.transact(() => {
        text.delete(0, text.length);
        text.insert(0, content);
      });
      setDraft(content);
    }

    function connect() {
      setConnection((current) => (current === 'loading' ? 'loading' : 'reconnecting'));
      const socket = new WebSocket(markdownWebSocketURL(documentId));
      socketRef.current = socket;

      socket.onopen = () => {
        if (stopped) {
          socket.close();
          return;
        }
        setConnection('connected');
        setError(null);
        socket.send(JSON.stringify({ type: 'presence' }));
      };

      socket.onmessage = (event) => {
        const message = JSON.parse(event.data as string) as ServerMessage;
        if (message.type === 'init') {
          applyContent(message.content ?? '');
          setCanEdit(Boolean(message.canEdit));
          setUsers(message.users ?? []);
          setLastSavedSeq(message.updateSeq ?? null);
          return;
        }
        if (message.type === 'update') {
          applyContent(message.content ?? '');
          setLastSavedSeq(message.updateSeq ?? null);
          return;
        }
        if (message.type === 'presence') {
          setUsers(message.users ?? []);
          return;
        }
        if (message.type === 'error') {
          setError(message.error ?? '协同编辑错误');
        }
      };

      socket.onerror = () => {
        setConnection('error');
        setError('WebSocket 连接失败');
      };

      socket.onclose = () => {
        if (stopped) {
          return;
        }
        setConnection('disconnected');
        reconnectRef.current = window.setTimeout(connect, 1200);
      };
    }

    getMarkdownSnapshot(documentId)
      .then((snapshot) => {
        if (stopped) {
          return;
        }
        applyContent(snapshot.content);
        setCanEdit(snapshot.canEdit);
        setUsers(snapshot.users);
        setLastSavedSeq(snapshot.versionNo);
        connect();
      })
      .catch((caught: unknown) => {
        if (stopped) {
          return;
        }
        setConnection('error');
        setError(errorMessage(caught, '加载 Markdown 快照失败'));
      });

    return () => {
      stopped = true;
      if (reconnectRef.current !== null) {
        window.clearTimeout(reconnectRef.current);
      }
      socketRef.current?.close();
    };
  }, [auth.status, documentId]);

  function onDraftChange(content: string) {
    if (!canEdit) {
      return;
    }
    const text = textRef.current;
    docRef.current.transact(() => {
      text.delete(0, text.length);
      text.insert(0, content);
    });
    setDraft(content);
    setSaveState('idle');
    if (socketRef.current?.readyState === WebSocket.OPEN) {
      socketRef.current.send(JSON.stringify({ type: 'update', content }));
    }
  }

  async function onSave() {
    if (!canEdit) {
      return;
    }
    setSaveState('saving');
    setError(null);
    try {
      await saveMarkdownDocument(documentId, draft);
      setSaveState('saved');
    } catch (caught) {
      setSaveState('error');
      setError(errorMessage(caught, '保存 Markdown 失败'));
    }
  }

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

  const readonly = !canEdit;
  const statusText = connectionStatusText(connection, readonly);
  const visibleUsers = users.slice().sort((left, right) => {
    if (left.userId === auth.user.id) {
      return -1;
    }
    if (right.userId === auth.user.id) {
      return 1;
    }
    return left.displayName.localeCompare(right.displayName);
  });

  return (
    <main className="editor-shell markdown-shell">
      <header className="editor-topbar">
        <a className="back-link" href={`/documents/${documentId}`}>
          返回文档
        </a>
        <div className="markdown-toolbar">
          <span className={`connection-pill connection-${connection}`}>{statusText}</span>
          <span className="document-meta">{lastSavedSeq === null ? '快照' : `序号 ${lastSavedSeq}`}</span>
          <button
            className="primary-button compact-action"
            disabled={readonly || saveState === 'saving'}
            onClick={() => void onSave()}
            type="button"
          >
            {saveButtonText(saveState)}
          </button>
          <a className="secondary-button" href={documentDownloadURL(documentId)}>
            下载
          </a>
        </div>
      </header>
      {error ? <p className="form-error markdown-save-error">{error}</p> : null}
      <section className="presence-bar">
        <span className="presence-summary">当前在线 {visibleUsers.length} 人</span>
        {visibleUsers.length === 0 ? (
          <span className="empty-inline">正在获取在线成员。</span>
        ) : (
          visibleUsers.map((user) => (
            <span className="presence-user" key={`${user.userId}-${user.displayName}`}>
              {user.displayName}
              {user.userId === auth.user.id ? <small>我</small> : null}
              <small>{user.canEdit ? '可编辑' : '只读'}</small>
            </span>
          ))
        )}
      </section>
      <section className="markdown-editor-grid rich-markdown-grid">
        <RichMarkdownEditor
          content={draft}
          disabled={connection === 'loading'}
          onChange={onDraftChange}
          readOnly={readonly}
        />
        <section className="markdown-pane markdown-preview-pane">
          <span>Markdown 源码</span>
          <div className="markdown-preview markdown-source-preview">
            <pre>{draft.trim() === '' ? '暂无 Markdown 内容。' : draft}</pre>
          </div>
        </section>
      </section>
    </main>
  );
}

export function RichMarkdownEditor({
  content,
  disabled,
  onChange,
  readOnly,
}: {
  content: string;
  disabled: boolean;
  onChange: (content: string) => void;
  readOnly: boolean;
}) {
  const hostRef = useRef<HTMLDivElement | null>(null);
  const viewRef = useRef<EditorView | null>(null);
  const onChangeRef = useRef(onChange);
  const readonlyRef = useRef(readOnly || disabled);
  const [view, setView] = useState<EditorView | null>(null);

  onChangeRef.current = onChange;
  readonlyRef.current = readOnly || disabled;

  useEffect(() => {
    const host = hostRef.current;
    if (!host) {
      return;
    }
    const view = new EditorView(host, {
      state: createEditorState(content),
      editable: () => !readonlyRef.current,
      dispatchTransaction(transaction) {
        const nextState = view.state.apply(transaction);
        view.updateState(nextState);
        if (transaction.docChanged) {
          onChangeRef.current(serializeMarkdown(nextState.doc));
        }
      },
    });
    viewRef.current = view;
    setView(view);
    return () => {
      view.destroy();
      viewRef.current = null;
      setView(null);
    };
  }, []);

  useEffect(() => {
    const view = viewRef.current;
    if (!view) {
      return;
    }
    view.setProps({ editable: () => !readonlyRef.current });
  }, [readOnly, disabled]);

  useEffect(() => {
    const view = viewRef.current;
    if (!view) {
      return;
    }
    const current = serializeMarkdown(view.state.doc);
    if (current === normalizeMarkdown(content)) {
      return;
    }
    view.updateState(createEditorState(content));
  }, [content]);

  const isReadOnly = readOnly || disabled;
  const marks = markdownSchema.marks;
  const nodes = markdownSchema.nodes;

  return (
    <section className="markdown-pane rich-markdown-pane">
      <div className="rich-markdown-toolbar" aria-label="Markdown 富文本工具栏">
        <EditorButton command={undo} disabled={isReadOnly} label="撤销" view={view}>
          ↶
        </EditorButton>
        <EditorButton command={redo} disabled={isReadOnly} label="重做" view={view}>
          ↷
        </EditorButton>
        <span className="toolbar-divider" />
        <EditorButton command={setBlockType(nodes.heading, { level: 1 })} disabled={isReadOnly} label="一级标题" view={view}>
          H1
        </EditorButton>
        <EditorButton command={setBlockType(nodes.heading, { level: 2 })} disabled={isReadOnly} label="二级标题" view={view}>
          H2
        </EditorButton>
        <EditorButton command={setBlockType(nodes.paragraph)} disabled={isReadOnly} label="正文" view={view}>
          正文
        </EditorButton>
        <EditorButton command={wrapIn(nodes.blockquote)} disabled={isReadOnly} label="引用" view={view}>
          “”
        </EditorButton>
        <EditorButton command={setBlockType(nodes.code_block)} disabled={isReadOnly} label="代码块" view={view}>
          {'</>'}
        </EditorButton>
        <span className="toolbar-divider" />
        <EditorButton command={toggleMark(marks.strong)} disabled={isReadOnly} label="加粗" view={view}>
          B
        </EditorButton>
        <EditorButton command={toggleMark(marks.em)} disabled={isReadOnly} label="斜体" view={view}>
          I
        </EditorButton>
        <EditorButton command={toggleMark(marks.strike)} disabled={isReadOnly} label="删除线" view={view}>
          S
        </EditorButton>
        <EditorButton command={toggleMark(marks.code)} disabled={isReadOnly} label="行内代码" view={view}>
          code
        </EditorButton>
        <EditorButton command={setLink()} disabled={isReadOnly} label="链接" view={view}>
          链接
        </EditorButton>
        <span className="toolbar-divider" />
        <EditorButton command={insertList('bullet_list')} disabled={isReadOnly} label="无序列表" view={view}>
          •
        </EditorButton>
        <EditorButton command={insertList('ordered_list')} disabled={isReadOnly} label="有序列表" view={view}>
          1.
        </EditorButton>
        <EditorButton command={insertTable()} disabled={isReadOnly} label="表格" view={view}>
          表格
        </EditorButton>
      </div>
      <div className={`rich-markdown-editor ${isReadOnly ? 'is-readonly' : ''}`} ref={hostRef} />
    </section>
  );
}

function EditorButton({
  children,
  command,
  disabled,
  label,
  view,
}: {
  children: string;
  command: Command;
  disabled: boolean;
  label: string;
  view: EditorView | null;
}) {
  function run() {
    if (!view || disabled) {
      return;
    }
    command(view.state, view.dispatch, view);
    view.focus();
  }
  return (
    <button className="rich-toolbar-button" disabled={disabled || !view} onMouseDown={(event) => event.preventDefault()} onClick={run} title={label} type="button">
      {children}
    </button>
  );
}

function createEditorState(content: string): EditorState {
  return EditorState.create({
    doc: parseMarkdown(content),
    plugins: [
      history(),
      keymap({
        'Mod-z': undo,
        'Mod-y': redo,
        'Shift-Mod-z': redo,
        Enter: chainCommands(newlineInCode, exitCode),
      }),
      keymap(baseKeys()),
      keymap(baseKeymap),
      dropCursor(),
      tableEditing(),
    ],
  });
}

function baseKeys(): Record<string, Command> {
  return {
    'Mod-b': toggleMark(markdownSchema.marks.strong),
    'Mod-i': toggleMark(markdownSchema.marks.em),
  };
}

function setLink(): Command {
  return (state, dispatch) => {
    const href = window.prompt('链接地址');
    if (!href) {
      return false;
    }
    if (!dispatch) {
      return true;
    }
    const { from, to, empty } = state.selection;
    const mark = markdownSchema.marks.link.create({ href });
    if (empty) {
      const text = window.prompt('链接文字') || href;
      dispatch(state.tr.insertText(text, from).addMark(from, from + text.length, mark));
      return true;
    }
    dispatch(state.tr.addMark(from, to, mark));
    return true;
  };
}

function insertList(type: 'bullet_list' | 'ordered_list'): Command {
  return (state, dispatch) => {
    const { from, to } = state.selection;
    const selected = state.doc.textBetween(from, to, '\n').trim();
    const lines = (selected === '' ? ['列表项'] : selected.split('\n')).filter(Boolean);
    const items = lines.map((line) =>
      markdownSchema.nodes.list_item.create(null, markdownSchema.nodes.paragraph.create(null, parseInlineText(line))),
    );
    const node = markdownSchema.nodes[type].create(null, items);
    if (dispatch) {
      dispatch(state.tr.replaceSelectionWith(node).scrollIntoView());
    }
    return true;
  };
}

function insertTable(): Command {
  return (state, dispatch) => {
    const cell = () => markdownSchema.nodes.table_cell.create(null, markdownSchema.nodes.paragraph.create());
    const row = () => markdownSchema.nodes.table_row.create(null, [cell(), cell(), cell()]);
    const table = markdownSchema.nodes.table.create(null, [row(), row(), row()]);
    if (dispatch) {
      dispatch(state.tr.replaceSelectionWith(table).scrollIntoView());
    }
    return true;
  };
}

function parseMarkdown(content: string): ProseMirrorNode {
  const container = document.createElement('div');
  container.innerHTML = markdownToHTML(content);
  const parsed = ProseMirrorDOMParser.fromSchema(markdownSchema).parse(container);
  if (parsed.childCount === 0) {
    return markdownSchema.nodes.doc.create(null, markdownSchema.nodes.paragraph.create());
  }
  return parsed;
}

function markdownToHTML(content: string): string {
  const blocks = content.replace(/\r\n?/g, '\n').split(/\n{2,}/);
  if (blocks.length === 0 || blocks.every((block) => block.trim() === '')) {
    return '<p></p>';
  }
  return blocks
    .map((block) => {
      const trimmed = block.trim();
      if (trimmed.startsWith('```')) {
        return `<pre><code>${escapeHTML(trimmed.replace(/^```\w*\n?/, '').replace(/```$/, ''))}</code></pre>`;
      }
      const heading = trimmed.match(/^(#{1,3})\s+(.+)$/);
      if (heading) {
        const level = heading[1].length;
        return `<h${level}>${inlineMarkdownToHTML(heading[2])}</h${level}>`;
      }
      if (trimmed.split('\n').every((line) => /^[-*]\s+/.test(line))) {
        return `<ul>${trimmed
          .split('\n')
          .map((line) => `<li>${inlineMarkdownToHTML(line.replace(/^[-*]\s+/, ''))}</li>`)
          .join('')}</ul>`;
      }
      if (trimmed.split('\n').every((line) => /^\d+\.\s+/.test(line))) {
        return `<ol>${trimmed
          .split('\n')
          .map((line) => `<li>${inlineMarkdownToHTML(line.replace(/^\d+\.\s+/, ''))}</li>`)
          .join('')}</ol>`;
      }
      if (isMarkdownTable(trimmed)) {
        return tableMarkdownToHTML(trimmed);
      }
      if (trimmed.startsWith('> ')) {
        return `<blockquote>${markdownToHTML(trimmed.replace(/^>\s?/gm, ''))}</blockquote>`;
      }
      return `<p>${trimmed.split('\n').map(inlineMarkdownToHTML).join('<br>')}</p>`;
    })
    .join('');
}

function inlineMarkdownToHTML(value: string): string {
  return escapeHTML(value)
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>')
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
    .replace(/\*([^*]+)\*/g, '<em>$1</em>')
    .replace(/~~([^~]+)~~/g, '<s>$1</s>');
}

function tableMarkdownToHTML(value: string): string {
  const rows = value
    .split('\n')
    .filter((line) => !/^\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|?$/.test(line))
    .map((line) => line.replace(/^\||\|$/g, '').split('|').map((cell) => cell.trim()));
  return `<table><tbody>${rows
    .map((row) => `<tr>${row.map((cell) => `<td>${inlineMarkdownToHTML(cell)}</td>`).join('')}</tr>`)
    .join('')}</tbody></table>`;
}

function isMarkdownTable(value: string): boolean {
  const lines = value.split('\n');
  return lines.length >= 2 && lines[0].includes('|') && /^\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|?$/.test(lines[1]);
}

function parseInlineText(value: string): ProseMirrorNode[] {
  const text = value.trim();
  return text === '' ? [] : [markdownSchema.text(text)];
}

function serializeMarkdown(doc: ProseMirrorNode): string {
  const parts: string[] = [];
  doc.forEach((node) => {
    parts.push(serializeBlock(node));
  });
  return normalizeMarkdown(parts.join('\n\n'));
}

function serializeBlock(node: ProseMirrorNode): string {
  switch (node.type.name) {
    case 'heading':
      return `${'#'.repeat(node.attrs.level)} ${serializeInline(node.content)}`;
    case 'blockquote':
      return serializeMarkdown(node).split('\n').map((line) => `> ${line}`).join('\n');
    case 'code_block':
      return `\`\`\`\n${node.textContent}\n\`\`\``;
    case 'bullet_list':
      return serializeList(node, '-');
    case 'ordered_list':
      return serializeList(node, '1.');
    case 'table':
      return serializeTable(node);
    default:
      return serializeInline(node.content);
  }
}

function serializeList(node: ProseMirrorNode, marker: string): string {
  const lines: string[] = [];
  node.forEach((item) => {
    const text = item.textContent.trim() || '列表项';
    lines.push(`${marker} ${text}`);
  });
  return lines.join('\n');
}

function serializeTable(node: ProseMirrorNode): string {
  const rows: string[][] = [];
  node.forEach((row) => {
    const cells: string[] = [];
    row.forEach((cell) => {
      cells.push(cell.textContent.trim());
    });
    rows.push(cells);
  });
  if (rows.length === 0) {
    return '';
  }
  const header = rows[0];
  const separator = header.map(() => '---');
  return [header, separator, ...rows.slice(1)]
    .map((row) => `| ${row.map((cell) => cell || ' ').join(' | ')} |`)
    .join('\n');
}

function serializeInline(fragment: Fragment): string {
  const parts: string[] = [];
  fragment.forEach((node) => {
    if (node.type.name === 'hard_break') {
      parts.push('\n');
      return;
    }
    if (node.isText) {
      parts.push(serializeText(node.text ?? '', node.marks));
      return;
    }
    parts.push(node.textContent);
  });
  return parts.join('');
}

function serializeText(text: string, marks: readonly Mark[]): string {
  return marks.reduce((value, mark) => {
    switch (mark.type.name) {
      case 'strong':
        return `**${value}**`;
      case 'em':
        return `*${value}*`;
      case 'strike':
        return `~~${value}~~`;
      case 'code':
        return `\`${value}\``;
      case 'link':
        return `[${value}](${mark.attrs.href})`;
      default:
        return value;
    }
  }, text);
}

function normalizeMarkdown(content: string): string {
  return content.replace(/\r\n?/g, '\n').replace(/[ \t]+\n/g, '\n').trimEnd();
}

function escapeHTML(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function connectionStatusText(connection: ConnectionState, readonly: boolean): string {
  if (connection === 'loading') {
    return '加载中';
  }
  if (connection === 'connected') {
    return readonly ? '已连接，只读' : '已连接';
  }
  if (connection === 'reconnecting') {
    return '重连中';
  }
  if (connection === 'error') {
    return '连接错误';
  }
  return '已断开';
}

function saveButtonText(state: SaveState): string {
  if (state === 'saving') {
    return '保存中';
  }
  if (state === 'saved') {
    return '已保存';
  }
  if (state === 'error') {
    return '重试保存';
  }
  return '保存';
}
