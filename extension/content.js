(() => {
  const installerUrl = chrome.runtime.getURL('install.html');

  function addButtons() {
    // WT Live inserts post cards and the post lightbox dynamically.
    for (const download of document.querySelectorAll('a.downloadButton, #clb a.downloads[href], .feed_item a.downloads[href]')) {
      if (download.dataset.wtSkinInstaller === '1') continue;
      const container = download.closest('[lang_group]');
      const match = location.pathname.match(/^\/post\/(\d+)(?:\/|$)/);
      const id = container?.getAttribute('lang_group') || match?.[1];
      if (!/^\d+$/.test(id || '')) continue;
      download.dataset.wtSkinInstaller = '1';
      const button = document.createElement('a');
      button.className = 'wt-skin-install';
      button.textContent = 'Установить скин';
      button.href = `${installerUrl}?post=${encodeURIComponent(id)}`;
      button.target = '_blank';
      button.rel = 'noopener noreferrer';
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
