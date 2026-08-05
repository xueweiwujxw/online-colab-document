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

  test('xlsx 实际渲染中文表格界面并保持 viewer 只读', async ({ browser, page }) => {
    await register(page, account('sheet-owner'));
    const documentId = await upload(page, await xlsxFile());
    const viewerContext = await browser.newContext();
    const viewerPage = await viewerContext.newPage();
    const viewer = account('sheet-viewer');
    await register(viewerPage, viewer);

    await page.goto(`/documents/${documentId}/permissions`);
    await page.getByLabel('搜索用户').fill(viewer.email);
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await page.getByRole('button', { name: new RegExp(viewer.email) }).click();
    await page.getByLabel('权限').selectOption('viewer');
    await page.getByRole('button', { name: '授权', exact: true }).click();
    await expect(page.getByText('已有：只读')).toBeVisible();

    await page.goto(`/documents/${documentId}/edit`);
    await waitForSheet(page);

    await viewerPage.goto(`/documents/${documentId}/edit`);
    await waitForSheet(viewerPage);
    await expect(viewerPage.getByText('只读')).toBeVisible();
    await viewerPage.setViewportSize({ width: 375, height: 812 });
    expect(await hasHorizontalOverflow(viewerPage)).toBe(false);

    await viewerContext.close();
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
  const iframe = page.locator('.office-sheet-iframe');
  await expect(iframe).toBeVisible({ timeout: 45_000 });
  const frame = page.frameLocator('.office-sheet-iframe');
  await expect(frame.getByText('文件', { exact: true })).toBeVisible({ timeout: 45_000 });
  await expect(frame.getByText('编辑', { exact: true })).toBeVisible();
  await expect
    .poll(() => frame.locator('canvas').evaluateAll((canvases) => canvases.some((canvas) => {
      const rect = canvas.getBoundingClientRect();
      return rect.width > 100 && rect.height > 100;
    })))
    .toBe(true);
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
