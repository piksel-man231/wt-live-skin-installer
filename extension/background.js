importScripts('vendor/fflate.js', 'archive.js', 'core.js');

function nativeInstall(post) {
  return new Promise(resolve => {
    chrome.runtime.sendNativeMessage('com.wtlive.skin_installer', { post }, reply => {
      const error = chrome.runtime.lastError;
      if (error) resolve({ available: false });
      else resolve({ available: true, reply });
    });
  });
}

async function installOrOpen(post) {
  const native = await nativeInstall(post);
  if (native.available) {
    if (native.reply?.ok) return { installed: native.reply.count };
    return { error: native.reply?.error || 'Помощник не смог установить скин.' };
  }

  // A previously selected folder can be used directly while its write grant survives.
  try {
    const directory = await WtSkinCore.store('get');
    if (directory?.kind === 'directory' && await directory.queryPermission({ mode: 'readwrite' }) === 'granted') {
      const skin = await WtSkinCore.loadPost(post);
      const count = await WtSkinCore.install(post, directory, skin);
      return { installed: count };
    }
  } catch (error) {
    return { error: error.message || 'Не удалось установить скин.' };
  }

  await chrome.tabs.create({ url: chrome.runtime.getURL(`install.html?post=${post}`) });
  return { opened: true };
}

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message?.type !== 'INSTALL_SKIN' || !/^\d{1,12}$/.test(message.post || '')) return;
  if (!sender.tab || !sender.url) return;
  let source;
  try { source = new URL(sender.url); } catch { return; }
  if (source.origin !== 'https://live.warthunder.com') return;
  installOrOpen(message.post).then(sendResponse, error => sendResponse({ error: error.message || 'Не удалось открыть установку.' }));
  return true;
});
