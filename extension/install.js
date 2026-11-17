(() => {
  const post = new URLSearchParams(location.search).get('post');
  const nameEl = document.getElementById('skin-name');
  const folderEl = document.getElementById('folder-label');
  const statusEl = document.getElementById('status');
  const chooseButton = document.getElementById('choose-folder');
  const installButton = document.getElementById('install');
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
    chooseButton.disabled = busy;
  }
  function store(mode, value) {
    return new Promise((resolve, reject) => {
      const request = indexedDB.open('wt-skin-installer', 1);
      request.onupgradeneeded = () => request.result.createObjectStore('settings');
      request.onerror = () => reject(request.error);
      request.onsuccess = () => {
        const db = request.result;
        const tx = db.transaction('settings', mode === 'get' ? 'readonly' : 'readwrite');
        const op = mode === 'get' ? tx.objectStore('settings').get('userSkins') : tx.objectStore('settings').put(value, 'userSkins');
        op.onsuccess = () => resolve(op.result);
        op.onerror = () => reject(op.error);
        tx.oncomplete = () => db.close();
      };
    });
  }
  async function loadPost() {
    if (!/^\d{1,12}$/.test(post || '')) throw new Error('Не найден номер публикации WT Live.');
    const response = await fetch('https://live.warthunder.com/api/posts/get/', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({ lang_group: post, language: 'en' })
    });
    if (!response.ok) throw new Error('WT Live не ответил на запрос публикации.');
    const data = await response.json();
    if (data.type !== 'camouflage' || !data.file?.link) throw new Error('Эта публикация не содержит камуфляж для установки.');
    const url = new URL(data.file.link);
    if (url.protocol !== 'https:' || url.hostname !== 'live.warthunder.com' || !url.pathname.startsWith('/dl/')) {
      throw new Error('WT Live вернул неожиданную ссылку на файл.');
    }
    if (data.file.size > 250 * 1024 * 1024) throw new Error('Архив больше допустимых 250 МБ.');
    skin = { name: data.file.name || `Публикация ${post}`, url: url.href };
    nameEl.textContent = skin.name;
    refresh();
  }
  async function nextFolder(root, base) {
    for (let i = 0; i < 1000; i++) {
      const name = i ? `${base}_${i + 1}` : base;
      try { await root.getDirectoryHandle(name); }
      catch (error) {
        if (error.name === 'NotFoundError') return root.getDirectoryHandle(name, { create: true });
        throw error;
      }
    }
    throw new Error('Не удалось подобрать имя папки для скина.');
  }
  async function writeFiles(root, group) {
    for (const file of group.files) {
      let folder = root;
      for (const part of file.path.slice(0, -1)) folder = await folder.getDirectoryHandle(part, { create: true });
      const handle = await folder.getFileHandle(file.path.at(-1), { create: true });
      const writable = await handle.createWritable();
      try { await writable.write(file.data); await writable.close(); }
      catch (error) { await writable.abort().catch(() => {}); throw error; }
    }
  }
  chooseButton.addEventListener('click', async () => {
    try {
      if (!window.showDirectoryPicker) throw new Error('Этот браузер не поддерживает выбор папки с правом записи.');
      const chosen = await showDirectoryPicker({ mode: 'readwrite', startIn: directory || 'documents' });
      if (chosen.name.toLowerCase() !== 'userskins') throw new Error('Выберите именно папку UserSkins в каталоге War Thunder.');
      directory = chosen;
      await store('put', chosen);
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
      status('Скачиваем архив…');
      const response = await fetch(skin.url);
      if (!response.ok) throw new Error('Не удалось скачать архив с WT Live.');
      const bytes = new Uint8Array(await response.arrayBuffer());
      if (bytes.length > 250 * 1024 * 1024) throw new Error('Архив больше допустимых 250 МБ.');
      status('Проверяем и распаковываем архив…');
      const groups = WtSkinArchive.unpack(bytes);
      let done = 0;
      for (const group of groups) {
        status(`Устанавливаем ${++done} из ${groups.length}: ${group.label}…`);
        const folder = await nextFolder(directory, `wt_live_${post}_${group.label}`.slice(0, 120));
        await writeFiles(folder, group);
      }
      status(`Готово: установлено скинов — ${groups.length}. Обновите список пользовательских камуфляжей в игре.`, 'success');
    } catch (error) { status(error.message || 'Не удалось установить скин.', 'error'); }
    finally { busy = false; refresh(); }
  });
  store('get').then(handle => { if (handle?.kind === 'directory') { directory = handle; refresh(); } }).catch(() => {});
  loadPost().catch(error => { nameEl.textContent = 'Не удалось открыть публикацию'; status(error.message, 'error'); });
})();
