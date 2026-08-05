(() => {
  const translations = new Map([
    ['File', '文件'], ['Edit', '编辑'], ['View', '视图'], ['Insert', '插入'], ['Format', '格式'], ['Data', '数据'], ['Help', '帮助'],
    ['General', '常规'], ['Undo', '撤销'], ['Redo', '重做'], ['Cut', '剪切'], ['Copy', '复制'], ['Paste', '粘贴'],
    ['Bold', '加粗'], ['Italic', '斜体'], ['Underline', '下划线'], ['Strikethrough', '删除线'], ['Font', '字体'], ['Font Size', '字号'],
    ['Text Color', '文字颜色'], ['Fill Color', '填充颜色'], ['Borders', '边框'], ['Merge Cells', '合并单元格'], ['Wrap Text', '自动换行'],
    ['Align Left', '左对齐'], ['Align Center', '水平居中'], ['Align Right', '右对齐'], ['Currency', '货币'], ['Percentage', '百分比'],
    ['Sheet', '工作表'], ['Add Sheet', '新建工作表'], ['Delete Sheet', '删除工作表'], ['Rename Sheet', '重命名工作表'],
    ['Find', '查找'], ['Replace', '替换'], ['Sort', '排序'], ['Filter', '筛选'], ['Loading workbook…', '正在加载表格…'], ['Failed to load workbook:', '加载表格失败：'],
  ]);

  function translate(root) {
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    for (let node = walker.nextNode(); node; node = walker.nextNode()) {
      const value = node.nodeValue?.trim();
      const translated = value && translations.get(value);
      if (translated) node.nodeValue = node.nodeValue.replace(value, translated);
    }
    root.querySelectorAll?.('[aria-label], [title], [placeholder]').forEach((element) => {
      for (const attribute of ['aria-label', 'title', 'placeholder']) {
        const value = element.getAttribute(attribute);
        const translated = value && translations.get(value);
        if (translated) element.setAttribute(attribute, translated);
      }
    });
  }

  const observer = new MutationObserver((records) => {
    for (const record of records) for (const node of record.addedNodes) if (node.nodeType === Node.ELEMENT_NODE) translate(node);
  });
  observer.observe(document.documentElement, { childList: true, subtree: true });
  translate(document.documentElement);
})();
