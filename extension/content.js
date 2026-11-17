(() => {
  function addButtons() {
    // WT Live inserts post cards and the post lightbox dynamically.
    for (const download of document.querySelectorAll('a.downloadButton, #clb a.downloads[href], .feed_item a.downloads[href]')) {
      if (download.dataset.wtSkinInstaller === '1') continue;
      const container = download.closest('[lang_group]');
      const match = location.pathname.match(/^\/post\/(\d+)(?:\/|$)/);
      const id = container?.getAttribute('lang_group') || match?.[1];
      if (!/^\d+$/.test(id || '')) continue;
      download.dataset.wtSkinInstaller = '1';
      const button = document.createElement('button');
      button.className = 'wt-skin-install';
      button.type = 'button';
      button.textContent = 'Установить скин';
      button.addEventListener('click', event => {
        event.preventDefault();
        event.stopPropagation();
        chrome.runtime.sendMessage({ type: 'OPEN_INSTALLER', post: id }, () => {
          if (chrome.runtime.lastError) {
            button.textContent = 'Не удалось открыть — обновите расширение';
          }
        });
      });
      download.insertAdjacentElement('afterend', button);
    }
  }

  addButtons();
  let scheduled = false;
  new MutationObserver(() => {
    if (scheduled) return;
    scheduled = true;
    requestAnimationFrame(() => {
      scheduled = false;
      addButtons();
    });
  }).observe(document.documentElement, { childList: true, subtree: true });
})();
