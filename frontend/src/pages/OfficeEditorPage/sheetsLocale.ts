import { LocaleType, Tools } from '@univerjs/core';

import UniverDvZhCN from '@univerjs/data-validation/locale/zh-CN';
import UniverDocsUIZhCN from '@univerjs/docs-ui/locale/zh-CN';
import UniverDrawingUIZhCN from '@univerjs/drawing-ui/locale/zh-CN';
import UniverFindReplaceZhCN from '@univerjs/find-replace/locale/zh-CN';
import UniverSheetsCfUIZhCN from '@univerjs/sheets-conditional-formatting-ui/locale/zh-CN';
import UniverSheetsDvZhCN from '@univerjs/sheets-data-validation/locale/zh-CN';
import UniverSheetsDvUIZhCN from '@univerjs/sheets-data-validation-ui/locale/zh-CN';
import UniverSheetsDrawingUIZhCN from '@univerjs/sheets-drawing-ui/locale/zh-CN';
import UniverSheetsFilterZhCN from '@univerjs/sheets-filter/locale/zh-CN';
import UniverSheetsFilterUIZhCN from '@univerjs/sheets-filter-ui/locale/zh-CN';
import UniverSheetsFormulaUIZhCN from '@univerjs/sheets-formula-ui/locale/zh-CN';
import UniverSheetsHyperLinkZhCN from '@univerjs/sheets-hyper-link/locale/zh-CN';
import UniverSheetsHyperLinkUIZhCN from '@univerjs/sheets-hyper-link-ui/locale/zh-CN';
import UniverSheetsNoteUIZhCN from '@univerjs/sheets-note-ui/locale/zh-CN';
import UniverSheetsNumfmtUIZhCN from '@univerjs/sheets-numfmt-ui/locale/zh-CN';
import UniverSheetsSortUIZhCN from '@univerjs/sheets-sort-ui/locale/zh-CN';
import UniverSheetsTableZhCN from '@univerjs/sheets-table/locale/zh-CN';
import UniverSheetsTableUIZhCN from '@univerjs/sheets-table-ui/locale/zh-CN';
import UniverSheetsThreadCommentUIZhCN from '@univerjs/sheets-thread-comment-ui/locale/zh-CN';
import UniverSheetsZhCN from '@univerjs/sheets/locale/zh-CN';
import UniverSheetsUIZhCN from '@univerjs/sheets-ui/locale/zh-CN';
import UniverThreadCommentUIZhCN from '@univerjs/thread-comment-ui/locale/zh-CN';
import UniverUIZhCN from '@univerjs/ui/locale/zh-CN';

const zhCN = Tools.deepMerge(
  {},
  UniverUIZhCN,
  UniverDocsUIZhCN,
  UniverSheetsZhCN,
  UniverSheetsUIZhCN,
  UniverSheetsFormulaUIZhCN,
  UniverSheetsNumfmtUIZhCN,
  UniverSheetsTableUIZhCN,
  UniverSheetsSortUIZhCN,
  UniverSheetsFilterUIZhCN,
  UniverSheetsCfUIZhCN,
  UniverDvZhCN,
  UniverSheetsDvZhCN,
  UniverSheetsDvUIZhCN,
  UniverSheetsFilterZhCN,
  UniverSheetsHyperLinkZhCN,
  UniverSheetsTableZhCN,
  UniverDrawingUIZhCN,
  UniverSheetsDrawingUIZhCN,
  UniverSheetsHyperLinkUIZhCN,
  UniverSheetsNoteUIZhCN,
  UniverThreadCommentUIZhCN,
  UniverSheetsThreadCommentUIZhCN,
  UniverFindReplaceZhCN,
);

export const SHEETS_LOCALE = LocaleType.ZH_CN;

export const SHEETS_LOCALES = {
  [LocaleType.ZH_CN]: zhCN,
};
