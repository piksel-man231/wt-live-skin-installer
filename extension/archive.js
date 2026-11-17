// Archive planning is separate from browser access so it can be tested with fixture ZIPs.
globalThis.WtSkinArchive = (() => {
  const MAX_FILES = 500;
  const MAX_UNPACKED = 1024 * 1024 * 1024;
  const ALLOWED = /\.(?:blk|dds|tga|png|jpe?g|bmp|webp|tiff?|ktx|txt)$/i;

  function parts(path) {
    const normalized = path.replaceAll('\\', '/');
    const segments = normalized.split('/').filter(Boolean);
    if (!segments.length || normalized.startsWith('/') || segments.some(s => s === '.' || s === '..' || /[\x00-\x1f:]/.test(s))) {
      throw new Error('В архиве обнаружен небезопасный путь.');
    }
    if (segments.length > 8 || segments.some(s => s.length > 180)) {
      throw new Error('Слишком глубокая структура архива.');
    }
    return segments;
  }

  function unpack(bytes) {
    let count = 0;
    let total = 0;
    const files = fflate.unzipSync(bytes, { filter(info) {
      const path = parts(info.name);
      if (info.name.endsWith('/')) return false;
      if (!ALLOWED.test(path.at(-1))) return false;
      count++;
      total += info.originalSize;
      if (count > MAX_FILES || total > MAX_UNPACKED || info.originalSize > 512 * 1024 * 1024) {
        throw new Error('Архив слишком большой для безопасной установки.');
      }
      return true;
    }});
    const entries = Object.entries(files).map(([name, data]) => ({ path: parts(name), data }));
    if (!entries.some(e => /\.blk$/i.test(e.path.at(-1)))) {
      throw new Error('В архиве нет файла .blk — это не архив скина.');
    }

    // Remove common packaging folders (including a bundled UserSkins folder).
    while (entries.every(e => e.path.length > 1 && e.path[0] === entries[0].path[0])) {
      entries.forEach(e => e.path.shift());
    }

    const configs = entries.filter(e => /\.blk$/i.test(e.path.at(-1)));
    return configs.map(config => {
      const parent = config.path.slice(0, -1);
      const assets = entries.filter(e => {
        if (e === config || /\.blk$/i.test(e.path.at(-1))) return false;
        // Files next to the config and in its subfolders belong to this skin.
        // Root-level files are often shared by skins in separate folders.
        const sameTree = parent.every((segment, i) => e.path[i] === segment);
        return sameTree || (parent.length > 0 && e.path.length === 1);
      });
      const prefix = parent.length ? parent.join('_') + '_' : '';
      const label = (prefix + config.path.at(-1).replace(/\.blk$/i, ''))
        .replace(/[^\p{L}\p{N}._-]+/gu, '_').slice(0, 90) || 'skin';
      return {
        label,
        files: [config, ...assets].map(e => ({
          path: e === config ? [e.path.at(-1)] :
            parent.every((segment, i) => e.path[i] === segment) ? e.path.slice(parent.length) : e.path,
          data: e.data
        }))
      };
    });
  }
  return { unpack };
})();
