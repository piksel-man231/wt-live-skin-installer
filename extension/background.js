chrome.runtime.onMessage.addListener((message, sender) => {
  if (message?.type !== 'OPEN_INSTALLER' || !/^\d{1,12}$/.test(message.post || '')) return;
  if (!sender.tab || !sender.url) return;
  let source;
  try { source = new URL(sender.url); } catch { return; }
  if (source.origin !== 'https://live.warthunder.com') return;
  chrome.tabs.create({ url: chrome.runtime.getURL(`install.html?post=${message.post}`) });
});
