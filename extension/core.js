globalThis.WtSkinCore = (() => {
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
  async function loadPost(post) {
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
    return { name: data.file.name || `Публикация ${post}`, url: url.href };
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
  async function install(post, directory, skin, progress = () => {}) {
    progress('Скачиваем архив…');
    const response = await fetch(skin.url);
    if (!response.ok) throw new Error('Не удалось скачать архив с WT Live.');
    if (Number(response.headers.get('content-length')) > 250 * 1024 * 1024) throw new Error('Архив больше допустимых 250 МБ.');
    const bytes = new Uint8Array(await response.arrayBuffer());
    if (bytes.length > 250 * 1024 * 1024) throw new Error('Архив больше допустимых 250 МБ.');
    progress('Проверяем и распаковываем архив…');
    const groups = WtSkinArchive.unpack(bytes);
    let done = 0;
    for (const group of groups) {
      progress(`Устанавливаем ${++done} из ${groups.length}: ${group.label}…`);
      const folder = await nextFolder(directory, `wt_live_${post}_${group.label}`.slice(0, 120));
      await writeFiles(folder, group);
    }
    return groups.length;
  }
  return { store, loadPost, install };
})();
