(() => {
  'use strict';

  // Casual Docs and Casual Sheets currently ship their hosted chrome in
  // English. This gateway-side layer keeps the vendor services unmodified
  // while translating their UI as it is rendered. It deliberately excludes
  // editable document content, so user-authored text is never changed.
  const translations = new Map([
    ['File', '文件'], ['Edit', '编辑'], ['View', '视图'], ['Insert', '插入'], ['Format', '格式'], ['Data', '数据'], ['Help', '帮助'],
    ['Open', '打开'], ['Save', '保存'], ['Save as', '另存为'], ['Print', '打印'], ['Page setup', '页面设置'], ['Properties', '属性'], ['Close', '关闭'],
    ['Undo', '撤销'], ['Redo', '重做'], ['Cut', '剪切'], ['Copy', '复制'], ['Paste', '粘贴'], ['Paste special', '选择性粘贴'], ['Select all', '全选'],
    ['Find', '查找'], ['Replace', '替换'], ['Find and replace', '查找和替换'], ['Zoom', '缩放'], ['Zoom in', '放大'], ['Zoom out', '缩小'],
    ['Bold', '加粗'], ['Italic', '斜体'], ['Underline', '下划线'], ['Strikethrough', '删除线'], ['Font', '字体'], ['Font Size', '字号'],
    ['Text Color', '文字颜色'], ['Fill Color', '填充颜色'], ['Borders', '边框'], ['Merge Cells', '合并单元格'], ['Wrap Text', '自动换行'],
    ['Align Left', '左对齐'], ['Align Center', '水平居中'], ['Align Right', '右对齐'], ['General', '常规'], ['Currency', '货币'], ['Percentage', '百分比'],
    ['Sheet', '工作表'], ['Add Sheet', '新建工作表'], ['Delete Sheet', '删除工作表'], ['Rename Sheet', '重命名工作表'],
    ['Sort', '排序'], ['Filter', '筛选'], ['Comment', '批注'], ['Comments', '批注'], ['Reply', '回复'], ['Share', '共享'],
    ['Table', '表格'], ['Image', '图片'], ['Link', '链接'], ['Page Break', '分页符'], ['Table of Contents', '目录'], ['Symbol', '符号'],
    ['Loading…', '正在加载…'], ['Loading document…', '正在加载文档…'], ['Loading workbook…', '正在加载表格…'],
    ['Failed to load workbook:', '加载表格失败：'], ['Couldn’t load the document', '无法加载文档'], ['Search', '搜索'], ['Cancel', '取消'],
    ['Apply', '应用'], ['Delete', '删除'], ['Update', '更新'], ['Retry', '重试'], ['Insert', '插入'], ['OK', '确定'], ['Yes', '是'], ['No', '否'],
    ['Name', '名称'], ['Description', '说明'], ['Type', '类型'], ['Value', '值'], ['Formula', '公式'], ['Function', '函数'], ['Average', '平均值'],
    ['Count', '计数'], ['Min', '最小值'], ['Max', '最大值'], ['Sum', '求和'], ['Status', '状态'], ['Settings', '设置'],
    ['Read only', '只读'], ['View only', '仅查看'], ['Version history', '版本历史'], ['Download', '下载'], ['Upload', '上传'],
    ['Insert function', '插入函数'], ['Formula input', '公式输入'], ['Cancel edit', '取消编辑'], ['Enter', '确认'],
    ['Search menus', '搜索菜单'], ['Keyboard shortcuts', '键盘快捷键'], ['Word count', '字数统计'], ['Print preview', '打印预览'],
    ['Tools', '工具'], ['Unsaved changes', '未保存的更改'], ['Live', '在线'], ['other editor', '位其他编辑者'],
    ['Normal text', '正文'], ['Page', '第'], ['of', '页，共'], ['words', '个单词'], ['characters', '个字符'],
  ]);

  const attributes = ['aria-label', 'title', 'placeholder'];
  const editableSelector = '[contenteditable="true"], .ProseMirror, [role="textbox"]';
  const utf8Decoder = new TextDecoder();

  function isEditorContent(node) {
    const element = node.nodeType === Node.ELEMENT_NODE ? node : node.parentElement;
    // Cursor labels are editor overlays rather than user-authored document
    // content. Allow UTF-8 recovery for them while preserving body text.
    if (element?.closest('.ProseMirror-yjs-cursor, .yjs-cursor, [class*="yjs-cursor"]')) return false;
    return Boolean(element?.closest(editableSelector));
  }

  function recoverUtf8(value) {
    // Casual Docs older hosted builds parse JWT JSON directly from `atob()`,
    // which exposes UTF-8 bytes as Latin-1. Only transform likely mojibake and
    // keep the original if the bytes do not decode cleanly.
    if (!/[\u00c2-\u00f4]/.test(value)) return value;
    try {
      const decoded = utf8Decoder.decode(Uint8Array.from(value, (char) => char.charCodeAt(0)));
      return decoded.includes('\ufffd') ? value : decoded;
    } catch {
      return value;
    }
  }

  function translated(value) {
    const normalized = recoverUtf8(value);
    const trimmed = normalized.trim();
    const target = translations.get(trimmed);
    return target ? normalized.replace(trimmed, target) : normalized;
  }

  function translateElement(element) {
    if (!(element instanceof Element) || isEditorContent(element)) return;
    for (const attribute of attributes) {
      const value = element.getAttribute(attribute);
      if (value) element.setAttribute(attribute, translated(value));
    }
  }

  function translate(root) {
    if (root.nodeType === Node.TEXT_NODE) {
      if (!isEditorContent(root)) root.nodeValue = translated(root.nodeValue ?? '');
      return;
    }
    if (!(root instanceof Element || root instanceof Document)) return;
    if (root instanceof Element) translateElement(root);
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    for (let node = walker.nextNode(); node; node = walker.nextNode()) {
      if (!isEditorContent(node)) node.nodeValue = translated(node.nodeValue ?? '');
    }
    root.querySelectorAll?.('[aria-label], [title], [placeholder]').forEach(translateElement);
  }

  document.documentElement.lang = 'zh-CN';
  const observer = new MutationObserver((records) => {
    for (const record of records) {
      for (const node of record.addedNodes) translate(node);
    }
  });
  observer.observe(document.documentElement, { childList: true, subtree: true });
  translate(document.documentElement);
})();
