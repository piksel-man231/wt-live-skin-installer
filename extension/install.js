(() => {
  const post = new URLSearchParams(location.search).get('post');
  const nameEl = document.getElementById('skin-name');
  const folderEl = document.getElementById('folder-label');
  const statusEl = document.getElementById('status');
  const chooseButton = document.getElementById('choose-folder');
  const installButton = document.getElementById('install');
  const nativeButton = document.getElementById('native-install');
  let skin;
  let directory;
  let busy = false;

  function status(message, kind = '') {
    statusEl.textContent = message;
    statusEl.className = kind;
  }
  function refresh() {
    folderEl.textContent = directory ? `Выбрана папка: ${directory.name}` : 'Папка UserSkins ещё не выбрана.';
    installButton.disabled = !skin || !directory || busy;
    nativeButton.disabled = !skin || busy;
    chooseButton.disabled = busy;
  }
  async function loadPost() {
    skin = await WtSkinCore.loadPost(post);
    nameEl.textContent = skin.name;
    refresh();
  }
  chooseButton.addEventListener('click', async () => {
    try {
      if (!window.showDirectoryPicker) throw new Error('Этот браузер не поддерживает выбор папки с правом записи.');
      const chosen = await showDirectoryPicker({ mode: 'readwrite', startIn: directory || 'documents' });
      if (chosen.name.toLowerCase() !== 'userskins') throw new Error('Выберите именно папку UserSkins в каталоге War Thunder.');
      directory = chosen;
      await WtSkinCore.store('put', chosen);
      status('Папка сохранена. Теперь можно установить скин.');
      refresh();
    } catch (error) {
      if (error.name !== 'AbortError') status(error.message, 'error');
    }
  });
  installButton.addEventListener('click', async () => {
    busy = true;
    refresh();
    try {
      if ((await directory.requestPermission({ mode: 'readwrite' })) !== 'granted') {
        throw new Error('Браузеру нужен доступ на запись в UserSkins.');
      }
      const count = await WtSkinCore.install(post, directory, skin, status);
      status(`Готово: установлено скинов — ${count}. Обновите список пользовательских камуфляжей в игре.`, 'success');
    } catch (error) { status(error.message || 'Не удалось установить скин.', 'error'); }
    finally { busy = false; refresh(); }
  });
  nativeButton.addEventListener('click', () => {
    if (!skin || busy) return;
    busy = true;
    refresh();
    status('Помощник скачивает и устанавливает скин…');
    chrome.runtime.sendNativeMessage('com.wtlive.skin_installer', { post }, reply => {
      busy = false;
      refresh();
      const error = chrome.runtime.lastError;
      if (error) {
        status('Помощник не найден. Установите его по инструкции ниже и перезапустите браузер.', 'error');
      } else if (!reply?.ok) {
        status(reply?.error || 'Помощник не смог установить скин.', 'error');
      } else {
        status(`Готово: установлено скинов — ${reply.count}. Обновите список камуфляжей в игре.`, 'success');
      }
    });
  });
  WtSkinCore.store('get').then(handle => { if (handle?.kind === 'directory') { directory = handle; refresh(); } }).catch(() => {});
  WtSkinCore.store('put', true, 'setupVisited').catch(() => {});
  loadPost().catch(error => { nameEl.textContent = 'Не удалось открыть публикацию'; status(error.message, 'error'); });
})();
