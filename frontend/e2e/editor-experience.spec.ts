import { expect, test, type Page } from '@playwright/test';
import ExcelJS from 'exceljs';

type Account = { displayName: string; email: string; password: string };

test.describe('编辑器体验回归', () => {
  test('Markdown 富文本可编辑、插入表格和链接，并在三种视口保持可用', async ({ page }) => {
    await register(page, account('markdown-owner'));
    const documentId = await upload(page, markdownFile());
    await page.goto(`/documents/${documentId}/markdown`);

    await expect(page.getByRole('toolbar', { name: 'Markdown 富文本工具栏' })).toBeVisible();
    const editor = page.locator('.ProseMirror');
    await expect(editor).toBeEditable();
    await editor.click();
    await page.keyboard.press('Control+End');
    await page.keyboard.type('可编辑内容');
    await page.getByRole('button', { name: '加粗' }).click();
    await page.getByRole('button', { name: '插入表格' }).click();
    const tableForm = page.getByRole('form', { name: '插入表格设置' });
    await tableForm.getByLabel('行数').fill('2');
    await tableForm.getByLabel('列数').fill('4');
    await tableForm.getByRole('button', { name: '插入', exact: true }).click();
    await expect(editor.locator('table')).toBeVisible();
    await expect(editor.locator('table tr')).toHaveCount(2);
    await expect(editor.locator('table tr').first().locator('td')).toHaveCount(4);

    await page.getByRole('button', { name: '插入链接' }).click();
    const linkForm = page.getByRole('form', { name: '插入链接' });
    await linkForm.getByLabel('链接地址').fill('https://example.test/docs');
    await linkForm.getByLabel('显示文字').fill('产品文档');
    await linkForm.getByRole('button', { name: '插入', exact: true }).click();
    await expect(editor.getByRole('link', { name: '产品文档' })).toBeVisible();

    await page.getByRole('button', { name: '保存', exact: true }).click();
    await expect(page.getByRole('button', { name: '已保存', exact: true })).toBeVisible();
    await expect(page.getByRole('link', { name: '下载' })).toHaveAttribute('href', /\/download$/);

    for (const viewport of [
      { width: 1440, height: 960 },
      { width: 768, height: 1024 },
      { width: 375, height: 812 },
    ]) {
      await page.setViewportSize(viewport);
      await expect(page.locator('.rich-markdown-toolbar')).toBeVisible();
      expect(await hasHorizontalOverflow(page)).toBe(false);
    }
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
