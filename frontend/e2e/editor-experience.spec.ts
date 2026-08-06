import { expect, test, type Page } from '@playwright/test';
import ExcelJS from 'exceljs';

type Account = { displayName: string; email: string; password: string };

test.describe('编辑器体验回归', () => {
  test('Markdown 原生协同：同步、只读、重连与保存版本', async ({ browser, page }) => {
    test.setTimeout(120_000);
    const owner = account('markdown-owner');
    await register(page, owner);
    const documentId = await upload(page, markdownFile());
    const viewer = account('markdown-viewer');
    const viewerContext = await browser.newContext();
    const viewerPage = await viewerContext.newPage();
    await register(viewerPage, viewer);
    await page.goto(`/documents/${documentId}/permissions`);
    await grant(page, viewer.email, 'viewer');

    await Promise.all([
      page.goto(`/documents/${documentId}/markdown`),
      viewerPage.goto(`/documents/${documentId}/markdown`),
    ]);
    const ownerEditor = page.locator('[contenteditable="true"]').first();
    await expect(ownerEditor).toBeVisible({ timeout: 30_000 });
    await expect(viewerPage.locator('[contenteditable="true"]')).toHaveCount(0);
    const marker = `markdown-sync-${Date.now()}`;
    await ownerEditor.focus();
    await page.keyboard.press('Control+End');
    await page.keyboard.type(` ${marker}`);
    await expect(viewerPage.getByText(marker, { exact: false })).toBeVisible({ timeout: 20_000 });
    await expect(viewerPage.locator('.ProseMirror-yjs-cursor')).toBeVisible({ timeout: 20_000 });
    await expect(viewerPage.getByText(owner.displayName, { exact: true })).toBeVisible({ timeout: 20_000 });

    await page.reload();
    await expect(page.locator('[contenteditable="true"]').first()).toBeVisible({ timeout: 30_000 });
    await expect(page.getByText(marker, { exact: false })).toBeVisible({ timeout: 20_000 });
    await expect(wopiWriteStatus(page)).resolves.toBe(200);
    await expect(wopiWriteStatus(viewerPage)).resolves.toBe(403);
    await page.goto(`/documents/${documentId}/versions`);
    await expect(page.getByText('版本 2', { exact: true })).toBeVisible({ timeout: 20_000 });
    await viewerContext.close();
  });

  test('docx 原生协同：名称、远程光标、只读、重连与保存版本', async ({ browser, page }) => {
    test.setTimeout(120_000);
    const owner = account('docx-owner');
    await register(page, owner);
    const documentId = await upload(page, docxFile());
    const viewer = account('docx-viewer');
    const viewerContext = await browser.newContext();
    const viewerPage = await viewerContext.newPage();
    await register(viewerPage, viewer);
    await page.goto(`/documents/${documentId}/permissions`);
    await grant(page, viewer.email, 'viewer');

    await Promise.all([
      page.goto(`/documents/${documentId}/edit`),
      viewerPage.goto(`/documents/${documentId}/edit`),
    ]);
    const ownerEditor = page.locator('[contenteditable="true"]').first();
    await expect(ownerEditor).toBeVisible({ timeout: 30_000 });
    await expect(viewerPage.locator('[contenteditable="true"]')).toHaveCount(0);
    await expect(page.getByText(viewer.displayName, { exact: true })).toBeVisible({ timeout: 20_000 });

    const marker = `docx-sync-${Date.now()}`;
    await ownerEditor.focus();
    await page.keyboard.press('Control+End');
    await page.keyboard.type(` ${marker}`);
    await expect(viewerPage.getByText(marker, { exact: false })).toBeVisible({ timeout: 20_000 });
    await expect(viewerPage.locator('.ProseMirror-yjs-cursor')).toBeVisible({ timeout: 20_000 });
    await expect(viewerPage.getByText(owner.displayName, { exact: true })).toBeVisible({ timeout: 20_000 });

    await page.reload();
    const reconnectedEditor = page.locator('[contenteditable="true"]').first();
    await expect(reconnectedEditor).toBeVisible({ timeout: 30_000 });
    await expect(page.getByText(marker, { exact: false })).toBeVisible({ timeout: 20_000 });

    await expect(wopiWriteStatus(page)).resolves.toBe(200);
    await expect(wopiWriteStatus(viewerPage)).resolves.toBe(403);
    await page.goto(`/documents/${documentId}/versions`);
    await expect(page.getByText('版本 2', { exact: true })).toBeVisible({ timeout: 20_000 });
    await viewerContext.close();
  });

  test('xlsx 原生协同：同步、重连、只读与保存版本', async ({ browser, page }) => {
    test.setTimeout(120_000);
    const owner = account('sheet-owner');
    await register(page, owner);
    const documentId = await upload(page, await xlsxFile());
    const editor = account('sheet-editor');
    const editorContext = await browser.newContext();
    const editorPage = await editorContext.newPage();
    await register(editorPage, editor);
    const viewerContext = await browser.newContext();
    const viewerPage = await viewerContext.newPage();
    const viewer = account('sheet-viewer');
    await register(viewerPage, viewer);

    await page.goto(`/documents/${documentId}/permissions`);
    await grant(page, editor.email, 'editor');
    await page.goto(`/documents/${documentId}/permissions`);
    await grant(page, viewer.email, 'viewer');

    await Promise.all([
      page.goto(`/documents/${documentId}/edit`),
      editorPage.goto(`/documents/${documentId}/edit`),
    ]);
    await Promise.all([waitForSheet(page), waitForSheet(editorPage)]);
    await expect(page.getByText(owner.displayName, { exact: true })).toBeVisible();
    await expect(editorPage.getByText(editor.displayName, { exact: true })).toBeVisible();

    const marker = `sync-${Date.now()}`;
    await selectFirstVisibleSheetCell(page);
    await page.getByTestId('formula-input').fill(marker);
    await page.getByTestId('formula-input').press('Enter');
    await selectFirstVisibleSheetCell(editorPage);
    await expect(editorPage.getByTestId('formula-input')).toHaveValue(marker, { timeout: 20_000 });
    await expect(page.getByTestId('presence-cursor')).toBeVisible({ timeout: 20_000 });
    await expect(page.getByText(editor.displayName, { exact: true })).toBeVisible();

    await editorPage.reload();
    await waitForSheet(editorPage);
    await selectFirstVisibleSheetCell(editorPage);
    await expect(editorPage.getByTestId('formula-input')).toHaveValue(marker, { timeout: 20_000 });

    await viewerPage.goto(`/documents/${documentId}/edit`);
    await waitForSheet(viewerPage);
    await expect(viewerPage.getByTestId('view-only-banner')).toBeVisible();
    await selectFirstVisibleSheetCell(viewerPage);
    await expect(viewerPage.getByTestId('formula-input')).toHaveValue(marker, { timeout: 20_000 });
    await expect(viewerPage.getByTestId('formula-input')).not.toBeEditable();

    const saveResponse = page.waitForResponse((response) => (
      response.request().method() === 'POST'
      && new URL(response.url()).pathname === `/wopi/files/${documentId}/contents`
      && response.ok()
    ), { timeout: 60_000 });
    await page.keyboard.press('Control+S');
    await saveResponse;
    await page.goto(`/documents/${documentId}/versions`);
    await expect(page.locator('.version-row').first()).toBeVisible();
    await expect(page.getByText('版本 2', { exact: true })).toBeVisible();

    await editorContext.close();
    await viewerContext.close();
  });

  test('xlsx viewer 无法在权限页外获得编辑能力', async ({ browser, page }) => {
    await register(page, account('sheet-owner-readonly'));
    const documentId = await upload(page, await xlsxFile());
    const viewerContext = await browser.newContext();
    const viewerPage = await viewerContext.newPage();
    const viewer = account('sheet-viewer-readonly');
    await register(viewerPage, viewer);

    await page.goto(`/documents/${documentId}/permissions`);
    await page.getByLabel('搜索用户').fill(viewer.email);
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await page.getByRole('button', { name: new RegExp(viewer.email) }).click();
    await page.getByLabel('权限').selectOption('viewer');
    await page.getByRole('button', { name: '授权', exact: true }).click();
    await expect(page.getByText('已有：只读')).toBeVisible();

    await viewerPage.goto(`/documents/${documentId}/edit`);
    await waitForSheet(viewerPage);
    await expect(viewerPage.getByTestId('view-only-banner')).toBeVisible();
    await expect(viewerPage.getByTestId('formula-input')).not.toBeEditable();
    await viewerPage.setViewportSize({ width: 375, height: 812 });
    expect(await hasHorizontalOverflow(viewerPage)).toBe(false);

    await viewerContext.close();
  });

  test('Markdown 协同显示远程光标与用户名称', async ({ browser, page }) => {
    const owner = account('cursor-owner');
    await register(page, owner);
    const documentId = await upload(page, markdownFile());
    const editor = account('cursor-editor');
    const editorContext = await browser.newContext();
    const editorPage = await editorContext.newPage();
    await register(editorPage, editor);

    await page.goto(`/documents/${documentId}/permissions`);
    await page.getByLabel('搜索用户').fill(editor.email);
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await page.getByRole('button', { name: new RegExp(editor.email) }).click();
    await page.getByLabel('权限').selectOption('editor');
    await page.getByRole('button', { name: '授权', exact: true }).click();

    await page.goto(`/documents/${documentId}/markdown`);
    await editorPage.goto(`/documents/${documentId}/markdown`);
    const remoteEditor = editorPage.locator('.ProseMirror');
    await expect(editorPage.getByText('已连接', { exact: true })).toBeVisible({ timeout: 10_000 });
    await expect(remoteEditor).toHaveAttribute('contenteditable', 'true');
    await remoteEditor.click();
    await editorPage.keyboard.press('Control+End');
    await editorPage.keyboard.press('ArrowLeft');

    const remoteCursor = page.locator('.remote-cursor-label', { hasText: editor.displayName });
    await expect(remoteCursor).toBeVisible({ timeout: 10_000 });
    await expect(remoteCursor).toHaveCSS('pointer-events', 'none');
    await editorContext.close();
  });
});

async function register(page: Page, user: Account): Promise<void> {
  await page.goto('/register');
  await page.getByLabel('显示名').fill(user.displayName);
  await page.getByLabel('邮箱').fill(user.email);
  await page.getByLabel('密码').fill(user.password);
  await page.getByRole('button', { name: '注册并登录' }).click();
  await page.waitForFunction(() => window.location.pathname === '/documents');
}

async function upload(page: Page, file: { name: string; mimeType: string; buffer: Buffer }): Promise<string> {
  await page.locator('input[type="file"]').setInputFiles(file);
  const row = page.locator('.document-row').first();
  await expect(row).toBeVisible();
  const href = await row.locator('.document-title').getAttribute('href');
  if (!href) throw new Error('上传后未找到文档详情地址');
  return href.split('/').at(-1) ?? '';
}

async function waitForSheet(page: Page): Promise<void> {
  const namePrompt = page.getByTestId('name-prompt-backdrop');
  if (await namePrompt.isVisible().catch(() => false)) {
    await namePrompt.getByRole('button').last().click();
  }
  await expect(page.getByText('File', { exact: true })).toBeVisible({ timeout: 45_000 });
  await expect
    .poll(() => page.locator('canvas').evaluateAll((canvases) => canvases.some((canvas) => {
      const rect = canvas.getBoundingClientRect();
      return rect.width > 100 && rect.height > 100;
    })))
    .toBe(true);
}

async function selectFirstVisibleSheetCell(page: Page): Promise<void> {
  const canvas = page.locator('canvas');
  const index = await canvas.evaluateAll((items) => items.findIndex((item) => {
    const rect = item.getBoundingClientRect();
    return rect.width > 100 && rect.height > 100;
  }));
  if (index < 0) throw new Error('未找到可操作的表格画布');
  const target = canvas.nth(index);
  const box = await target.boundingBox();
  if (!box) throw new Error('表格画布不可见');
  await target.click({ position: { x: Math.min(180, box.width / 2), y: Math.min(180, box.height / 2) } });
}

async function grant(page: Page, email: string, role: 'editor' | 'viewer'): Promise<void> {
  await page.getByLabel('搜索用户').fill(email);
  await page.getByRole('button', { name: '搜索', exact: true }).click();
  await page.getByRole('button', { name: new RegExp(email) }).click();
  await page.getByLabel('权限').selectOption(role);
  await page.getByRole('button', { name: '授权', exact: true }).click();
}

async function hasHorizontalOverflow(page: Page): Promise<boolean> {
  return page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth);
}

function account(prefix: string): Account {
  const suffix = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  return { displayName: `${prefix}-${suffix}`, email: `${prefix}-${suffix}@example.test`, password: 'BrowserCheck123!' };
}

function markdownFile() {
  return { name: 'editor-experience.md', mimeType: 'text/markdown', buffer: Buffer.from('# 编辑器体验\n\n初始内容。\n') };
}

function docxFile() {
  return {
    name: 'editor-experience.docx',
    mimeType: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    buffer: Buffer.from('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', 'base64'),
  };
}

async function wopiWriteStatus(page: Page): Promise<number> {
  return page.evaluate(async () => {
    const token = new URLSearchParams(location.search).get('access_token');
    const payload = token?.split('.')[1];
    if (!token || !payload) return -1;
    const normalized = payload.replace(/-/g, '+').replace(/_/g, '/');
    const claims = JSON.parse(atob(normalized + '='.repeat((4 - normalized.length % 4) % 4))) as { file_id?: string };
    if (!claims.file_id) return -1;
    const path = `/wopi/files/${encodeURIComponent(claims.file_id)}/contents?access_token=${encodeURIComponent(token)}`;
    const source = await fetch(path);
    if (!source.ok) return source.status;
    const response = await fetch(path, { method: 'POST', body: await source.arrayBuffer() });
    return response.status;
  });
}

async function xlsxFile() {
  const workbook = new ExcelJS.Workbook();
  const sheet = workbook.addWorksheet('预算');
  sheet.columns = [
    { header: '项目', key: 'item', width: 20 },
    { header: '金额', key: 'amount', width: 14 },
  ];
  sheet.addRow({ item: '设计', amount: 1200 });
  sheet.addRow({ item: '研发', amount: 8600 });
  sheet.getRow(1).font = { bold: true };
  sheet.getColumn('amount').numFmt = '¥#,##0.00';
  return {
    name: 'editor-experience.xlsx',
    mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    buffer: Buffer.from(await workbook.xlsx.writeBuffer()),
  };
}
